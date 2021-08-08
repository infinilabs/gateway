package elastic

import (
	"infini.sh/framework/core/elastic"
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

var actionIndex = []byte("index")
var actionDelete = []byte("delete")
var actionCreate = []byte("create")
var actionUpdate = []byte("update")

var actionStart = []byte("\"")
var actionEnd = []byte("\"")

var indexStart = []byte("\"_index\"")
var indexEnd = []byte("\"")

var filteredFromValue = []byte(": \"")

var idStart = []byte("\"_id\"")
var idEnd = []byte("\"")

func parseActionMeta(data []byte) ([]byte, []byte, []byte) {
	action := util.ExtractFieldFromBytes(&data, actionStart, actionEnd, nil)
	index := util.ExtractFieldFromBytesWitSkipBytes(&data, indexStart, []byte("\""), indexEnd, filteredFromValue)
	id := util.ExtractFieldFromBytesWitSkipBytes(&data, idStart, []byte("\""), idEnd, filteredFromValue)
	return action, index, id
}

//TODO performance
func updateJsonWithNewIndex(scannedByte []byte, index, typeName, id string) (newBytes []byte) {
	var meta = elastic.BulkActionMetadata{}
	util.MustFromJSONBytes(scannedByte, &meta)
	if meta.Index != nil {
		if index != "" {
			meta.Index.Index = index
		}
		if typeName != "" {
			meta.Index.Type = typeName
		}
		if id != "" {
			meta.Index.ID = id
		}
	} else if meta.Create != nil {
		if index != "" {
			meta.Create.Index = index
		}
		if typeName != "" {
			meta.Create.Type = typeName
		}
		if id != "" {
			meta.Create.ID = id
		}
	} else if meta.Update != nil {
		if index != "" {
			meta.Update.Index = index
		}
		if typeName != "" {
			meta.Update.Type = typeName
		}
		if id != "" {
			meta.Update.ID = id
		}
	} else if meta.Delete != nil {
		if index != "" {
			meta.Delete.Index = index
		}
		if typeName != "" {
			meta.Delete.Type = typeName
		}
		if id != "" {
			meta.Delete.ID = id
		}
	}
	return util.MustToJSONBytes(meta)
}

func parseJson(scannedByte []byte) (action []byte, index, typeName, id string) {
	//use Json
	var meta = elastic.BulkActionMetadata{}
	util.MustFromJSONBytes(scannedByte, &meta)
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
