/*
Copyright 2016 Medcl (m AT medcl.net)

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package indexing

import (
	"bufio"
	"bytes"
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/param"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/stats"
	"infini.sh/framework/core/util"
	"infini.sh/gateway/common"
	"runtime"
	"sync"
	"time"
)

var bufferPool = &sync.Pool {
		New: func()interface{} {
			buff:=&bytes.Buffer{}
			buff.Grow(10*1024)
			return buff
		}}

//{ "index" : { "_index" : "test", "_id" : "1" } }
//{ "delete" : { "_index" : "test", "_id" : "2" } }
//{ "create" : { "_index" : "test", "_id" : "3" } }
//{ "update" : {"_id" : "1", "_index" : "test"} }
type BulkActionMetadata struct {
	Index *BulkIndexMetadata`json:"index,omitempty"`
	Delete *BulkIndexMetadata `json:"delete,omitempty"`
	Create *BulkIndexMetadata `json:"create,omitempty"`
	Update *BulkIndexMetadata `json:"update,omitempty"`
}

type BulkIndexMetadata struct {
	Index string  `json:"_index,omitempty"`
	Type string  `json:"_type,omitempty"`
	ID string  `json:"_id,omitempty"`
	RequireAlias bool  `json:"require_alias,omitempty"`
	Parent1 bool  `json:"_parent,omitempty"`
	Parent2 bool  `json:"parent,omitempty"`
	Routing1 bool  `json:"routing,omitempty"`
	Routing2 bool  `json:"_routing,omitempty"`
	Version1 bool  `json:"_version,omitempty"`
	Version2 bool  `json:"version,omitempty"`
}

var actionIndex= []byte("index")
var actionDelete= []byte("delete")
var actionCreate= []byte("create")
var actionUpdate= []byte("update")

var actionStart=[]byte("\"")
var actionEnd=[]byte("\"")

var indexStart=[]byte("\"_index\"")
var indexEnd=[]byte("\"")

var filteredFromValue=[]byte(": \"")

var idStart=[]byte("\"_id\"")
var idEnd=[]byte("\"")

func parseActionMeta(data []byte) ( []byte,[]byte,[]byte) {

	action:=util.ExtractFieldFromBytes(&data,actionStart,actionEnd,nil)
	index:=util.ExtractFieldFromBytesWitSkipBytes(&data,indexStart,[]byte("\""),indexEnd,filteredFromValue)
	id:=util.ExtractFieldFromBytesWitSkipBytes(&data,idStart,[]byte("\""),idEnd,filteredFromValue)

	return action,index,id
}

//"_index":"test" => "_index":"test", "_id":"id"
func insertUUID(scannedByte []byte)(newBytes []byte,id string)   {
	id=util.GetUUID()
	newData:=util.InsertBytesAfterField(&scannedByte,[]byte("\"_index\""),[]byte("\""),[]byte("\""),[]byte(",\"_id\":\""+id+"\""))
	return newData,id
}

func updateJsonWithUUID(scannedByte []byte)(newBytes []byte,id string)  {
	var meta BulkActionMetadata
	meta=BulkActionMetadata{}
	util.MustFromJSONBytes(scannedByte,&meta)
	id=util.GetUUID()
	if meta.Index!=nil{
		meta.Index.ID=id
	}else if meta.Create!=nil{
		meta.Create.ID=id
	}
	return util.MustToJSONBytes(meta),id
}

func parseJson(scannedByte []byte)(action []byte,index,id string)  {
	//use Json
	var meta BulkActionMetadata
	meta=BulkActionMetadata{}
	util.MustFromJSONBytes(scannedByte,&meta)

	if meta.Index!=nil{
		index=meta.Index.Index
		id=meta.Index.ID
		action=actionIndex
	}else if meta.Create!=nil{
		index=meta.Create.Index
		id=meta.Create.ID
		action=actionCreate
	}else if meta.Update!=nil{
		index=meta.Update.Index
		id=meta.Update.ID
		action=actionUpdate
	}else if meta.Delete!=nil{
		index=meta.Delete.Index
		action=actionDelete
		id=meta.Delete.ID
	}

	return action,index,id
}


type BulkReshuffleJoint struct {
	param.Parameters
}

func (joint BulkReshuffleJoint) Name() string {
	return "bulk_reshuffle"
}

func (joint BulkReshuffleJoint) Process(c *pipeline.Context) error {
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
				log.Error("error in json indexing,", v)
			}
		}
	}()

	workers, _ := joint.GetInt("worker_size", 1)
	bulkSizeInMB, _ := joint.GetInt("bulk_size_in_mb", 10)
	bulkSizeInMB = 1000000 * bulkSizeInMB

	wg := sync.WaitGroup{}
	totalSize := 0
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go joint.NewBulkWorker(&totalSize, bulkSizeInMB, &wg)
	}

	wg.Wait()

	return nil
}

func (joint BulkReshuffleJoint) NewBulkWorker(count *int, bulkSizeInMB int, wg *sync.WaitGroup) {

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
				log.Error("error in json indexing worker,", v)
				wg.Done()
			}
		}
	}()

	clusterName := joint.MustGetString("elasticsearch")

	timeOut := joint.GetIntOrDefault("idle_timeout_in_second", 5)
	idleDuration := time.Duration(timeOut) * time.Second
	idleTimeout := time.NewTimer(idleDuration)
	defer idleTimeout.Stop()

	//client := elastic.GetClient(clusterName)
	queueName:=fmt.Sprintf("%v_bulk", clusterName)

	esMajorVersion:=elastic.GetClient(clusterName).GetMajorVersion()

	safetyParse:=joint.GetBool("safety_parse",true)
	reshuffleType:=joint.GetStringOrDefault("level","node")
	submitMode:=joint.GetStringOrDefault("mode","async") //sync and async
	fixNullID:=joint.GetBool("fix_null_id",true) //sync and async

	for {
		idleTimeout.Reset(idleDuration)

		select {

		case body := <-queue.ReadChan(queueName):

			stats.IncrementBy("bulk_incoming", "bytes_received", int64(len(body)))

			//start

			scanner := bufio.NewScanner(bytes.NewReader(body))
			scanner.Split(util.GetSplitFunc([]byte("\n")))
			nextIsMeta :=true

			//index-shardID -> buffer
			docBuf := map[string]*bytes.Buffer{}
			var buff *bytes.Buffer
			shardID:=0
			for scanner.Scan() {
				scannedByte := scanner.Bytes()
				if scannedByte ==nil||len(scannedByte)<=0{
					continue
				}
				if nextIsMeta {
					nextIsMeta =false

					var index string
					var id string
					var action []byte

					if safetyParse{
						//parse with json
						action,index,id=parseJson(scannedByte)
					}else{
						var indexb,idb []byte

						//TODO action: update ,index:  ,id: 1,_indextest
						//{ "update" : {"_id" : "1", "_index" : "test"} }
						//字段顺序换了。
						action,indexb,idb=parseActionMeta(scannedByte)
						index=string(indexb)
						id=string(idb)

						if len(action)==0||index==""{
							log.Warn("invalid bulk action:",string(action),",index:",string(indexb),",id:",string(idb),", try json parse:",string(scannedByte))
							action,index,id=parseJson(scannedByte)
						}
					}

					if (bytes.Equal(action,[]byte("index"))||bytes.Equal(action,[]byte("create")))&&len(id)==0 && fixNullID {
						if safetyParse{
							scannedByte,id=updateJsonWithUUID(scannedByte)
						}else{
							scannedByte,id=insertUUID(scannedByte)
						}
						if global.Env().IsDebug{
							log.Trace("generated ID,",id,",",string(scannedByte))
						}
					}

					if len(action)==0||index==""||id=="" {
						log.Error("invalid bulk action:",string(action),",index:",string(index),",id:",string(id))
						//TODO
						return
					}

					if  bytes.Equal(action,actionDelete){
						//check metadata, if not delete, then is Meta is false
						nextIsMeta =true
					}

					GETMETA:
					metadata:=elastic.GetMetadata(clusterName)
					if metadata==nil{
						log.Warnf("elasticsearch [%v] metadata is nil, skip reshuffle",clusterName)
						//TODO retry
						time.Sleep(10*time.Second)
						goto GETMETA
						return
					}

					indexSettings,ok:=metadata.Indices[index]

					if !ok{
						metadata=elastic.GetMetadata(clusterName)
						if global.Env().IsDebug{
							log.Trace("index was not found in index settings,",index,",",string(scannedByte))
						}
						alias,ok:=metadata.Aliases[index]
						if ok{
							if global.Env().IsDebug{
								log.Trace("found index in alias settings,",index,",",string(scannedByte))
							}
							newIndex:=alias.WriteIndex
							if alias.WriteIndex==""{
								if len(alias.Index)==1{
									newIndex=alias.Index[0]
									if global.Env().IsDebug{
										log.Trace("found index in alias settings, no write_index, but only have one index, will use it,",index,",",string(scannedByte))
									}
								}else{
									log.Error("writer_index was not found in alias settings,",index,",",alias)
									//TODO
									time.Sleep(10*time.Second)
									goto GETMETA
									return
								}
							}
							indexSettings,ok=metadata.Indices[newIndex]
							if ok{
								if global.Env().IsDebug{
									log.Trace("index was found in index settings,",index,"=>",newIndex,",",string(scannedByte),",",indexSettings)
								}
								index=newIndex
								goto CONTINUE_RESHUFFLE
							}else{
								if global.Env().IsDebug{
									log.Trace("writer_index was not found in index settings,",index,",",string(scannedByte))
								}
							}
						}else{
							if global.Env().IsDebug{
								log.Trace("index was not found in alias settings,",index,",",string(scannedByte))
							}
						}

						//fmt.Println(util.ToJson(metadata.Indices,true))
						log.Error("index setting not found,",index,",",string(scannedByte))
						time.Sleep(10*time.Second)
						goto GETMETA
						//TODO
						return
					}

				CONTINUE_RESHUFFLE:

					if indexSettings.Shards!=1{
						//如果 shards=1，则直接找主分片所在节点，否则计算一下。
						shardID=elastic.GetShardID(esMajorVersion,[]byte(id),indexSettings.Shards)

						if global.Env().IsDebug{
							log.Tracef("%s/%s => %v",index,id,shardID)
						}

					}

					shardInfo:=metadata.GetPrimaryShardInfo(index,shardID)
					if shardInfo==nil{
						log.Error("shardInfo was not found,",index,",",shardID)
						time.Sleep(10*time.Second)
						goto GETMETA
						return
					}

					//write meta
					bufferKey:=common.GetNodeLevelShuffleKey(clusterName,shardInfo.NodeID)
					if reshuffleType=="shard"{
						bufferKey=common.GetShardLevelShuffleKey(clusterName,index,shardID)
					}

					if global.Env().IsDebug{
						log.Tracef("%s/%s => %v , %v",index,id,shardID,bufferKey)
					}

					buff,ok=docBuf[bufferKey]
					if!ok{
						buff=bufferPool.Get().(*bytes.Buffer)
						buff.Reset()
						docBuf[bufferKey]=buff
					}
					buff.Write(scannedByte)
					if global.Env().IsDebug{
						log.Trace("metadata:",string(scannedByte))
					}
					buff.WriteString("\n")

				}else{
					nextIsMeta =true
					//handle request body
					buff.Write(scannedByte)
					if global.Env().IsDebug{
						log.Trace("data:",string(scannedByte))
					}
					buff.WriteString("\n")
				}
			}

			//client:=elastic.GetClient(clusterName)

			//TODO 从内存移动到流式存储
			for x,y:=range docBuf{
				if submitMode=="sync"{
					//client.Bulk(y)
					//ctx.Response.SetDestination(fmt.Sprintf("%v:%v","sync",x))
				}else{
					err:=queue.Push(x,y.Bytes())
					if err!=nil{
						panic(err)
					}
					//ctx.Response.SetDestination(fmt.Sprintf("%v:%v","async",x))
				}
				y.Reset()
				bufferPool.Put(y)
			}


			//end



			//docBuf.Reset()
			//(*count)++
			//
			//if mainBuf.Len() > (bulkSizeInMB) {
			//	if global.Env().IsDebug {
			//		log.Trace("hit buffer size, ", mainBuf.Len())
			//	}
			//	goto CLEAN_BUFFER
			//}

		case <-idleTimeout.C:
			if global.Env().IsDebug{
				log.Tracef("%v no message input", idleDuration)
			}
			goto CLEAN_BUFFER
		}

		CLEAN_BUFFER:

	}
}
