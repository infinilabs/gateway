/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package replication_correlation

import (
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/global"
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
}

type ReplicationCorrectionProcessor struct {
	config *Config

	prepareStageBitmap sync.Map
	firstStageBitmap   sync.Map
	finalStageRecords  sync.Map

	lastOffsetInPrepareStage int64 //last offset of prepare stage message
	lastOffsetInFinalStage   int64 //last offset of final stage message

	lastMessageFetchedTimeInAnyStage     time.Time //last time fetched any message
	lastMessageFetchedTimeInPrepareStage time.Time

	totalMessageProcessedInPrepareStage int
	totalMessageProcessedInFinalStage   int

	totalFinishedMessage   int
	totalUnFinishedMessage int

	commitableTimestampInPrepareStage   int64 //last message timestamp in wal -60s
	commitableMessageOffsetInFinalStage queue.Offset
	commitableMessageOffsetInFirstStage queue.Offset
}

func init() {
	pipeline.RegisterProcessorPlugin("replication_correlation", New)
}

func New(c *config.Config) (pipeline.Processor, error) {
	cfg := Config{
		SafetyCommitOffsetPaddingSize: 10000,
		SafetyCommitIntervalInSeconds: 60,
		SafetyCommitRetryTimes:        60,
	}

	if err := c.Unpack(&cfg); err != nil {
		log.Error(err)
		return nil, fmt.Errorf("failed to unpack the configuration of flow_runner processor: %s", err)
	}

	runner := ReplicationCorrectionProcessor{config: &cfg}

	if cfg.SafetyCommitIntervalInSeconds <= 0 {
		cfg.SafetyCommitIntervalInSeconds = 60
	}
	if cfg.SafetyCommitRetryTimes <= 0 {
		cfg.SafetyCommitIntervalInSeconds = 60
	}
	if cfg.SafetyCommitOffsetPaddingSize <= 0 {
		cfg.SafetyCommitOffsetPaddingSize = 10000
	}

	runner.firstStageBitmap = sync.Map{}
	runner.prepareStageBitmap = sync.Map{}
	runner.finalStageRecords = sync.Map{}

	return &runner, nil
}

func (processor *ReplicationCorrectionProcessor) Stop() error {
	return nil
}

func (processor *ReplicationCorrectionProcessor) Name() string {
	return "replication_correlation"
}

func (processor *ReplicationCorrectionProcessor) fetchMessages(ctx *pipeline.Context,consumer queue.ConsumerAPI, handler func(consumer queue.ConsumerAPI, msg []queue.Message) bool, wg *sync.WaitGroup) {
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

	if ctx.IsCanceled(){
		return
	}

	messages, _, err := consumer.FetchMessages(&ctx1, 5000)
	if err != nil {
		panic(err)
	}

	if len(messages) > 0 {

		processor.lastMessageFetchedTimeInAnyStage = time.Now()

		if !handler(consumer, messages) {
			return
		}
	}

	if len(messages) == 0 {
		return
	}

	if global.ShuttingDown() {
		return
	}

	if ctx.IsCanceled(){
		return
	}

	goto Fetch
}

func (processor *ReplicationCorrectionProcessor) cleanup(uID interface{}) {
	processor.prepareStageBitmap.Delete(uID)
	processor.firstStageBitmap.Delete(uID)
	processor.finalStageRecords.Delete(uID)
}

