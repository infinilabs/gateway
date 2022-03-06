/* Copyright © INFINI Ltd. All rights reserved.
 * web: https://infinilabs.com
 * mail: hello#infini.ltd */

package index_backup

import (
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/s3"
	"infini.sh/framework/core/util"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"github.com/emirpasic/gods/sets/hashset"
	"github.com/fsnotify/fsnotify"
	"strings"
	"time"
)

type Config struct {
	Index string `config:"index"`
}

type IndexBackupProcessor struct {
	config    *Config
	skipFiles *hashset.Set
	watch     *fsnotify.Watcher
}

var signalChannel = make(chan bool, 1)

func init()  {
	pipeline.RegisterProcessorPlugin("index_backup", New)
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

	go func() {
		for {
			select {
			case ev := <-runner.watch.Events:
				{
					log.Error("File: ", ev.Name,",",ev.Op)
					//if ev.Op&fsnotify.Create == fsnotify.Create {
					//	log.Println("创建文件 : ", ev.Name)
					//}
					//if ev.Op&fsnotify.Write == fsnotify.Write {
					//	log.Println("写入文件 : ", ev.Name)
					//}
					//if ev.Op&fsnotify.Remove == fsnotify.Remove {
					//	log.Println("删除文件 : ", ev.Name)
					//}
					//if ev.Op&fsnotify.Rename == fsnotify.Rename {
					//	log.Println("重命名文件 : ", ev.Name)
					//}
					//if ev.Op&fsnotify.Chmod == fsnotify.Chmod {
					//	log.Println("修改权限 : ", ev.Name)
					//}
				}
			case err := <-runner.watch.Errors:
				{
					log.Error("error : ", err)
					return
				}
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

	//#on console
	//get cluster and index uuid
	//get which nodes from routing table
	//check if agent is installed

	//#on agent
	//each node should find shard's location
	//upload file to s3 and enable new files watch

	//processor.config.Index


	path:="/Users/medcl/Downloads/elasticsearch-7.9.0/data/nodes/0/indices/TYCNuC2oQHGBhdV_l4-urg/0/index"

	processor.cloneIndex(path,nil)

	select {
	case <-ctx.Context.Done():
		return nil
	}

	return nil
}

func (processor *IndexBackupProcessor) cloneIndex(filePath string,lastTime *time.Time)*time.Time  {

	time:=time.Now()
	filepath.Walk(filePath, func(file string, info os.FileInfo, err error) error {

		if info==nil{
			return nil
		}

		dir:=path.Dir(file)
		if !info.IsDir(){
			fileName:=info.Name()
			if processor.skipFiles.Contains(fileName)||
				util.PrefixStr(fileName,".")||
				util.SuffixStr(fileName,".tmp"){
				log.Warn(info.Name()," skipping")
				return nil
			}

			arr:=strings.Split(fileName,".")
			if len(arr)==2{
				name:=arr[0]
				//ext:=arr[1]
				if util.FileExists(path.Join(dir,name+".cfs"))||util.FileExists(path.Join(dir,name+".cfe")){
					ok,err:=processor.uploadFile(file)
					if !ok||err!=nil{
						log.Error(ok,err)
						return errors.Errorf("error on uploading to s3: %v",err)
					}
				}
			}

		}

		return nil
	})

	processor.AddFileToWatch(filePath)

	return &time
}

func (processor *IndexBackupProcessor) AddFileToWatch(path string){
	if processor.watch!=nil{
		err := processor.watch.Add(path);
		if err != nil {
			log.Error(err)
		}
	}
}

func (processor *IndexBackupProcessor) uploadFile(path string)(bool,error) {
	objName:=util.TrimLeftStr(path,"/Users/medcl/Downloads/elasticsearch-7.9.0/data/nodes/0")
	log.Debug("processing file:",path,"=>",objName)

	_,err:=util.CopyFile(path,global.Env().GetDataDir()+"/backup/"+objName)
	return true,err

	return s3.SyncUpload(path,"my_blob_store",
		"cn-beijing-001",
		"infini-store",
		objName)
}
