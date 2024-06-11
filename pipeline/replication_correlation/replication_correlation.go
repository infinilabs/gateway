/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package replication_correlation

import (
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/locker"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/stats"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
	"runtime"
	"strings"
	"sync"
	"time"
)

type Config struct {
	SafetyCommitOffsetPaddingSize int64 `config:"safety_commit_offset_padding_size"` //offset between last and wal
	SafetyCommitIntervalInSeconds int64 `config:"safety_commit_interval_in_seconds"` //time between last and wal
	SafetyCommitRetryTimes        int   `config:"safety_commit_retry_times"`         //retry times
	PartitionSize                 int   `config:"partition_size"`                    //retry times
}

type MessageRecord struct {

	MessageOffset queue.Offset

	RecordOffset    string
	RecordTimestamp string
	recordTime int64
}

type ReplicationCorrectionGroup struct {
	partitionID int

	config              *Config
	PreStageQueueName   string `config:"pre_stage_queue"`
	//FirstStageQueueName string `config:"first_stage_queue"`
	FinalStageQueueName string `config:"final_stage_queue"`

	firstStageRecords sync.Map
	finalStageRecords sync.Map

	lastOffsetInPrepareStage int64 //last offset of prepare stage message
	lastOffsetInFinalStage   int64 //last offset of final stage message

	lastMessageFetchedTimeInAnyStage     time.Time //last time fetched any message
	lastMessageFetchedTimeInPrepareStage time.Time

	totalMessageProcessedInPrepareStage int
	totalMessageProcessedInFinalStage   int

	totalFinishedMessage   int
	totalUnFinishedMessage int

	commitableTimestampInPrepareStage          int64 //last message timestamp in wal - safe_commit_interval
	commitableMessageOffsetInFinalStage        queue.Offset
	commitableMessageOffsetInFirstStage        queue.Offset
	lastTimestampFetchedAnyMessageInFinalStage time.Time
	latestRecordTimestampInPrepareStage        int64
}

type ReplicationCorrectionProcessor struct {
	config *Config
}

func init() {
	pipeline.RegisterProcessorPlugin("replication_correlation", New)
}

func (runner *ReplicationCorrectionProcessor) newGroup(id int) *ReplicationCorrectionGroup {
	suffix := ""

	if runner.config.PartitionSize > 0 {
		suffix = fmt.Sprintf("##%d", id)
	}

	group := ReplicationCorrectionGroup{
		partitionID:         id,
		PreStageQueueName:   "primary_write_ahead_log" + suffix,
		//FirstStageQueueName: "primary_first_commit_log" + suffix,
		FinalStageQueueName: "primary_final_commit_log" + suffix,
	}
	group.config = runner.config
	group.firstStageRecords = sync.Map{}
	group.finalStageRecords = sync.Map{}
	return &group
}

func New(c *config.Config) (pipeline.Processor, error) {
	cfg := Config{
		SafetyCommitOffsetPaddingSize: 10000000,
		SafetyCommitIntervalInSeconds: 120,
		SafetyCommitRetryTimes:        60,
	}

	if err := c.Unpack(&cfg); err != nil {
		log.Error(err)
		return nil, fmt.Errorf("failed to unpack the configuration of flow_runner processor: %s", err)
	}

	if cfg.SafetyCommitIntervalInSeconds <= 0 {
		cfg.SafetyCommitIntervalInSeconds = 120
	}
	if cfg.SafetyCommitRetryTimes <= 0 {
		cfg.SafetyCommitRetryTimes = 60
	}
	if cfg.SafetyCommitOffsetPaddingSize <= 0 {
		cfg.SafetyCommitOffsetPaddingSize = 10000000
	}

	runner := ReplicationCorrectionProcessor{config: &cfg}

	return &runner, nil
}

func (processor *ReplicationCorrectionProcessor) Stop() error {
	return nil
}

func (processor *ReplicationCorrectionProcessor) Name() string {
	return "replication_correlation"
}

