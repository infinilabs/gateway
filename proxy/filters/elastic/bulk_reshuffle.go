package elastic

import (
	"bufio"
	"bytes"
	"fmt"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/param"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/gateway/common"
	log "github.com/cihub/seelog"
	"strings"
	"sync"
)

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
			buff.Grow(10*1024)
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
}

var actionIndex= []byte("index")
var actionDelete= []byte("delete")
var actionCreate= []byte("create")
var actionUpdate= []byte("update")

var actionStart=[]byte("\"")
var actionEnd=[]byte("\"")

var indexStart=[]byte("\"_index\"")
var indexEnd=[]byte("\",")

var filteredFromValue=[]byte(": \"")

var idStart=[]byte("\"_id\"")
var idEnd=[]byte("}")

func parseActionMeta(data []byte) ( []byte,[]byte,[]byte) {

	action:=util.ExtractFieldFromBytes(&data,actionStart,actionEnd,nil)
	index:=util.ExtractFieldFromBytes(&data,indexStart,indexEnd,filteredFromValue)
	id:=util.ExtractFieldFromBytes(&data,idStart,idEnd,filteredFromValue)

	return action,index,id
}

func parseJson(scannedByte []byte)(action []byte,index,id string)  {
	//use Json
	var meta BulkActionMetadata
	meta=BulkActionMetadata{}
	util.FromJSONBytes(scannedByte,&meta)

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

func (this BulkReshuffle) Process(ctx *fasthttp.RequestCtx) {

	ctx.Set(common.CACHEABLE, false)

	clusterName:=this.MustGetString("elasticsearch")
	metadata:=elastic.GetMetadata(clusterName)
	if metadata==nil{
		log.Warnf("elasticsearch [%v] metadata is nil, skip reshuffle",clusterName)
		//fmt.Println("metadta is nil")
		return
	}

	initPool()

	path:=string(ctx.URI().Path())

	//TODO 处理 {INDEX}/_bulk 的情况
	//filebeat 等都是 bulk 结尾的请求了。
	//需要拆解 bulk 请求，重新封装
	if util.PrefixStr(path,"/_bulk"){

		reshuffleType:=this.GetStringOrDefault("level","node")
		submitMode:=this.GetStringOrDefault("mode","sync") //sync and async

		body:=ctx.Request.GetRawBody()

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

				if true{
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

					if len(action)==0||index==""||id==""{
						log.Warn("invalid bulk action:",string(action),",index:",string(indexb),",id:",string(idb),", try json parse")
						action,index,id=parseJson(scannedByte)
					}
				}

				if  bytes.Equal(action,actionDelete){
					//check metadata, if not delete, then is Meta is false
					nextIsMeta =true
				}

				indexSettings,ok:=metadata.Indices[index]

				if !ok{
					metadata=elastic.GetMetadata(clusterName)
					//fmt.Println(util.ToJson(metadata.Indices,true))
					log.Warn("index setting not found,",index,",",string(scannedByte))
					return
				}

				if indexSettings.Shards!=1{
					//如果 shards=1，则直接找主分片所在节点，否则计算一下。
					shardID=elastic.GetShardID([]byte(id),indexSettings.Shards)

					//shardsInfo:=metadata.GetPrimaryShardInfo(index,shardID)
					//nodeInfo:=metadata.GetNodeInfo(shardsInfo.NodeID)
					//fmt.Println(index,id,shardID,shardsInfo,nodeInfo)
					//TODO cache index-shard -> endpoint, 10s

				}

				shardInfo:=metadata.GetPrimaryShardInfo(index,shardID)
				//write meta
				bufferKey:=common.GetNodeLevelShuffleKey(clusterName,shardInfo.NodeID)
				if reshuffleType=="shard"{
					bufferKey=common.GetShardLevelShuffleKey(clusterName,index,shardID)
				}

				buff,ok=docBuf[bufferKey]
				if!ok{
					buff=bufferPool.Get().(*bytes.Buffer)
					buff.Reset()
					docBuf[bufferKey]=buff
				}
				buff.Write(scannedByte)
				buff.WriteString("\n")

			}else{
				nextIsMeta =true
				//handle request body
				buff.Write(scannedByte)
				buff.WriteString("\n")
			}
		}

		client:=elastic.GetClient(clusterName)

		for x,y:=range docBuf{
			if submitMode=="sync"{
				client.Bulk(y)
				ctx.Response.SetDestination(fmt.Sprintf("%v:%v","sync",x))
			}else{
				err:=queue.Push(x,y.Bytes())
				if err!=nil{
					panic(err)
				}
				ctx.Response.SetDestination(fmt.Sprintf("%v:%v","async",x))
			}
			y.Reset()
			bufferPool.Put(y)
		}

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
