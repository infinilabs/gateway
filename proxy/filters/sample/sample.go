package sample

import (
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/lib/fasthttp"
	"math/rand"
	"sync"
)

type SampleFilter struct {
	Ratio    float32 `config:"ratio"`
	randPool *sync.Pool
}

func (filter *SampleFilter) Name() string {
	return "sample"
}

func (filter *SampleFilter) Filter(ctx *fasthttp.RequestCtx) {

	v := int(filter.Ratio * 100)
	seeds := filter.randPool.Get().(*rand.Rand)
	defer filter.randPool.Put(seeds)

	r := seeds.Intn(100)

	if global.Env().IsDebug {
		log.Debugf("check sample rate [%v] of [%v]", r, v)
	}

	if r <= v {
		if global.Env().IsDebug {
			log.Debugf("this request is lucky to continue: [%v] of [%v], %v", r, v, ctx.URI().String())
		}
		return
	}
	ctx.Finished()

}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("sample",NewSampleFilter,&SampleFilter{})
}

func NewSampleFilter(c *config.Config) (pipeline.Filter, error) {

	runner := SampleFilter{
		Ratio: 0.1,
		randPool: &sync.Pool{
			New: func() interface{} {
				return rand.New(rand.NewSource(100))
			},
		},
	}
	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	return &runner, nil
}