func parseIDAndOffset(v string) (id, offset string) {
	arr := strings.Split(v, "#")
	if len(arr) != 2 {
		panic("invalid message format:" + v)
	}
	return arr[0], arr[1]
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
				log.Errorf("error in processor [%v], [%v]", processor.Name(), v)
				ctx.RecordError(fmt.Errorf("replay processor panic: %v", r))
			}
		}
	}()

	lastTimestampFetchedAnyMessageInFinalStage := time.Now()
	var timestampMessageFetchedInFinalStage int64 = -1

	wg := sync.WaitGroup{}

	wg.Add(1)
	firstCommitqConfig,firstCommitConsumerConfig,firstCommitLogConsumer := processor.getConsumer("primary_first_commit_log")
	defer queue.ReleaseConsumer(firstCommitqConfig, firstCommitConsumerConfig,firstCommitLogConsumer)
	//check first stage commit
	go processor.fetchMessages(ctx,firstCommitLogConsumer, func(consumer queue.ConsumerAPI, messages []queue.Message) bool {
		for _, message := range messages {

			if processor.commitable(message) {
				processor.commitableMessageOffsetInFirstStage = message.NextOffset
			}

			v := string(message.Data)

			id, _ := parseIDAndOffset(v)

			processor.firstStageBitmap.Store(id, message.Offset.String())
		}
		return true
	}, &wg)

	wg.Add(1)
	finalCommitqConfig,finalCommitConsumerConfig,finalCommitLogConsumer := processor.getConsumer("primary_final_commit_log")
	defer queue.ReleaseConsumer(finalCommitqConfig, finalCommitConsumerConfig,finalCommitLogConsumer)
	//check second stage commit
	go processor.fetchMessages(ctx,finalCommitLogConsumer, func(consumer queue.ConsumerAPI, messages []queue.Message) bool {

		lastTimestampFetchedAnyMessageInFinalStage = time.Now()

		for _, message := range messages {

			//commit log less then wal log more than 60s, then it should be safety to commit
			if processor.commitable(message) {
				processor.commitableMessageOffsetInFinalStage = message.NextOffset
			}

			processor.totalMessageProcessedInFinalStage++
			timestampMessageFetchedInFinalStage = message.Timestamp

			processor.lastOffsetInFinalStage = message.Offset.Position

			v := string(message.Data)
			id, _ := parseIDAndOffset(v)
			processor.finalStageRecords.Store(id, message.Offset.String())
		}
		log.Debugf("final stage message count: %v, map:%v", processor.totalMessageProcessedInFinalStage, util.GetSyncMapSize(&processor.finalStageRecords))
		return true
	}, &wg)

	time.Sleep(10 * time.Second)

	wg.Add(1)
	walCommitqConfig, walCommitConsumerConfig,WALConsumer := processor.getConsumer("primary_write_ahead_log")
	defer queue.ReleaseConsumer(walCommitqConfig, walCommitConsumerConfig,WALConsumer)
	//fetch the message from the wal queue
	go processor.fetchMessages(ctx,WALConsumer, func(consumer queue.ConsumerAPI, messages []queue.Message) bool {
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
				if processor.commitableMessageOffsetInFirstStage.Position > 0 {
					firstCommitLogConsumer.CommitOffset(processor.commitableMessageOffsetInFirstStage)
				}
				if processor.commitableMessageOffsetInFinalStage.Position > 0 {
					finalCommitLogConsumer.CommitOffset(processor.commitableMessageOffsetInFinalStage)
				}
			}

		}()

		for _, message := range messages {
			processor.totalMessageProcessedInPrepareStage++
			processor.lastOffsetInPrepareStage = message.Offset.Position
			if global.ShuttingDown() {
				return false
			}

			req := fasthttp.AcquireRequest()
			//log.Errorf("message, %v %v, %v",message.Offset, util.ByteSize(uint64(len(message.Data))),string(message.Data))
			err := req.Decode(message.Data)
			if err != nil {
				panic(err)
			}
			idByte := req.Header.Peek("X-Replicated-ID")
			if idByte == nil || len(idByte) == 0 {
				panic("invalid id")
			}

			fasthttp.ReleaseRequest(req)

			uID := string(idByte)

			//update commit offset
			lastCommitableMessageOffset = message.NextOffset
			processor.commitableTimestampInPrepareStage = message.Timestamp - processor.config.SafetyCommitIntervalInSeconds

			retry_times := 0

		RETRY:
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
					stats.Increment("replication_crc", "retry_times_exceed_10")
					hit = true
				}

				if time.Since(processor.lastMessageFetchedTimeInAnyStage) > time.Second*30 {
					stats.Increment("replication_crc", "no_message_fetched_in_any_stage_more_than_30s")
					hit = true
				}

				if time.Since(lastTimestampFetchedAnyMessageInFinalStage) > time.Second*30 {
					stats.Increment("replication_crc", "no_message_fetched_in_final_stage_more_than_30s")
					hit = true
				}

				if timestampMessageFetchedInFinalStage > 0 && (timestampMessageFetchedInFinalStage-60) > message.Timestamp {
					stats.Increment("replication_crc", "message_fetched_in_final_stage_more_than_60s")
					hit = true
				}

				//if last commit time is more than 30 seconds, compare to wal or now, then this message maybe lost
				if hit { //too long no message returned, maybe finished

					var errLog string
					//check if in first stage
					if _,ok:=processor.firstStageBitmap.Load(uID);ok {
						errLog = fmt.Sprintf("request %v, offset: %v, %v in first stage but not in final stage", uID, message.Offset, message.Timestamp)
					} else {
						errLog = fmt.Sprintf("request %v, offset: %v, %v exists in wal but not in any stage", uID, message.Offset, message.Timestamp)
					}

					queue.Push(queue.GetOrInitConfig("replicate_failure_log"), []byte(errLog))

					err := queue.Push(queue.GetOrInitConfig("primary-failure"), message.Data)
					if err != nil {
						panic(err)
					}

					processor.totalUnFinishedMessage++
					processor.cleanup(uID)

					retry_times = 0
				} else {
					if global.Env().IsDebug {
						log.Debugf("request %v, offset: %v,"+
							" retry_times: %v, docs_in_wal_stage: %v,docs_in_final_stage: %v, last_offset: %v, last_wal_offset: %v, gap: %v",
							uID, message.Offset,
							retry_times, util.GetSyncMapSize(&processor.prepareStageBitmap), util.GetSyncMapSize(&processor.finalStageRecords), processor.lastOffsetInFinalStage, processor.lastOffsetInPrepareStage)
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

	wg.Wait()

	//cleanup
	if time.Since(processor.lastMessageFetchedTimeInPrepareStage) > time.Second*60 {

		if processor.commitableMessageOffsetInFirstStage.Position > 0 {
			firstCommitLogConsumer.CommitOffset(processor.commitableMessageOffsetInFirstStage)
		}

		if processor.commitableMessageOffsetInFinalStage.Position > 0 {
			finalCommitLogConsumer.CommitOffset(processor.commitableMessageOffsetInFinalStage)
		}
	}

	log.Infof("finished replication correlation, total:%v,finished:%v,unfinished:%v, final: %v, final_records:%v, first:%v, idle:%v",
		processor.totalMessageProcessedInPrepareStage, processor.totalFinishedMessage, processor.totalUnFinishedMessage,
		processor.totalMessageProcessedInFinalStage,
		util.GetSyncMapSize(&processor.finalStageRecords),
		util.GetSyncMapSize(&processor.firstStageBitmap),
		time.Since(processor.lastMessageFetchedTimeInPrepareStage))

	return nil
}

func (processor *ReplicationCorrectionProcessor) getConsumer(queueName string) (*queue.QueueConfig,*queue.ConsumerConfig,queue.ConsumerAPI) {
	qConfig := queue.GetOrInitConfig(queueName)
	cConfig := queue.GetOrInitConsumerConfig(qConfig.ID, "crc", "name1")
	consumer, err := queue.AcquireConsumer(qConfig,
		cConfig)
	if err != nil {
		panic(err)
	}
	return qConfig,cConfig,consumer
}

func (processor *ReplicationCorrectionProcessor) commitable(message queue.Message) bool {
	//old message than wal, or wal fetched but idle for 30s
	return processor.commitableTimestampInPrepareStage > 0 && message.Timestamp > 0 && message.Timestamp < processor.commitableTimestampInPrepareStage ||
		(processor.totalMessageProcessedInPrepareStage > 0 && time.Since(processor.lastMessageFetchedTimeInPrepareStage) > time.Second*30)
}
