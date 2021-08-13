package redis

import (
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/param"
	"infini.sh/framework/lib/bytebufferpool"
	"infini.sh/framework/lib/fasthttp"
	"src/github.com/go-redis/redis"
	"sync"
)


var client *redis.Client
var l sync.RWMutex
var inited bool

func (p RedisOutput) getRedisClient() *redis.Client {

	if client != nil {
		return client
	}

	l.Lock()
	defer l.Unlock()

	if client != nil {
		return client
	}

	client = redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%v", p.GetStringOrDefault("host","localhost"), p.GetIntOrDefault("port",6379)),
		Password: p.GetStringOrDefault("password",""),
		DB:       p.GetIntOrDefault("db",0),
	})

	_, err := client.Ping().Result()
	if err != nil {
		panic(err)
	}

	return client
}

type RedisOutput struct {
	param.Parameters
}

func (filter RedisOutput) Name() string {
	return "redis"
}

func (filter RedisOutput) Process(ctx *fasthttp.RequestCtx) {

	buffer:=bytebufferpool.Get()

	if filter.GetBool("request",true){
		data := ctx.Request.Encode()
		buffer.Write(data)
	}

	if filter.GetBool("response",true){
		data:=ctx.Response.Encode()
		buffer.Write(data)
	}

	if buffer.Len()>0{
		v,err:=filter.getRedisClient().Publish(filter.MustGetString("channel"),buffer.Bytes()).Result()
		if global.Env().IsDebug{
			log.Trace(v,err)
		}
		if err!=nil{
			panic(err)
		}
		buffer.Reset()
		bytebufferpool.Put(buffer)
	}

}

