/*
Copyright Medcl (m AT medcl.net)

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

package sample

import (
	"infini.sh/framework/core/global"
	"math/rand"
	"infini.sh/framework/core/param"
	"infini.sh/framework/lib/fasthttp"
	log "github.com/cihub/seelog"
)

type SampleFilter struct {
	param.Parameters
}


func (filter SampleFilter) Name() string {
	return "sample"
}
var seeds=rand.New(rand.NewSource(50))

func (filter SampleFilter) Process(ctx *fasthttp.RequestCtx) {
	ratio:=filter.GetFloat32OrDefault("ratio",0.1)
	v:=int(ratio*100)
	r:=seeds.Intn(100)

	if global.Env().IsDebug{
		log.Debugf("check sample rate [%v] of [%v]",r,v)
	}

	if  r <= v{
		if global.Env().IsDebug{
			log.Debugf("this request is lucky to continue: [%v] of [%v], %v",r,v,ctx.URI().String())
		}
		return
	}
	ctx.Finished()
}
