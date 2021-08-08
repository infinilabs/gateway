package elastic

import (
	"github.com/buger/jsonparser"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/param"
	"infini.sh/framework/core/util"
	"sync"
)

var JSON_CONTENT_TYPE = "application/json"

type BulkReshuffle struct {
	param.Parameters
}

func (this BulkReshuffle) Name() string {
	return "bulk_reshuffle"
}

var actionIndex ="index"
var actionDelete = "delete"
var actionCreate = "create"
var actionUpdate = "update"

var actionStart = []byte("\"")
var actionEnd = []byte("\"")

var actions = []string{"index","delete","create","update"}

func parseActionMeta(data []byte) (action, index, typeName, id string) {

	match:=false
	for _,v:=range actions{
		jsonparser.ObjectEach(data, func(key []byte, value []byte, dataType jsonparser.ValueType, offset int) error {
			switch util.UnsafeBytesToString(key) {
			case "_index":
				index=string(value)
				break
			case "_type":
				typeName=string(value)
				break
			case "_id":
				id=string(value)
				break
			}
			match=true
			return nil
		}, v)
		action=v
		if match{
			//fmt.Println(action,",",index,",",typeName,",", id)
			return action, index,typeName, id
		}
	}

	log.Warn("fallback to unsafe parse:",string(data))

	action = string(util.ExtractFieldFromBytes(&data, actionStart, actionEnd, nil))
	index,_=jsonparser.GetString(data,action,"_index")
	typeName,_=jsonparser.GetString(data,action,"_type")
	id,_=jsonparser.GetString(data,action,"_id")

	if index!=""{
		return action, index,typeName, id
	}

	log.Warn("fallback to safety parse:",string(data))
	return safetyParseActionMeta(data)
}

func updateJsonWithNewIndex(action string,scannedByte []byte, index, typeName, id string) (newBytes []byte,err error) {

	if global.Env().IsDebug{
		log.Trace("update:",action,",",index,",",typeName,",",id)
	}

	newBytes= make([]byte,len(scannedByte))
	copy(newBytes,scannedByte)

	if index != "" {
		newBytes,err=jsonparser.Set(newBytes, []byte("\""+index+"\""),action,"_index")
		if err!=nil{
			return newBytes,err
		}
	}
	if typeName != "" {
		newBytes,err=jsonparser.Set(newBytes, []byte("\""+typeName+"\""),action,"_type")
		if err!=nil{
			return newBytes,err
		}
	}
	if id != "" {
		newBytes,err=jsonparser.Set(newBytes, []byte("\""+id+"\""),action,"_id")
		if err!=nil{
			return newBytes,err
		}
	}

	return newBytes,err
}

//performance is poor
func safetyParseActionMeta(scannedByte []byte) (action , index, typeName, id string) {

	////{ "index" : { "_index" : "test", "_id" : "1" } }
	var meta = elastic.BulkActionMetadata{}
	meta.UnmarshalJSON(scannedByte)
	if meta.Index != nil {
		index = meta.Index.Index
		typeName = meta.Index.Type
		id = meta.Index.ID
		action = actionIndex
	} else if meta.Create != nil {
		index = meta.Create.Index
		typeName = meta.Create.Type
		id = meta.Create.ID
		action = actionCreate
	} else if meta.Update != nil {
		index = meta.Update.Index
		typeName = meta.Update.Type
		id = meta.Update.ID
		action = actionUpdate
	} else if meta.Delete != nil {
		index = meta.Delete.Index
		typeName = meta.Delete.Type
		action = actionDelete
		id = meta.Delete.ID
	}

	return action, index, typeName, id
}

var versions = map[string]int{}
var versionLock = sync.RWMutex{}
