/* ©INFINI, All Rights Reserved.
 * mail: contact#infini.ltd */

package routing

import (
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/param"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/gateway/common"
	"math/rand"
	log "github.com/cihub/seelog"
	"sync"
)

type RatioRoutingFlowFilter struct {
	param.Parameters
}

func (filter RatioRoutingFlowFilter) Name() string {
	return "ratio"
}

var randPool *sync.Pool

func initPool() {
	if randPool!=nil{
		return
	}
	randPool = &sync.Pool {
		New: func()interface{} {
			return rand.New(rand.NewSource(100))
		},
	}
}

func (filter RatioRoutingFlowFilter) Process(ctx *fasthttp.RequestCtx) {

	initPool()

	ratio:=filter.GetFloat32OrDefault("ratio",0.1)
	continueAfterMatch :=filter.GetBool("continue",false)

	v:=int(ratio*100)

	seeds:=randPool.Get().(*rand.Rand)
	defer randPool.Put(seeds)

	r:=seeds.Intn(100)

	if global.Env().IsDebug{
		log.Debugf("split traffic, check [%v] of [%v]",r,v)
	}

	if  r <= v{
		ctx.Resume()
		flowName:=filter.MustGetString("flow")
		flow:=common.MustGetFlow(flowName)
		if global.Env().IsDebug{
			log.Debugf("request [%v] go on flow: [%s]",ctx.URI().String(),flow.ToString())
		}
		flow.Process(ctx)
		if !continueAfterMatch {
			ctx.Finished()
		}
	}

}
