// Copyright (C) INFINI Labs & INFINI LIMITED.
//
// The INFINI Framework is offered under the GNU Affero General Public License v3.0
// and as commercial software.
//
// For commercial licensing, contact us at:
//   - Website: infinilabs.com
//   - Email: hello@infini.ltd
//
// Open Source licensed under AGPL V3:
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package sample

import (
	"fmt"
	"math/rand"
	"sync"

	log "github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/lib/fasthttp"
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

	if r < v {
		if global.Env().IsDebug {
			log.Debugf("this request is lucky to continue: [%v] of [%v], %v", r, v, ctx.PhantomURI().String())
		}
		return
	}
	ctx.Finished()

}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("sample", NewSampleFilter, &SampleFilter{})
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
