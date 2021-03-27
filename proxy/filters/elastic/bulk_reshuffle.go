package elastic

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/param"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/stats"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/gateway/common"
	"net/http"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"
)

var JSON_CONTENT_TYPE = "application/json"

type BulkReshuffle struct {
	param.Parameters
}

func (this BulkReshuffle) Name() string {
	return "bulk_reshuffle"
}

var bufferPool *sync.Pool

func initPool() {
	if bufferPool !=nil{
		return
	}
	bufferPool = &sync.Pool {
		New: func()interface{} {
			buff:=&bytes.Buffer{}
			return buff
		},
	}
}

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

var versions= map[string]int{}
var versionLock=sync.Mutex{}

func (this BulkReshuffle) Process(filterCfg *common.FilterConfig,ctx *fasthttp.RequestCtx) {

	path:=string(ctx.URI().Path())

	//TODO 处理 {INDEX}/_bulk 的情况
	//filebeat 等都是 bulk 结尾的请求了。
	//需要拆解 bulk 请求，重新封装
	if util.PrefixStr(path,"/_bulk"){

		ctx.Set(common.CACHEABLE, false)

		clusterName:=this.MustGetString("elasticsearch")
		esMajorVersion,ok:=versions[clusterName]
		if !ok{
			versionLock.Lock()
			esMajorVersion:=elastic.GetClient(clusterName).GetMajorVersion()
			versions[clusterName]=esMajorVersion
			versionLock.Unlock()
		}

		metadata:=elastic.GetMetadata(clusterName)
		if metadata==nil{
			log.Warnf("elasticsearch [%v] metadata is nil, skip reshuffle",clusterName)
			//fmt.Println("metadta is nil")
			return
		}

		esConfig:=elastic.GetConfig(clusterName)

		initPool()

		safetyParse:=this.GetBool("safety_parse",true)
		validMetadata:=this.GetBool("valid_metadata",false)
		reshuffleType:=this.GetStringOrDefault("level","node")
		submitMode:=this.GetStringOrDefault("mode","sync") //sync and async
		fixNullID:=this.GetBool("fix_null_id",true) //sync and async
		IndexAnalysis:=this.GetBool("index_stats",true) //sync and async
		enabledShards,checkShards := this.GetStringArray("shards")

		body:=ctx.Request.GetRawBody()

		scanner := bufio.NewScanner(bytes.NewReader(body))
		scanner.Split(util.GetSplitFunc([]byte("\n")))
		nextIsMeta :=true

		//index-shardID -> buffer
		docBuf := map[string]*bytes.Buffer{}
		buffEndpoints := map[string]string{}
		skipNext:=false
		var buff *bytes.Buffer
		var indexStatsData map[string]int
		var indexStatsLock sync.Mutex
		shardID:=0
		for scanner.Scan() {
			scannedByte := scanner.Bytes()
			if scannedByte ==nil||len(scannedByte)<=0{
				continue
			}

			if skipNext{
				skipNext=false
				nextIsMeta =true
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

				//统计索引次数
				stats.Increment("elasticsearch."+clusterName+".indexing",index)
				if IndexAnalysis{
					//init
					if indexStatsData==nil{
						indexStatsLock.Lock()
						if indexStatsData==nil{
							indexStatsData=map[string]int{}
						}
						indexStatsLock.Unlock()
					}

					//stats
					indexStatsLock.Lock()
					v,ok:=indexStatsData[index]
					if !ok{
						indexStatsData[index]=1
					}else{
						indexStatsData[index]=v+1
					}
					indexStatsLock.Unlock()
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

				if validMetadata{
					obj:=map[string]interface{}{}
					err:=util.FromJSONBytes(scannedByte,&obj)
					if err!=nil{
						log.Error("error on validate action metadata")
						panic(err)
					}

				}

				if len(action)==0||index==""||id=="" {
					log.Warn("invalid bulk action:",string(action),",index:",string(index),",id:",string(id),",",string(scannedByte))
					return
				}

				if  bytes.Equal(action,actionDelete){
					//check metadata, if not delete, then is Meta is false
					nextIsMeta =true
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
								log.Warn("writer_index was not found in alias settings,",index,",",alias)
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
					log.Warn("index setting not found,",index,",",string(scannedByte))
					return
				}

				CONTINUE_RESHUFFLE:

				if indexSettings.Shards!=1{
					//如果 shards=1，则直接找主分片所在节点，否则计算一下。
					shardID=elastic.GetShardID(esMajorVersion,[]byte(id),indexSettings.Shards)

					if global.Env().IsDebug{
						log.Tracef("%s/%s => %v",index,id,shardID)
					}

					//shardsInfo:=metadata.GetPrimaryShardInfo(index,shardID)
					//nodeInfo:=metadata.GetNodeInfo(shardsInfo.NodeID)
					//fmt.Println(index,id,shardID,shardsInfo,nodeInfo)
					//TODO cache index-shard -> endpoint, 10s

					//save endpoint for bufferkey
					if checkShards && len(enabledShards)>0{
						if !util.ContainsAnyInArray(strconv.Itoa(shardID),enabledShards){
							log.Debugf("%s-%s not enabled, skip processing",index,shardID)
							skipNext=true
							continue
						}
					}

				}

				shardInfo:=metadata.GetPrimaryShardInfo(index,shardID)
				if shardInfo==nil{
					log.Warn("shardInfo was not found,",index,",",shardID)
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
					//buff=bufferPool.Get().(*bytes.Buffer)
					buff=&bytes.Buffer{}
					//buff=bufferPool.Get().(*bytes.Buffer)
					//buff.Reset()
					docBuf[bufferKey]=buff

					nodeInfo := metadata.GetNodeInfo(shardInfo.NodeID)
					if nodeInfo==nil{
						log.Warn("nodeInfo not found,",shardID,",",shardInfo.NodeID)
						return
					}
					buffEndpoints[bufferKey]=nodeInfo.Http.PublishAddress
					//if global.Env().IsDebug{
					//	log.Debug(shardInfo.Index,",",shardInfo.ShardID,",",nodeInfo.Http.PublishAddress)
					//}

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

		for x,y:=range docBuf{
			if submitMode=="sync"{
				endpoint,ok:=buffEndpoints[x]
				if !ok{
					log.Error("shard endpoint was not found,",x,",",shardID)
					//TODO
					return
				}

				ok=this.Bulk(&esConfig,endpoint,y)
				if !ok{
					log.Error("bulk failed on endpoint,",x,",",shardID)
					//TODO
					return
				}

				ctx.Response.SetDestination(fmt.Sprintf("%v:%v","sync",x))
			}else{
				err:=queue.Push(x,y.Bytes())
				if err!=nil{
					panic(err)
				}
				ctx.Response.SetDestination(fmt.Sprintf("%v:%v","async",x))
			}
			//y.Reset()
			//bufferPool.Put(y) //TODO
		}


		if IndexAnalysis{
			ctx.Set("bulk_index_stats",indexStatsData)
		}

		ctx.Set("elastic_cluster_name",clusterName)

		ctx.SetContentType(JSON_CONTENT_TYPE)
		ctx.WriteString("{\n  \"took\" : 0,\n  \"errors\" : false,\n  \"items\" : []\n}")
		ctx.Response.SetStatusCode(200)
		ctx.Finished()
		return
	}

	return


	//处理单次请求。
	pathItems:=strings.Split(path,"/")
	if len(pathItems)!=4{
		//fmt.Println("not a valid indexing request,",len(pathItems),pathItems)
		return
	}

	//validate index,type,id
	//index:=pathItems[1]
	//docType:=pathItems[2]
	//docID:=pathItems[3]

	//fmt.Println("index:",index,",type:",docType,",id:",docID)

	//get index settings
	//numOfTotalShards:=




	//if shardID!=4{
	//	ctx.Finished()
	//	return
	//}

	return
	//排除条件，非 _ 开头的索引。
	//可以指定排除和允许的索引，设置匹配的索引名称，通配符。

	//PUT/POST index/_doc/UUID
	//只有匹配到是单独的索引请求才会进行合并处理。
	//放内存里面，按节点或者分片为单位进行缓存，或者固定的通道数，固定通道数<按节点<按分片。
	//count、size 和 timeout 任意满足即进行 bulk 提交。

	//通过 ID 获取到分片所在节点位置，没有 ID 就获取到包含主分片的节点，均衡选择，或者主动生成 ID。

	//变成 bulk 格式



	//defer writerPool.Put(w)
	//
	//err := request.MarshalFastJSON(w)
	//if err != nil {
	//	panic(err)
	//}
	//
	//err = queue.Push(this.GetStringOrDefault("queue_name","request_logging"),w.Bytes() )
	//if err != nil {
	//	panic(err)
	//}

}

//TODO 提取出来，作为公共方法，和 indexing/bulking_indexing 的方法合并

var fastHttpClient = &fasthttp.Client{
	TLSConfig: &tls.Config{InsecureSkipVerify: true},
}

func (joint BulkReshuffle) Bulk(cfg *elastic.ElasticsearchConfig, endpoint string, data *bytes.Buffer) bool{
	if data == nil || data.Len() == 0 {
		return true
	}
	data.WriteRune('\n')

	if cfg.IsTLS() {
		endpoint = "https://" + endpoint
	} else {
		endpoint = "http://" + endpoint
	}
	url := fmt.Sprintf("%s/_bulk", endpoint)
	compress := joint.GetBool("compress",true)

	req := fasthttp.AcquireRequest()
	req.Reset()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)   // <- do not forget to release
	defer fasthttp.ReleaseResponse(resp) // <- do not forget to release

	req.SetRequestURI(url)
	req.Header.SetMethod(http.MethodPost)
	req.Header.SetUserAgent("bulk_indexing")

	if compress {
		req.Header.Set("Accept-Encoding", "gzip")
		req.Header.Set("content-encoding", "gzip")
	}

	req.Header.SetContentType("application/json")

	if cfg.BasicAuth != nil{
		req.URI().SetUsername(cfg.BasicAuth.Username)
		req.URI().SetPassword(cfg.BasicAuth.Password)
	}

	if data.Len() > 0 {
		if compress {
			_, err := fasthttp.WriteGzipLevel(req.BodyWriter(), data.Bytes(), fasthttp.CompressBestSpeed)
			if err != nil {
				panic(err)
			}
		} else {
			//req.SetBody(body)
			req.SetBodyStreamWriter(func(w *bufio.Writer) {
				w.Write(data.Bytes())
				w.Flush()
			})

		}
	}
	retryTimes:=0

DO:

	err := fastHttpClient.Do(req, resp)

	if err != nil {
		if global.Env().IsDebug{
			log.Error(err)
		}
		return false
	}

	if resp == nil {
		if global.Env().IsDebug{
			log.Error(err)
		}
		return false
	}


	// Do we need to decompress the response?
	var resbody =resp.GetRawBody()
	if global.Env().IsDebug{
		log.Trace(resp.StatusCode(),string(resbody))
	}

	if resp.StatusCode()==400{

		if joint.GetBool("log_bulk_message",true) {
			path1 := path.Join(global.Env().GetWorkingDir(), "bulk_400_failure.log")
			truncateSize := joint.GetIntOrDefault("error_message_truncate_size", -1)
			util.FileAppendNewLineWithByte(path1, []byte("URL:"))
			util.FileAppendNewLineWithByte(path1, []byte(url))
			util.FileAppendNewLineWithByte(path1, []byte("Request:"))
			reqBody:=data.Bytes()
			resBody1:=resbody
			if truncateSize>0{
				if len(reqBody)>truncateSize{
					reqBody=reqBody[:truncateSize]
				}
				if len(resBody1)>truncateSize{
					resBody1=resBody1[:truncateSize]
				}
			}
			util.FileAppendNewLineWithByte(path1,reqBody )
			util.FileAppendNewLineWithByte(path1, []byte("Response:"))
			util.FileAppendNewLineWithByte(path1, resBody1)
		}
		return false
	}

	//TODO check respbody's error
	if resp.StatusCode() == http.StatusOK || resp.StatusCode() == http.StatusCreated {

		//200{"took":2,"errors":true,"items":[
		if resp.StatusCode()==http.StatusOK{
			//handle error items
			//"errors":true
			hit:=util.LimitedBytesSearch(resbody,[]byte("\"errors\":true"),64)
			if hit{
				if joint.GetBool("log_bulk_message",true) {
					path1 := path.Join(global.Env().GetWorkingDir(), "bulk_req_failure.log")
					truncateSize := joint.GetIntOrDefault("error_message_truncate_size", -1)
					util.FileAppendNewLineWithByte(path1, []byte("URL:"))
					util.FileAppendNewLineWithByte(path1, []byte(url))
					util.FileAppendNewLineWithByte(path1, []byte("Request:"))
					reqBody:=data.Bytes()
					resBody1:=resbody
					if truncateSize>0{
						if len(reqBody)>truncateSize{
							reqBody=reqBody[:truncateSize]
						}
						if len(resBody1)>truncateSize{
							resBody1=resBody1[:truncateSize]
						}
					}
					util.FileAppendNewLineWithByte(path1,reqBody )
					util.FileAppendNewLineWithByte(path1, []byte("Response:"))
					util.FileAppendNewLineWithByte(path1, resBody1)
				}
				if joint.GetBool("warm_retry_message",false){
					log.Warnf("elasticsearch bulk error, retried %v times, will try again",retryTimes)
				}

				retryTimes++
				delayTime := joint.GetIntOrDefault("retry_delay_in_second", 5)
				time.Sleep(time.Duration(delayTime)*time.Second)
				goto DO
			}
		}

		return true
	} else if resp.StatusCode()==429 {
		log.Warnf("elasticsearch rejected, retried %v times, will try again",retryTimes)
		delayTime := joint.GetIntOrDefault("retry_delay_in_second", 5)
		time.Sleep(time.Duration(delayTime)*time.Second)
		if retryTimes>300{
			if joint.GetBool("warm_retry_message",false){
				log.Errorf("elasticsearch rejected, retried %v times, quit retry",retryTimes)
			}
			return false
		}
		retryTimes++
		goto DO
	}else {
		if joint.GetBool("log_bulk_message",true){
			path1:=path.Join(global.Env().GetWorkingDir(),"bulk_error_failure.log")
			truncateSize := joint.GetIntOrDefault("error_message_truncate_size", -1)
			util.FileAppendNewLineWithByte(path1, []byte("URL:"))
			util.FileAppendNewLineWithByte(path1, []byte(url))
			util.FileAppendNewLineWithByte(path1, []byte("Request:"))
			reqBody:=data.Bytes()
			resBody1:=resbody
			if truncateSize>0{
				if len(reqBody)>truncateSize{
					reqBody=reqBody[:truncateSize-1]
				}
				if len(resBody1)>truncateSize{
					resBody1=resBody1[:truncateSize-1]
				}
			}
			util.FileAppendNewLineWithByte(path1,reqBody )
			util.FileAppendNewLineWithByte(path1, []byte("Response:"))
			util.FileAppendNewLineWithByte(path1, resBody1)

		}
		if joint.GetBool("warm_retry_message",false){
			log.Errorf("invalid bulk response, %v - %v",resp.StatusCode(),string(resbody))
		}
		return false
	}
	return true
}

