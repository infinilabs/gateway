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
	SafetyCommitOffsetPaddingSize int64 `config:"safety_commit_offset_padding_size"` //offset between last and wal
	SafetyCommitIntervalInSeconds int64 `config:"safety_commit_interval_in_seconds"` //time between last and wal
}

type ReplicationCorrectionProcessor struct {
	config *Config

	walBitmap        *hashset.Set
	firstStageBitmap *hashset.Set
	finalRecords     sync.Map

	requestMap sync.Map

	lastOffsetInFirstStage int64     //last offset of first stage message
	lastOffsetInFinalStage int64     //last offset of final stage message
	lastOffsetInWAL        int64     //last offset of final stage message
	lastFetchedMessage     time.Time //last time fetched any message

	totalWALMessageProcessed int

	finishedMessage                  int
	unfinishedMessage                int
	finalMessageCount                int
	lastWALCommitableTimestamp       int64 //last message timestamp in wal -60s
	lastFinalCommitableMessageOffset queue.Offset
	lastFirstCommitableMessageOffset queue.Offset
	timestampMessageFetchedInWAL     time.Time
}

func init() {
	pipeline.RegisterProcessorPlugin("replication_correlation", New)
}

func New(c *config.Config) (pipeline.Processor, error) {
	cfg := Config{
		SafetyCommitOffsetPaddingSize: 10000,
		SafetyCommitIntervalInSeconds: 60,
	}

	if err := c.Unpack(&cfg); err != nil {
		log.Error(err)
		return nil, fmt.Errorf("failed to unpack the configuration of flow_runner processor: %s", err)
	}

	runner := ReplicationCorrectionProcessor{config: &cfg}

	if cfg.SafetyCommitIntervalInSeconds <= 0 {
		cfg.SafetyCommitIntervalInSeconds = 60
	}

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
	messages, _, err := consumer.FetchMessages(&ctx1, 5000)
	if err != nil {
		panic(err)
	}

	if len(messages) > 0 {

		processor.lastFetchedMessage = time.Now()

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
	firstCommitLogConsumer := processor.getConsumer("primary_first_commit_log")
	//check first stage commit
	go processor.fetchMessages(firstCommitLogConsumer, func(consumer queue.ConsumerAPI, messages []queue.Message) bool {
		for _, message := range messages {

			if processor.commitable(message) {
				processor.lastFirstCommitableMessageOffset = message.NextOffset
			}

			processor.lastOffsetInFirstStage = message.Offset.Position

			v := string(message.Data)

			id, _ := parseIDAndOffset(v)

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
			if processor.commitable(message) {
				processor.lastFinalCommitableMessageOffset = message.NextOffset
			}

			processor.finalMessageCount++
			timestampMessageFetchedInFinalStage = message.Timestamp

			processor.lastOffsetInFinalStage = message.Offset.Position

			v := string(message.Data)
			id, _ := parseIDAndOffset(v)
			processor.finalRecords.Store(id, message.Offset.String())
		}
		log.Debugf("final stage message count: %v, map:%v", processor.finalMessageCount, util.GetSyncMapSize(&processor.finalRecords))
		return true
	}, &wg)

	time.Sleep(10 * time.Second)

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
				WALConsumer.CommitOffset(lastCommitableMessageOffset)
				if processor.lastFirstCommitableMessageOffset.Position > 0 {
					firstCommitLogConsumer.CommitOffset(processor.lastFirstCommitableMessageOffset)
				}
				if processor.lastFinalCommitableMessageOffset.Position > 0 {
					finalCommitLogConsumer.CommitOffset(processor.lastFinalCommitableMessageOffset)
				}
			}

		}()

		req := fasthttp.AcquireRequest()
		for _, message := range messages {
			processor.totalWALMessageProcessed++
			processor.lastOffsetInWAL = message.Offset.Position
			if global.ShuttingDown() {
				return false
			}

			err := req.Decode(message.Data)
			if err != nil {
				panic(err)
			}
			idByte := req.Header.Peek("X-Replicated-ID")
			if idByte == nil || len(idByte) == 0 {
				panic("invalid id")
			}

			uID := string(idByte)

			//update commit offset
			lastCommitableMessageOffset = message.NextOffset
			processor.lastWALCommitableTimestamp = message.Timestamp - processor.config.SafetyCommitIntervalInSeconds

			retry_times := 0

		RETRY:
			//check final stage
			if _, ok := processor.finalRecords.Load(uID); ok {
				//valid request, cleanup
				processor.cleanup(uID)
				processor.finishedMessage++
			} else {
				hit := false
				if (processor.lastOffsetInFinalStage - processor.lastOffsetInWAL) > processor.config.SafetyCommitOffsetPaddingSize {
					hit = true
				}

				if retry_times > 10 {
					hit = true
				}

				if time.Since(processor.lastFetchedMessage) > time.Second*30 {
					hit = true
				}

				if time.Since(lastTimestampFetchedAnyMessageInFinalStage) > time.Second*30 {
					hit = true
				}

				if timestampMessageFetchedInFinalStage > 0 && (timestampMessageFetchedInFinalStage-60) > message.Timestamp {
					hit = true
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

					queue.Push(queue.GetOrInitConfig("replicate_failure_log"), []byte(errLog))

					err := queue.Push(queue.GetOrInitConfig("primary-failure"), message.Data)
					if err != nil {
						panic(err)
					}

					processor.unfinishedMessage++
					processor.cleanup(uID)

					retry_times = 0
				} else {
					if global.Env().IsDebug {
						log.Debugf("request %v, offset: %v,"+
							" retry_times: %v, docs_in_wal_stage: %v,docs_in_final_stage: %v, last_offset: %v, last_wal_offset: %v, gap: %v",
							uID, message.Offset,
							retry_times, processor.walBitmap.Size(), util.GetSyncMapSize(&processor.finalRecords), processor.lastOffsetInFinalStage, processor.lastOffsetInWAL)
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
	if time.Since(processor.timestampMessageFetchedInWAL) > time.Second*60 {

		if processor.lastFirstCommitableMessageOffset.Position > 0 {
			firstCommitLogConsumer.CommitOffset(processor.lastFirstCommitableMessageOffset)
		}

		if processor.lastFinalCommitableMessageOffset.Position > 0 {
			finalCommitLogConsumer.CommitOffset(processor.lastFinalCommitableMessageOffset)
		}
	}

	log.Debugf("total:%v,finished:%v,unfinished:%v, final: %v, final_records:%v, first:%v, idle:%v",
		processor.totalWALMessageProcessed, processor.finishedMessage, processor.unfinishedMessage, processor.finalMessageCount,
		util.GetSyncMapSize(&processor.finalRecords),
		 processor.firstStageBitmap.Size(),
		time.Since(processor.timestampMessageFetchedInWAL))

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
	return processor.lastWALCommitableTimestamp > 0 && message.Timestamp > 0 && message.Timestamp < processor.lastWALCommitableTimestamp ||
		(processor.totalWALMessageProcessed > 0 && time.Since(processor.timestampMessageFetchedInWAL) > time.Second*30)
}