func (processor *ReplicationCorrectionGroup) fetchMessages(ctx *pipeline.Context,tag string, consumer queue.ConsumerAPI, handler func(consumer queue.ConsumerAPI, msg []queue.Message) bool, wg *sync.WaitGroup) {
	defer func() {
		if !global.Env().IsDebug {
			if r := recover(); r != nil {
				var v string
				switch r.(type) {
				case error:
					v = r.(error).Error()
				case runtime.Error:
					v = r.(runtime.Error).Error()
				case string:
					v = r.(string)
				}
				if !util.ContainStr(v, "owning this topic") {
					log.Errorf("error in processor [%v], [%v]", processor.Name(), v)
				}
			}
		}
		wg.Done()
	}()

	ctx1 := queue.Context{}
Fetch:

	if ctx.IsCanceled() {
		return
	}

	messages, _, err := consumer.FetchMessages(&ctx1, 5000)
	if err != nil {
		panic(err)
	}

	log.Debugf("get %v messages from queue: %v, %v", len(messages),tag,ctx1.String())

	if len(messages) > 0 {

		processor.lastMessageFetchedTimeInAnyStage = time.Now()

		if !handler(consumer, messages) {
			return
		}
	}

	if len(messages) == 0 {
		time.Sleep(10*time.Second)
	}

	if global.ShuttingDown() {
		return
	}

	if time.Since(processor.lastMessageFetchedTimeInAnyStage) > time.Second*time.Duration(processor.config.SafetyCommitIntervalInSeconds) {
		return
	}

	if ctx.IsCanceled() {
		return
	}

	goto Fetch
}

func (processor *ReplicationCorrectionGroup) cleanup(uID interface{}) {
	//processor.prepareStageBitmap.Delete(uID)
	//processor.firstStageRecords.Delete(uID)
	//processor.finalStageRecords.Delete(uID)
}

var defaultHTTPPool=fasthttp.NewRequestResponsePool("replication_crc")

func parseIDAndOffset(v string) (id, offset,timestamp string) {
	arr := strings.Split(v, "#")
	if len(arr) != 3 {
		panic("invalid message format:" + v)
	}
	return arr[0], arr[1],arr[2]
}

func (processor *ReplicationCorrectionProcessor) Process(ctx *pipeline.Context) error {
	defer func() {
		if !global.Env().IsDebug {
			if r := recover(); r != nil {
				var v string
				switch r.(type) {
				case error:
					v = r.(error).Error()
				case runtime.Error:
					v = r.(runtime.Error).Error()
				case string:
					v = r.(string)
				}
				if !util.ContainStr(v, "already acquired") {
					log.Errorf("error in processor [%v], [%v]", processor.Name(), v)
				}
				ctx.RecordError(fmt.Errorf("replay processor panic: %v", r))
			}
		}
	}()

	wg := sync.WaitGroup{}
	if processor.config.PartitionSize > 0 {
		for i := 0; i < processor.config.PartitionSize; i++ {
			group := processor.newGroup(i)
			log.Debugf("start to process partition %v", i)
			wg.Add(1)
			go group.process(ctx, &wg)
		}
	}
	wg.Wait()

	log.Debug("all partitions are done")

	return nil
}

