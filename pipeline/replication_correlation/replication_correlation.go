/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package replication_correlation

import (
	"fmt"
	log "github.com/cihub/seelog"
	"github.com/emirpasic/gods/sets/hashset"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/param"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
	"runtime"
	"strings"
	"sync"
	"time"
)

type Config struct {
	WALMessageField         param.ParaKey `config:"wal_message_field"`
	FirstStageMessageField  param.ParaKey `config:"first_stage_message_field"`
	FinalStageMessageField  param.ParaKey `config:"final_stage_message_field"`
	WALMessageCheckingLimit uint64        `config:"wal_message_checking_limit"`
	SafetyCheckSize         int64         `config:"safety_check_size"`
}

type ReplicationCorrectionProcessor struct {
	config *Config
	walBitmap        *hashset.Set
	preStageBitmap   *hashset.Set
	firstStageBitmap *hashset.Set

	finalRecords sync.Map

	requestMap             sync.Map
	lastOffsetInFirstStage int64     //last offset of first stage message
	lastOffsetInFinalStage int64     //last offset of final stage message
	lastOffsetInWAL        int64     //last offset of final stage message
	lastFetchedMessage     time.Time //last time fetched any message

	totalWALMessageProcessed         int
	finishedMessage                  int
	unfinishedMessage                int
	finalMessageCount                int
	lastWALCommitableTimestamp       int64 //last message timestamp in wal -60s
	lastFinalCommitableMessageOffset queue.Offset
	lastFirstCommitableMessageOffset queue.Offset
	lastPreCommitableMessageOffset   queue.Offset
	timestampMessageFetchedInWAL     time.Time
}

func init() {
	pipeline.RegisterProcessorPlugin("replication_correlation", New)
}

func New(c *config.Config) (pipeline.Processor, error) {
	cfg := Config{
		WALMessageField:         "wal_messages",
		FirstStageMessageField:  "first_stage_messages",
		FinalStageMessageField:  "final_stage_messages",
		WALMessageCheckingLimit: 5000,
		SafetyCheckSize:         5000,
	}

	if err := c.Unpack(&cfg); err != nil {
		log.Error(err)
		return nil, fmt.Errorf("failed to unpack the configuration of flow_runner processor: %s", err)
	}

	runner := ReplicationCorrectionProcessor{config: &cfg}

	runner.preStageBitmap = hashset.New()   //roaring.NewBitmap()
	runner.firstStageBitmap = hashset.New() //roaring.NewBitmap()
	runner.walBitmap = hashset.New()        //roaring.NewBitmap()
	runner.requestMap = sync.Map{}
	runner.finalRecords = sync.Map{}

	return &runner, nil
}

func (processor *ReplicationCorrectionProcessor) Stop() error {
	return nil
}

func (processor *ReplicationCorrectionProcessor) Name() string {
	return "replication_correlation"
}

func (processor *ReplicationCorrectionProcessor) fetchMessages(consumer queue.ConsumerAPI, handler func(consumer queue.ConsumerAPI, msg []queue.Message) bool, wg *sync.WaitGroup) {
	//var offset queue.Offset

	//var skipCommit bool = true

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
				//skipCommit = true
			}
		}
		////TODO, only commit align to WAL message offset
		//if !skipCommit && offset.Position > 0 {
		//	if consumer != nil {
		//		consumer.CommitOffset(offset)
		//	}
		//}
		wg.Done()
		//log.Errorf("finished process %v", queueName)
	}()

	ctx1 := queue.Context{}
Fetch:
	messages, _, err := consumer.FetchMessages(&ctx1, 5000)
	if err != nil {
		panic(err)
	}

	//log.Infof("get %v messages from queue: %v", len(messages), queueName)

	if len(messages) > 0 {

		processor.lastFetchedMessage = time.Now()

		if !handler(consumer, messages) {
			return
		}
		//
		//lstMsg := messages[len(messages)-1]
		//msgTimestamp := util.FromUnixTimestamp(lstMsg.Timestamp)
		//msgTimestamp.Add(-60 * time.Second)
		//if processor.lastWALCommitableTimestamp > 0 && msgTimestamp.Unix() < processor.lastWALCommitableTimestamp {
		//	log.Error("committable timestamp is less than last committable timestamp:", processor.lastWALCommitableTimestamp, ""+
		//		",", msgTimestamp.Unix())
		//	skipCommit = false
		//	offset = lstMsg.NextOffset
		//} else {
		//	skipCommit = true
		//}
		//err := consumer.CommitOffset(messages[len(messages)-1].NextOffset)
		////log.Error("commit offset:", qConfig.Name, ",", cConfig.ID, ",", ctx1.NextOffset,",",err)
		//if err != nil {
		//	panic(err)
		//}
	}

	if len(messages) == 0 {
		return
	}

	if global.ShuttingDown() {
		return
	}

	goto Fetch
}

