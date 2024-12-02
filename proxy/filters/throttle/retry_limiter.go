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

package throttle

import (
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
	"time"
)

type RetryLimiter struct {
	MaxRetryTimes int      `config:"max_retry_times"`
	SleepInterval int      `config:"retry_interval_in_ms"`
	Queue         string   `config:"queue_name"`
	TagsOnSuccess []string `config:"tag_on_success"` //hit limiter then add tag
}

func (filter *RetryLimiter) Name() string {
	return "retry_limiter"
}

const RetryKey = "Retried_times"

func (filter *RetryLimiter) Filter(ctx *fasthttp.RequestCtx) {

	timeBytes := ctx.Request.Header.Peek(RetryKey)
	times := 0
	if timeBytes != nil {
		t, err := util.ToInt(string(timeBytes))
		if err == nil {
			times = t
		}
	}
	if global.Env().IsDebug{
		log.Debugf("retry times: %v > %v",times,filter.MaxRetryTimes)
	}

	if times > filter.MaxRetryTimes {
		log.Debugf("hit max retry times: %v > %v",times,filter.MaxRetryTimes)
		ctx.Finished()
		ctx.Request.Header.Del(RetryKey)
		queue.Push(queue.GetOrInitConfig(filter.Queue), ctx.Request.Encode())
		time.Sleep(time.Duration(filter.SleepInterval) * time.Millisecond)
		if len(filter.TagsOnSuccess)>0{
			ctx.UpdateTags(filter.TagsOnSuccess,nil)
		}
		return
	}

	times++
	ctx.Request.Header.Set(RetryKey, util.IntToString(times))
}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("retry_limiter",pipeline.FilterConfigChecked(NewRetryLimiter, pipeline.RequireFields("queue_name")),&RetryLimiter{})
}

func NewRetryLimiter(c *config.Config) (pipeline.Filter, error) {

	runner := RetryLimiter{
		MaxRetryTimes: 3,
		SleepInterval: 1000,
	}
	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	return &runner, nil
}