func (processor *ReplicationCorrectionGroup) process(ctx *pipeline.Context, w *sync.WaitGroup) error {

	defer func() {
		if !global.Env().IsDebug {
			if r := recover(); r != nil {
				var v string
				switch r.(type) {
				case error:
					v = r.(error).Error()
				case runtime.Error:
					v = r.(runtime.Error).Error()
				case string:
					v = r.(string)
				}
				if !util.ContainStr(v, "already acquired") {
					log.Errorf("error in processor [%v], [%v]", processor.Name(), v)
				}
				ctx.RecordError(fmt.Errorf("replay processor panic: %v", r))
			}
		}
		w.Done()
		processor.firstStageRecords = sync.Map{}
		processor.finalStageRecords = sync.Map{}
	}()

	//skip empty queue
	cfg, ok := queue.GetConfigByKey(processor.PreStageQueueName)
	if !ok {
		log.Debugf("empty queue, skip processing")
		return nil
	}
	if queue.LatestOffset(cfg).Position <= 0 {
		log.Debugf("empty queue, skip processing")
		time.Sleep(1 * time.Second)
		return nil
	}

	if ok, _ := locker.Hold("tasks", "replication_correlation", global.Env().SystemConfig.NodeConfig.ID, 180*time.Second, true); !ok {
		log.Infof("replication_correlation already running some where, skip processing")
		time.Sleep(1 * time.Second)
		return nil
	}

	processor.lastTimestampFetchedAnyMessageInFinalStage = time.Now()
	var timestampMessageFetchedInFinalStage int64 = -1

	wg := sync.WaitGroup{}

	//wg.Add(1)
	//firstCommitqConfig, firstCommitConsumerConfig, firstCommitLogConsumer := processor.getConsumer(processor.FirstStageQueueName)
	//defer queue.ReleaseConsumer(firstCommitqConfig, firstCommitConsumerConfig, firstCommitLogConsumer)
	////check first stage commit
	//go processor.fetchMessages(ctx, "first",firstCommitLogConsumer, func(consumer queue.ConsumerAPI, messages []queue.Message) bool {
	//	for _, message := range messages {
	//
	//		if processor.commitable(message) {
	//			processor.commitableMessageOffsetInFirstStage = message.NextOffset
	//		}
	//
	//		v := string(message.Data)
	//
	//		id, offset,timestamp := parseIDAndOffset(v)
	//		processor.firstStageRecords.Store(id, MessageRecord{MessageOffset:message.NextOffset,RecordOffset: offset,RecordTimestamp: timestamp})
	//	}
	//	return true
	//}, &wg)

	wg.Add(1)
	finalCommitqConfig, finalCommitConsumerConfig, finalCommitLogConsumer := processor.getConsumer(processor.FinalStageQueueName)
	defer queue.ReleaseConsumer(finalCommitqConfig, finalCommitConsumerConfig, finalCommitLogConsumer)
	//check second stage commit
	go processor.fetchMessages(ctx,"final", finalCommitLogConsumer, func(consumer queue.ConsumerAPI, messages []queue.Message) bool {

		processor.lastTimestampFetchedAnyMessageInFinalStage = time.Now()

		for _, message := range messages {

			//commit log less then wal log more than 60s, then it should be safety to commit
			if processor.commitable(message) {
				processor.commitableMessageOffsetInFinalStage = message.NextOffset
			}

			processor.totalMessageProcessedInFinalStage++
			timestampMessageFetchedInFinalStage = message.Timestamp

			processor.lastOffsetInFinalStage = message.NextOffset.Position

			v := string(message.Data)
			id, offset,timestamp := parseIDAndOffset(v)
			processor.finalStageRecords.Store(id, MessageRecord{MessageOffset:message.NextOffset,RecordOffset: offset,RecordTimestamp: timestamp})
		}
		log.Debugf("final stage message count: %v, map:%v", processor.totalMessageProcessedInFinalStage, util.GetSyncMapSize(&processor.finalStageRecords))
		return true
	}, &wg)

	time.Sleep(10 * time.Second)

	wg.Add(1)
	walCommitqConfig, walCommitConsumerConfig, WALConsumer := processor.getConsumer(processor.PreStageQueueName)
	defer queue.ReleaseConsumer(walCommitqConfig, walCommitConsumerConfig, WALConsumer)
	//fetch the message from the wal queue
	go processor.fetchMessages(ctx, "wal",WALConsumer, func(consumer queue.ConsumerAPI, messages []queue.Message) bool {
		processor.lastMessageFetchedTimeInPrepareStage = time.Now()
		var lastCommitableMessageOffset queue.Offset
		defer func() {
			if !global.Env().IsDebug {
				if r := recover(); r != nil {
					var v string
					switch r.(type) {
					case error:
						v = r.(error).Error()
					case runtime.Error:
						v = r.(runtime.Error).Error()
					case string:
						v = r.(string)
					}
					if !util.ContainStr(v, "owning this topic") {
						log.Errorf("error in processor [%v], [%v]", processor.Name(), v)
					}
				}
			}

			if WALConsumer != nil && lastCommitableMessageOffset.Position > 0 {
				WALConsumer.CommitOffset(lastCommitableMessageOffset)
			}

		}()

		for _, message := range messages {
			processor.totalMessageProcessedInPrepareStage++
			processor.lastOffsetInPrepareStage = message.NextOffset.Position
			if global.ShuttingDown() {
				return false
			}

			req := defaultHTTPPool.AcquireRequest()
			err := req.Decode(message.Data)
			if err != nil {
				panic(err)
			}
			idByte := req.Header.Peek("X-Replicated-ID")
			if idByte == nil || len(idByte) == 0 {
				panic("invalid id")
			}
			uID := string(idByte)

			timeByte := req.Header.Peek("X-Replicated-Timestamp")
			if timeByte == nil || len(timeByte) == 0 {
				panic("invalid timestamp")
			}

			tn, err := util.ToInt64(string(timeByte))
			if err != nil {
				panic(err)
			}
			//timestamp:= util.FromUnixTimestamp(tn)
			processor.latestRecordTimestampInPrepareStage = tn

			defaultHTTPPool.ReleaseRequest(req)

			//update commit offset
			lastCommitableMessageOffset = message.NextOffset
			processor.commitableTimestampInPrepareStage = message.Timestamp - processor.config.SafetyCommitIntervalInSeconds

			retry_times := 0

		RETRY:

			if global.ShuttingDown() {
				return false
			}

			if time.Since(processor.lastMessageFetchedTimeInAnyStage) > time.Second*time.Duration(processor.config.SafetyCommitIntervalInSeconds) {
				return false
			}

			//check final stage
			if _, ok := processor.finalStageRecords.Load(uID); ok {
				//valid request, cleanup
				processor.cleanup(uID)
				processor.totalFinishedMessage++
			} else {
				hit := false
				if (processor.lastOffsetInFinalStage - processor.lastOffsetInPrepareStage) > processor.config.SafetyCommitOffsetPaddingSize {
					stats.Increment("replication_crc", "safe_commit_offset_padding_size")
					hit = true
				}

				if retry_times > processor.config.SafetyCommitRetryTimes {
					stats.Increment("replication_crc", fmt.Sprintf("retry_times_exceed_%v",processor.config.SafetyCommitRetryTimes))
					hit = true
				}

				if time.Since(processor.lastTimestampFetchedAnyMessageInFinalStage) > time.Second*time.Duration(processor.config.SafetyCommitIntervalInSeconds) && (processor.lastTimestampFetchedAnyMessageInFinalStage.Unix()-message.Timestamp > processor.config.SafetyCommitIntervalInSeconds) {
					stats.Increment("replication_crc", "no_message_fetched_in_final_stage_more_than_safety_interval")
					//log.Error("no message fetched in final stage more than 120s, ", time.Since(processor.lastTimestampFetchedAnyMessageInFinalStage))
					hit = true
				}

				if timestampMessageFetchedInFinalStage > 0 && (timestampMessageFetchedInFinalStage-processor.config.SafetyCommitIntervalInSeconds) > message.Timestamp {
					stats.Increment("replication_crc", "message_fetched_in_final_stage_more_than_safety_interval")
					hit = true
				}

				//if last commit time is more than 30 seconds, compare to wal or now, then this message maybe lost
				if hit { //too long no message returned, maybe finished

					var errLog string
					//check if in first stage
					if _, ok := processor.firstStageRecords.Load(uID); ok {
						errLog = fmt.Sprintf("request %v, offset: %v, %v in first stage but not in final stage", uID, message.NextOffset, message.Timestamp)
					} else {
						errLog = fmt.Sprintf("request %v, offset: %v, %v exists in wal but not in any stage", uID, message.NextOffset, message.Timestamp)
					}

					err := queue.Push(queue.GetOrInitConfig("replicate_failure_log"), []byte(errLog))
					if err != nil {
						panic(err)
					}

					err = queue.Push(queue.GetOrInitConfig("primary-failure"), message.Data)
					if err != nil {
						panic(err)
					}

					processor.totalUnFinishedMessage++
					processor.cleanup(uID)

					retry_times = 0
				} else {
					if global.Env().IsDebug {
						log.Infof("request %v, offset: %v,"+
							" retry_times: %v, docs_in_first_stage: %v, docs_in_final_stage: %v, last_final_offset: %v, last_wal_offset: %v",
							uID, message.NextOffset,
							retry_times, util.GetSyncMapSize(&processor.firstStageRecords), util.GetSyncMapSize(&processor.finalStageRecords), processor.lastOffsetInFinalStage, processor.lastOffsetInPrepareStage)
					}
					time.Sleep(1 * time.Second)
					retry_times++

					//retry
					goto RETRY
				}
			}

		}
		return true
	}, &wg)

	//commit first and final stage in side way
	wg.Add(1)
	go func(ctx *pipeline.Context) {
		defer func() {
			if !global.Env().IsDebug {
				if r := recover(); r != nil {
					var v string
					switch r.(type) {
					case error:
						v = r.(error).Error()
					case runtime.Error:
						v = r.(runtime.Error).Error()
					case string:
						v = r.(string)
					}
					log.Errorf("panic in commit first and final stage: %v", v)
				}
			}
			wg.Done()
		}()
		var commitAnywayWaitInSeconds=time.Duration(processor.config.SafetyCommitIntervalInSeconds)

		//first
		var lastFirstCommitAbleMessageRecord MessageRecord
		//var lastFirstCommit time.Time
		//var needCommitFirstStage bool

		//final
		var lastFinalCommitAbleMessageRecord MessageRecord
		var lastFinalCommit time.Time
		var needCommitFinalStage bool

		for{

			if global.ShuttingDown() {
				return
			}

			if ctx.IsCanceled() {
				return
			}

			if time.Since(processor.lastMessageFetchedTimeInAnyStage) > time.Second*time.Duration(processor.config.SafetyCommitIntervalInSeconds) {
				return
			}

			time.Sleep(10*time.Second)

			//cleanup first log
			processor.firstStageRecords.Range(func(key, value interface{}) bool {
				x := value.(MessageRecord)
				msgTime,err:=util.ToInt64(x.RecordTimestamp)
				if err!=nil{
					panic(err)
				}

				var commitAnyway bool
				if !(msgTime > 0 && processor.latestRecordTimestampInPrepareStage>0){
					walHasLag:=queue.ConsumerHasLag(walCommitqConfig,walCommitConsumerConfig)
					if walHasLag{
						return true
					}else{
						commitAnyway=true
					}
				}

				timegap:=processor.latestRecordTimestampInPrepareStage-msgTime
				if processor.totalMessageProcessedInPrepareStage > 0 && time.Since(processor.lastMessageFetchedTimeInPrepareStage) > time.Second*commitAnywayWaitInSeconds{
					if time.Since(processor.lastMessageFetchedTimeInAnyStage) > time.Second*commitAnywayWaitInSeconds{
						commitAnyway=true
					}
				}

				if commitAnyway||timegap>processor.config.SafetyCommitIntervalInSeconds{
					//update to latest committable message
					if (commitAnyway&&x.MessageOffset.LatestThan(lastFirstCommitAbleMessageRecord.MessageOffset))||
						(msgTime> lastFirstCommitAbleMessageRecord.recordTime&&x.MessageOffset.LatestThan(lastFirstCommitAbleMessageRecord.MessageOffset)){
						x.recordTime=msgTime
						lastFirstCommitAbleMessageRecord=x
						processor.commitableMessageOffsetInFirstStage=x.MessageOffset
						//needCommitFirstStage=true
						log.Debug("update first commit:",x.MessageOffset)
					}
				}


				//if needCommitFirstStage{
				//	if time.Since(lastFirstCommit)>time.Second*10{
				//		log.Debug("committing first offset:",processor.commitableMessageOffsetInFirstStage)
				//		firstCommitLogConsumer.CommitOffset(processor.commitableMessageOffsetInFirstStage)
				//		lastFirstCommit=time.Now()
				//		needCommitFirstStage=false
				//		timegap1:=msgTime- lastFirstCommitAbleMessageRecord.recordTime
				//		log.Trace(x.RecordTimestamp,",",x.RecordOffset,",time_gap: ",timegap,"s, ",timegap1,"s, record:",msgTime," vs latest:",processor.latestRecordTimestampInPrepareStage,", updating to commit:",x.MessageOffset,lastFirstCommitAbleMessageRecord,",",processor.config.SafetyCommitIntervalInSeconds)
				//	}
				//}
				return true
			})


			//cleanup final log
			processor.finalStageRecords.Range(func(key, value interface{}) bool {
				x := value.(MessageRecord)
				msgTime,err:=util.ToInt64(x.RecordTimestamp)
				if err!=nil{
					panic(err)
				}

				var commitAnyway bool
				if !(msgTime > 0 && processor.latestRecordTimestampInPrepareStage>0){
					walHasLag:=queue.ConsumerHasLag(walCommitqConfig,walCommitConsumerConfig)
					if walHasLag{
						return true
					}else{
						commitAnyway=true
					}
				}

				timegap:=processor.latestRecordTimestampInPrepareStage-msgTime
				if processor.totalMessageProcessedInPrepareStage > 0 && time.Since(processor.lastMessageFetchedTimeInPrepareStage) > time.Second*commitAnywayWaitInSeconds{
					if time.Since(processor.lastMessageFetchedTimeInAnyStage) > time.Second*commitAnywayWaitInSeconds{
						commitAnyway=true
					}
				}

				if commitAnyway||timegap>processor.config.SafetyCommitIntervalInSeconds{
					//update to latest committable message
					if (commitAnyway&&x.MessageOffset.LatestThan(lastFinalCommitAbleMessageRecord.MessageOffset))||
						(msgTime> lastFinalCommitAbleMessageRecord.recordTime&&x.MessageOffset.LatestThan(lastFinalCommitAbleMessageRecord.MessageOffset)){
						x.recordTime=msgTime
						lastFinalCommitAbleMessageRecord=x
						processor.commitableMessageOffsetInFinalStage=x.MessageOffset
						needCommitFinalStage=true
						log.Debug("update final commit:",x.MessageOffset)
					}
				}


				if needCommitFinalStage{
					if time.Since(lastFinalCommit)>time.Second*10{
						log.Debug("committing final offset:",processor.commitableMessageOffsetInFinalStage)
						finalCommitLogConsumer.CommitOffset(processor.commitableMessageOffsetInFinalStage)
						lastFinalCommit=time.Now()
						needCommitFinalStage=false
						timegap1:=msgTime- lastFinalCommitAbleMessageRecord.recordTime
						log.Trace(x.RecordTimestamp,",",x.RecordOffset,",time_gap: ",timegap,"s, ",timegap1,"s, record:",msgTime," vs latest:",processor.latestRecordTimestampInPrepareStage,", updating to commit:",x.MessageOffset,lastFinalCommitAbleMessageRecord,",",processor.config.SafetyCommitIntervalInSeconds)
					}
				}
				return true
			})

		}

	}(ctx)

	wg.Wait()

	//wal done, and no more message in other queue as well, then commit
	//cleanup
	if !ctx.IsCanceled() && !global.ShuttingDown() && time.Since(processor.lastMessageFetchedTimeInPrepareStage) > time.Second*60 && time.Since(processor.lastMessageFetchedTimeInAnyStage) > time.Second*60 {

		//if processor.commitableMessageOffsetInFirstStage.Position > 0 {
		//	firstCommitLogConsumer.CommitOffset(processor.commitableMessageOffsetInFirstStage)
		//}

		if processor.commitableMessageOffsetInFinalStage.Position > 0 {
			finalCommitLogConsumer.CommitOffset(processor.commitableMessageOffsetInFinalStage)
		}
	}

	//log.Infof("#%v finished correlation, prepare:%v, finished:%v, unfinished:%v, first_map:%v, final_map:%v, prepare_idle:%v",
	//	processor.partitionID,
	//	processor.totalMessageProcessedInPrepareStage,
	//	processor.totalFinishedMessage,
	//	processor.totalUnFinishedMessage,
	//	util.GetSyncMapSize(&processor.firstStageRecords),
	//	util.GetSyncMapSize(&processor.finalStageRecords),
	//	time.Since(processor.lastMessageFetchedTimeInPrepareStage))

	return nil
}

func (processor *ReplicationCorrectionGroup) getConsumer(queueName string) (*queue.QueueConfig, *queue.ConsumerConfig, queue.ConsumerAPI) {
	qConfig := queue.GetOrInitConfig(queueName)
	cConfig := queue.GetOrInitConsumerConfig(qConfig.ID, "crc", "name1")
	consumer, err := queue.AcquireConsumer(qConfig,
		cConfig,"worker_id")
	if err != nil {
		panic(err)
	}
	return qConfig, cConfig, consumer
}

func (processor *ReplicationCorrectionGroup) commitable(message queue.Message) bool {
	//old message than wal, or wal fetched but idle for 30s
	return processor.commitableTimestampInPrepareStage > 0 && message.Timestamp > 0 && message.Timestamp < processor.commitableTimestampInPrepareStage ||
		(processor.totalMessageProcessedInPrepareStage > 0 && time.Since(processor.lastMessageFetchedTimeInPrepareStage) > time.Second*time.Duration(processor.config.SafetyCommitIntervalInSeconds))
}

func (processor *ReplicationCorrectionGroup) Name() string {
	return processor.PreStageQueueName
}