func (processor *ReplicationCorrectionProcessor) cleanup(uID interface{}) {
	processor.walBitmap.Remove(uID)
	processor.requestMap.Delete(uID)
	processor.firstStageBitmap.Remove(uID)
	processor.finalRecords.Delete(uID)
}

func parseIDAndOffset(v string) (id, offset string) {
	arr := strings.Split(v, "#")
	if len(arr) != 2 {
		panic("invalid message format:" + v)
	}
	return arr[0], arr[1]
}

func (processor *ReplicationCorrectionProcessor) Process(ctx *pipeline.Context) error {

	//ok, _ := locker.Hold("tasks", "replication_correlation", global.Env().SystemConfig.NodeConfig.ID, time.Duration(60)*time.Second, true)
	//if !ok {
	//	log.Infof("another replication_correlation processor is running, skip this time")
	//}

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
	//timestampMessageFetchedInWAL := time.Now()
	//messagesInWAL := 0

	var timestampMessageFetchedInFinalStage int64 = -1

	wg := sync.WaitGroup{}
	//fetch messages within time window

	//endTime := time.Now().Add(30 * time.Second * -1).Unix()
	endTime := time.Now().Unix()
	endTimeForRestCommit := time.Now().Unix()
	var lastOffsetInPreCommit queue.Offset
	numOfPreMessages := 0
	wg.Add(1)

	preCommitLogConsumer := processor.getConsumer("primary_pre_commit_log")
	//check first stage commit
	go processor.fetchMessages(preCommitLogConsumer, func(consumer queue.ConsumerAPI, messages []queue.Message) bool {
		for _, message := range messages {

			if processor.commitable(message) {

				//log.Error("committable timestamp is less than last committable timestamp:", processor.lastWALCommitableTimestamp, ""+
				//	",", message.Timestamp)

				processor.lastPreCommitableMessageOffset = message.NextOffset
			}

			numOfPreMessages++
			if message.Timestamp >= endTime {
				//return false
			}

			lastOffsetInPreCommit = message.Offset

			v := string(message.Data)
			id, _ := parseIDAndOffset(v)

			processor.preStageBitmap.Add(id)

		}
		return true
	}, &wg)

	wg.Add(1)
	firstCommitLogConsumer := processor.getConsumer("primary_first_commit_log")
	//check first stage commit
	go processor.fetchMessages(firstCommitLogConsumer, func(consumer queue.ConsumerAPI, messages []queue.Message) bool {
		for _, message := range messages {

			if processor.commitable(message)  {
				processor.lastFirstCommitableMessageOffset = message.NextOffset
			}

			if message.Timestamp > endTimeForRestCommit {
				//return false
			}

			processor.lastOffsetInFirstStage = message.Offset.Position

			v := string(message.Data)

			id, _ := parseIDAndOffset(v)

			//arr := strings.Split(v, "#")
			//if len(arr) != 2 {
			//	panic("invalid message format:" + v)
			//}
			//id, er := util.ToInt64(arr[0])
			//if er != nil {
			//	panic("invalid id:" + arr[0] + ",error:" + er.Error())
			//}
			//offset := arr[1]
			//log.Error(id, ",", offset, ",size:", util.ByteSize(processor.walBitmap.GetSizeInBytes()))
			processor.firstStageBitmap.Add((id))
		}
		return true
	}, &wg)

	wg.Add(1)
	finalCommitLogConsumer := processor.getConsumer("primary_final_commit_log")
	//check second stage commit
	go processor.fetchMessages(finalCommitLogConsumer, func(consumer queue.ConsumerAPI, messages []queue.Message) bool {

		lastTimestampFetchedAnyMessageInFinalStage = time.Now()

		for _, message := range messages {

			//commit log less then wal log more than 60s, then it should be safety to commit
			if processor.commitable(message)  {
				processor.lastFinalCommitableMessageOffset = message.NextOffset
			}

			processor.finalMessageCount++
			timestampMessageFetchedInFinalStage = message.Timestamp
			if message.Timestamp > endTimeForRestCommit {
				//return false
			}

			processor.lastOffsetInFinalStage = message.Offset.Position

			v := string(message.Data)
			id, _ := parseIDAndOffset(v)
			//offset := arr[1]
			//log.Error(id, ",", message.Offset.String(), ",", message.Timestamp)
			//if processor.finalStageBitmap.Contains(id) {
			//	//log.Errorf("final, duplicate id: %v",id)
			//}

			//if v,ok:=processor.finalRecords.Load(id);ok{
			//	log.Errorf("final, duplicate id: %v, offset: %v",id,v)
			//}

			processor.finalRecords.Store(id, message.Offset.String())

			//processor.finalStageBitmap.Add(id)
		}

		//log.Errorf("final stage message count: %v, hash: %v, map:%v", processor.finalMessageCount, processor.finalStageBitmap.Size(), util.GetSyncMapSize(&processor.finalRecords))

		return true
	}, &wg)

	time.Sleep(10 * time.Second)

	////processor.requestMap = sync.Map{}
	//var lastCommitTime = time.Now()

	//var lastCheckTime = time.Now()

	wg.Add(1)
	WALConsumer := processor.getConsumer("primary_write_ahead_log")
	//fetch the message from the wal queue
	go processor.fetchMessages(WALConsumer, func(consumer queue.ConsumerAPI, messages []queue.Message) bool {
		processor.timestampMessageFetchedInWAL = time.Now()
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

				//log.Errorf("start to commit: %v", lastCommitableMessageOffset)
				WALConsumer.CommitOffset(lastCommitableMessageOffset)

				if processor.lastPreCommitableMessageOffset.Position > 0 {
					//log.Errorf("commit pre log: %v", processor.lastPreCommitableMessageOffset)
					preCommitLogConsumer.CommitOffset(processor.lastPreCommitableMessageOffset)
				}

				if processor.lastFirstCommitableMessageOffset.Position > 0 {
					//log.Errorf("commit first log: %v", processor.lastFirstCommitableMessageOffset)
					firstCommitLogConsumer.CommitOffset(processor.lastFirstCommitableMessageOffset)
				}

				if processor.lastFinalCommitableMessageOffset.Position > 0 {
					//log.Errorf("commit final log: %v", processor.lastFinalCommitableMessageOffset)
					finalCommitLogConsumer.CommitOffset(processor.lastFinalCommitableMessageOffset)
				}
			}

		}()

		//log.Errorf("WAL message size: %v", len(messages))
		//hitSizeLimit:=false
		//if processor.walBitmap.GetCardinality() > processor.config.WALMessageCheckingLimit {
		//	log.Error("too large wal bitmap size:", processor.walBitmap.GetCardinality(), ",sleep 1 second")
		//	hitSizeLimit=true
		//}

		req := fasthttp.AcquireRequest()
		for _, message := range messages {
			processor.totalWALMessageProcessed++
			//messagesInWAL++
			processor.lastOffsetInWAL = message.Offset.Position
			if global.ShuttingDown() {
				return false
			}

			err := req.Decode(message.Data)
			if err != nil {
				panic(err)
			}
			uID := string(req.Header.Peek("X-Replicated-ID"))
			//id, err := util.ToInt64(string(idstr))
			//if err != nil {
			//	panic("invalid id:" + string(idstr) + ",error:" + err.Error())
			//}

			//log.Error(id, ",", message.Offset, ",size:", util.ByteSize(processor.walBitmap.GetSizeInBytes()))

			//uID := (id)

			retry_times := 0

			//update commit offset
			lastCommitableMessageOffset = message.NextOffset
			processor.lastWALCommitableTimestamp = message.Timestamp - 60 //TODO: remove this hard code

		RETRY:
			//check final stage
			if _, ok := processor.finalRecords.Load(uID); ok { //.Contains(uID)
				//valid request, cleanup
				processor.cleanup(uID)
				processor.finishedMessage++
				//log.Error("OK:",uID,",",message.Offset)

				//err := consumer.CommitOffset(message.NextOffset)
				//if err != nil {
				//	panic(err)
				//}

			} else {

				hit := false
				gap := processor.lastOffsetInFinalStage - processor.lastOffsetInWAL

				if gap > processor.config.SafetyCheckSize {
					hit = true
					//log.Error("hit gap:", gap)
				}

				if retry_times > 10 {
					hit = true
					//log.Error("hit retry times:", retry_times)
				}

				if time.Since(processor.lastFetchedMessage) > time.Second*30 {
					hit = true
					//log.Error("hit lastFetchedMessage:", time.Since(processor.lastFetchedMessage))
				}

				if time.Since(lastTimestampFetchedAnyMessageInFinalStage) > time.Second*30 {
					hit = true
					//log.Error("hit last fetched final log:", time.Since(lastTimestampFetchedAnyMessageInFinalStage), ",docs:", processor.finalStageBitmap.Size())
				}

				if timestampMessageFetchedInFinalStage > 0 && (timestampMessageFetchedInFinalStage-60) > message.Timestamp {
					hit = true
					//log.Error("last commit > msg timestamp:", timestampMessageFetchedInFinalStage, message.Timestamp)
				}

				//if last commit time is more than 30 seconds, compare to wal or now, then this message maybe lost
				if hit { //too long no message returned, maybe finished

					var errLog string
					//check if in first stage
					if processor.firstStageBitmap.Contains(uID) {
						errLog = fmt.Sprintf("request %v, offset: %v, %v in first stage but not in final stage", uID, message.Offset, message.Timestamp)
					} else {
						errLog = fmt.Sprintf("request %v, offset: %v, %v exists in wal but not in any stage", uID, message.Offset, message.Timestamp)
					}

					//log.Error("FAIL:",uID,",",errLog)

					queue.Push(queue.GetOrInitConfig("replicate_failure_log"), []byte(errLog))

					err := queue.Push(queue.GetOrInitConfig("primary-failure"), message.Data)
					if err != nil {
						panic(err)
					}

					processor.unfinishedMessage++
					processor.cleanup(uID)

					//lastCommitableMessageOffset = message.NextOffset
					//processor.lastWALCommitableTimestamp = message.Timestamp

					//save message for later usage
					//processor.requestMap.Store(uID, message) //TODO

					//unknown request status, add to wal, and will check later
					//processor.walBitmap.Add(uID)

					retry_times = 0
				} else {
					//log.Infof("request %v, offset: %v,"+
					//	" retry_times: %v, docs_in_wal_stage: %v,docs_in_final_stage: %v, last_offset: %v, last_wal_offset: %v, gap: %v",
					//	uID, message.Offset,
					//	retry_times, processor.walBitmap.Size(), processor.finalStageBitmap.Size(), processor.lastOffsetInFinalStage, processor.lastOffsetInWAL, gap)
					time.Sleep(1 * time.Second)
					retry_times++

					//retry
					goto RETRY
				}
			}

		}

		//y := processor.walBitmap.Clone()
		//y.And(processor.finalStageBitmap)
		//
		//if y() > 0 {
		//	log.Error("wal bitmap and final stage bitmap has intersection, size:", y.GetCardinality())
		//	z := y.ToArray()
		//	for _, x := range z {
		//		processor.cleanup(x)
		//	}
		//}

		//if time.Since(lastCheckTime).Seconds() > 30||hitSizeLimit {
		//	lastCheckTime=time.Now()
		//	arr:=processor.walBitmap.ToArray()
		//	var errLog string
		//	for _,x:=range arr{
		//		v,ok:=processor.requestMap.Load(x)
		//		if ok{
		//			msg,ok:=v.(queue.Message)
		//			if ok{
		//				if time.Since(util.FromUnixTimestamp(msg.Timestamp)).Seconds()>30{
		//					//check if in first stage
		//					if processor.firstStageBitmap.Contains(x) {
		//						errLog = fmt.Sprintf("request %v, offset: %v, %v in first stage but not in final stage", x, msg.Offset,msg.Timestamp)
		//					} else {
		//						errLog = fmt.Sprintf("request %v, offset: %v, %v exists in wal but not in any stage", x, msg.Offset,msg.Timestamp)
		//					}
		//
		//					queue.Push(queue.GetOrInitConfig("replicate_failure_log"), []byte(errLog))
		//
		//					err := queue.Push(queue.GetOrInitConfig("primary-failure"), msg.Data)
		//					if err != nil {
		//						panic(err)
		//					}
		//
		//					processor.cleanup(x)
		//				}
		//			}
		//		}
		//
		//	}
		//}

		////remove from wal
		//if processor.walBitmap.GetCardinality() == 0 {
		//if time.Since(lastCommitTime) > time.Second*10 {
		if consumer != nil && lastCommitableMessageOffset.Position > 0 {
			//consumer.CommitOffset(lastCommitableMessageOffset)
			//lastCommitTime = time.Now()
		}
		//}
		//}

		return true

	}, &wg)

	wg.Wait()

	//log.Errorf("total:%v,finished:%v,unfinished:%v,pre:%v, %v, %v,first:%v,final:%v, last_pre_offset:%v", processor.totalWALMessageProcessed,processor.finishedMessage,processor.unfinishedMessage,
	//	numOfPreMessages, len(processor.preStageBitmap.ToArray()), processor.preStageBitmap.GetCardinality(), processor.firstStageBitmap.GetCardinality(), processor.finalStageBitmap.GetCardinality(),
	//	lastOffsetInPreCommit.String())

	//cleanup
	if time.Since(processor.timestampMessageFetchedInWAL) > time.Second*60 {

		if processor.lastPreCommitableMessageOffset.Position > 0 {
			//log.Errorf("commit pre log: %v", processor.lastPreCommitableMessageOffset)
			preCommitLogConsumer.CommitOffset(processor.lastPreCommitableMessageOffset)
		}

		if processor.lastFirstCommitableMessageOffset.Position > 0 {
			//log.Errorf("commit first log: %v", processor.lastFirstCommitableMessageOffset)
			firstCommitLogConsumer.CommitOffset(processor.lastFirstCommitableMessageOffset)
		}

		if processor.lastFinalCommitableMessageOffset.Position > 0 {
			//log.Errorf("commit final log: %v", processor.lastFinalCommitableMessageOffset)
			finalCommitLogConsumer.CommitOffset(processor.lastFinalCommitableMessageOffset)
		}
	}

	log.Debugf("total:%v,finished:%v,unfinished:%v, final_docs: %v, final_records:%v ,pre:%v, %v, %v,first:%v, last_pre_offset:%v, idle:%v",
		processor.totalWALMessageProcessed, processor.finishedMessage, processor.unfinishedMessage, processor.finalMessageCount, util.GetSyncMapSize(&processor.finalRecords),
		numOfPreMessages, processor.preStageBitmap.Size(), processor.preStageBitmap.Size(), processor.firstStageBitmap.Size(),
		lastOffsetInPreCommit.String(), time.Since(processor.timestampMessageFetchedInWAL))

	////get 3 bitmap
	//x := processor.preStageBitmap.Clone()
	//x.Xor(processor.finalStageBitmap)
	//x.And(processor.preStageBitmap)
	//
	//log.Errorf("pre and final intersection size:%v, result: %v", x.GetCardinality(), x.ToArray())

	return nil
}

func (processor *ReplicationCorrectionProcessor) getConsumer(queueName string) queue.ConsumerAPI {
	qConfig := queue.GetOrInitConfig(queueName)
	cConfig := queue.GetOrInitConsumerConfig(qConfig.ID, "crc", "name1")
	consumer, err := queue.AcquireConsumer(qConfig,
		cConfig)
	if err != nil {
		panic(err)
	}
	return consumer
}

func (processor *ReplicationCorrectionProcessor) commitable(message queue.Message) bool {
	//old message than wal, or wal fetched but idle for 30s
	return processor.lastWALCommitableTimestamp > 0 && message.Timestamp > 0 && message.Timestamp < processor.lastWALCommitableTimestamp||
		(processor.totalWALMessageProcessed>0&&time.Since(processor.timestampMessageFetchedInWAL) > time.Second*30)
}
