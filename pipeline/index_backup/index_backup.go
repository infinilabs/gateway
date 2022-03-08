/* Copyright © INFINI Ltd. All rights reserved.
 * web: https://infinilabs.com
 * mail: hello#infini.ltd */

package index_backup

import (
	"fmt"
	log "github.com/cihub/seelog"
	"github.com/emirpasic/gods/sets/hashset"
	"github.com/fsnotify/fsnotify"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/kv"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/s3"
	"infini.sh/framework/core/util"
	"os"
	path2 "path"
	"path/filepath"
	"runtime"
	"time"
)

type Config struct {
	Elasticsearch string `config:"elasticsearch"`
	Index string 		`config:"index"`
	UploadToS3   bool   `config:"upload_to_s3"`
	UploadToFSRepo   bool   `config:"upload_to_fs_repo"`
	LocalRepoPath   string   `config:"local_repo_path"`

	S3 struct{
		Async   bool   `config:"async"`
		Server   string   `config:"server"`
		Location   string   `config:"location"`
		Bucket   string   `config:"bucket"`
	}`config:"s3"`
}

type IndexBackupProcessor struct {
	config    *Config
	skipFiles *hashset.Set
	watch     *fsnotify.Watcher
	changes map[string]*ChangedItem
	changesChan chan *ChangedItem
}

var signalChannel = make(chan bool, 1)

func init()  {
	pipeline.RegisterProcessorPlugin("index_backup", New)
}

type ChangedItem struct {
	IndexUUID string
	SegmentID string
	FilePath string
	FileName string
	Timestamp time.Time
}

func New(c *config.Config) (pipeline.Processor, error) {
	cfg := Config{
	}

	if err := c.Unpack(&cfg); err != nil {
		log.Error(err)
		return nil, fmt.Errorf("failed to unpack the configuration of flow_runner processor: %s", err)
	}
	runner:= IndexBackupProcessor{config: &cfg}
	runner.skipFiles=hashset.New()
	runner.skipFiles.Add("write.lock")
	runner.skipFiles.Add(".DS_Store")

	runner.changes=map[string]*ChangedItem{}
	runner.changesChan=make(chan *ChangedItem,100)

	var err error
	runner.watch, err = fsnotify.NewWatcher();
	if err != nil {
		panic(err)
	}

	global.RegisterShutdownCallback(func() {
		if runner.watch!=nil{
			runner.watch.Close()
		}
	})

	//event listener
	go func() {
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
					log.Errorf("error in index_backup [%v]", v)
				}
			}
		}()

		for {
			select {
			case ev := <-runner.watch.Events:
				{
					segmentID:=ParseSegmentID(filepath.Base(ev.Name))
					log.Trace("File: ", ev.Name,",",ev.Op,",segmentID:",segmentID)
					if ev.Op&fsnotify.Write == fsnotify.Write{
						//async, merge events, upload after file idle for 5 seconds
						runner.changesChan<- &ChangedItem{
							SegmentID: segmentID,
							FilePath: ev.Name,
							Timestamp: time.Now(),
						}
					}
				}
			case err := <-runner.watch.Errors:
				{
					log.Error("error : ", err)
					return
				}
			}
		}
	}()

	//upload checker
	go func() {
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
					log.Errorf("error in index_backup [%v]", v)
				}
			}
		}()

		timer:=util.AcquireTimer(5*time.Second)
		for{
			select {
			case item:=<-runner.changesChan:
				runner.changes[item.FilePath]=item
				break
			case <-timer.C:
				timer.Reset(5*time.Second)
				temp:=map[string]*ChangedItem{}
				for k,v:=range runner.changes{
					log.Trace("processing: ",k)
					if time.Since(v.Timestamp)>5*time.Second{
						log.Debug("upload file after idle>5 seconds: ",k)
						ok,err:=runner.uploadFile(v)
						if err!=nil{
							panic(err)
						}
						if ok{
							runner.updateLastFileUploadedTimestamp(v.IndexUUID,v.FileName,v.Timestamp.Unix())
						}
					}else{
						temp[k]=v
					}
				}
				runner.changes= temp
				break
			}
		}

	}()

	return &runner,nil
}

func (processor IndexBackupProcessor) Stop() error {
	signalChannel <- true
	return nil
}

func (processor *IndexBackupProcessor) Name() string {
	return "index_backup"
}

func (processor *IndexBackupProcessor) Process(ctx *pipeline.Context) error {
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
				log.Errorf("error in processor [%v], [%v]",processor.Name(), v)
				ctx.Failed()
			}
		}
	}()

	//log.Error("processing index backup")

	//#on console
	//get cluster and index uuid
	//get which nodes from routing table
	//check if agent is installed

	meta:=elastic.GetMetadata(processor.config.Elasticsearch)
	if meta==nil{
		panic(errors.Errorf("metadata for: %v was not found",processor.config.Elasticsearch))
	}

	indices,err:=elastic.GetClient(processor.config.Elasticsearch).GetIndices(processor.config.Index)
	if err!=nil{
		panic(err)
	}
	pathToWatch:=map[string]string{}

	for indexName,indexInfo:=range *indices{

		shardsTables,err:=meta.GetIndexPrimaryShardsRoutingTable(indexName)
		if err!=nil{
			panic(err)
		}

		for _,v:=range shardsTables{
			//log.Error("node:",v.Node,",index:",v.Index,",shard:",v.Shard,",state:",v.State,",primary:",v.Primary)
			nodeInfo:=meta.GetNodeInfo(v.Node)
			//log.Error(v.Node,",",nodeInfo)
			if nodeInfo!=nil{
				//log.Error("node:",nodeInfo.Name,",ip:",nodeInfo.Ip,",",nodeInfo.Settings)
				path1,ok:=nodeInfo.Settings["path"].(map[string]interface{})
				//log.Error(path,",",ok)
				if ok{
					home,ok:=path1["home"]
					if ok{
						//log.Error("home path:",home)
						ips:=util.GetLocalIPs()
						for _,ip:=range ips{
							//log.Error("checking ip:",ip," vs ",nodeInfo.Ip)
							if nodeInfo.Ip==ip||nodeInfo.Ip=="127.0.0.1"{
								path:=path2.Join(util.ToString(home),
									"/data/nodes/0/indices/",
									indexInfo.ID,
									util.ToString(v.Shard),
									"index")

								processor.initialCloneIndex(indexInfo.ID,path)
								pathToWatch[indexInfo.ID]=path
								break
							}
						}
					}
				}
			}
		}
	}




	//#on agent
	//each node should find shard's location
	//upload file to s3 and enable new files watch

	//upload checker
	timer:=util.AcquireTimer(60*time.Second)
	for{
		select {
		case <-ctx.Context.Done():
			return nil
		case <-timer.C:
			timer.Reset(60*time.Second)
			for indexUUID,path:=range pathToWatch{
				processor.initialCloneIndex(indexUUID,path)
			}
			break
		}
	}
	return nil
}

func (processor *IndexBackupProcessor)updateLastFileUploadedTimestamp(uuid,file string,timestamp int64){
	if uuid!=""{
		err:=kv.AddValue("index_backup_last_access_timestamp",[]byte(uuid+"__"+file),util.Int64ToBytes(timestamp))
		if err!=nil{
			panic(err)
		}
	}
}

func (processor *IndexBackupProcessor)getLastFileUploadedTimestamp(uuid,file string)int64{
	var lastAccessTime int64
	if uuid!=""{
		lastAccessBytes,err:=kv.GetValue("index_backup_last_access_timestamp",[]byte(uuid+"__"+file))
		if err==nil&&len(lastAccessBytes)>0{
			lastAccessTime=util.BytesToInt64(lastAccessBytes)
		}
	}
	return lastAccessTime
}

func (processor *IndexBackupProcessor) initialCloneIndex(indexUUID,filePath string) {

	log.Debug("scanning index folder:",filePath)
	if !util.FileExists(filePath){
		log.Error("index folder not exists:",filePath)
		return
	}

	filepath.Walk(filePath, func(file string, info os.FileInfo, err error) error {

		if info==nil{
			return nil
		}

		if !info.IsDir(){
			fileName:=info.Name()
			fileAccessTimestamp:=processor.getLastFileUploadedTimestamp(indexUUID,fileName)
			if info.ModTime().Unix()<=fileAccessTimestamp{
				log.Tracef("old file: %v, already uploaded, skip processing",info.Name())
				return nil
			}
			if processor.skipFiles.Contains(fileName)||
				util.PrefixStr(fileName,".")||
				util.SuffixStr(fileName,".tmp"){
				return nil
			}

			processor.changesChan<- &ChangedItem{
				SegmentID: ParseSegmentID(fileName),
				FilePath: file,
				FileName: fileName,
				Timestamp: info.ModTime(),
			}
		}
		return nil
	})

	processor.AddFileToWatch(filePath)
}

func (processor *IndexBackupProcessor) AddFileToWatch(path string){
	if processor.watch!=nil{
		err := processor.watch.Add(path);
		if err != nil {
			log.Error(err)
		}
	}
}

func (processor *IndexBackupProcessor) uploadFile(evt *ChangedItem)(bool,error) {
	path:=evt.FilePath
	if !util.FileExists(path){
		return false,nil
	}

	objName:=util.TrimLeftStr(path,"/Users/medcl/Downloads/elasticsearch-7.9.0/data/nodes/0")
	log.Trace("processing file upload:",path,"=>",objName)

	if processor.config.UploadToFSRepo{
		dstPath:=global.Env().GetDataDir()+"/backup/"+objName
		if processor.config.LocalRepoPath!=""{
			dstPath=processor.config.LocalRepoPath+objName
		}
		dir:=filepath.Dir(dstPath)
		if !util.FileExists(dir){
			err:=os.MkdirAll(dir,0777)
			if err!=nil{
				panic(err)
			}
		}
		_,err:=util.CopyFile(path,dstPath)
		return true,err
	}

	if processor.config.UploadToS3{
		return s3.SyncUpload(path,processor.config.S3.Server,
			processor.config.S3.Location,
			processor.config.S3.Bucket,
			objName)
	}
	return false, nil
}
