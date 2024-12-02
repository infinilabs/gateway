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

package redis_pubsub

import (
	"context"
	"fmt"

	log "github.com/cihub/seelog"
	"github.com/go-redis/redis/v8"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/lib/bytebufferpool"
	"infini.sh/framework/lib/fasthttp"
)

type RedisPubSub struct {
	Request  bool   `config:"request"`
	Response bool   `config:"response"`
	Channel  string `config:"channel"`
	Host     string `config:"host"`
	Password string `config:"password"`
	Port     int    `config:"port"`
	Db       int    `config:"db"`

	client *redis.Client
}

func (filter *RedisPubSub) Name() string {
	return "redis_pubsub"
}

func (filter *RedisPubSub) Filter(ctx *fasthttp.RequestCtx) {

	buffer := bytebufferpool.Get("redis_pubsub")
	defer bytebufferpool.Put("redis_pubsub", buffer)

	if filter.Request {
		data := ctx.Request.Encode()
		buffer.Write(data)
	}

	if filter.Response {
		data := ctx.Response.Encode()
		buffer.Write(data)
	}

	if buffer.Len() > 0 {
		v, err := filter.client.Publish(ctx, filter.Channel, buffer.Bytes()).Result()
		if global.Env().IsDebug {
			log.Trace(v, err)
		}
		if err != nil {
			panic(err)
		}
	}

}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("redis_pubsub", pipeline.FilterConfigChecked(NewRedisPubSub, pipeline.RequireFields("channel")), &RedisPubSub{})
}

func NewRedisPubSub(c *config.Config) (pipeline.Filter, error) {

	runner := RedisPubSub{
		Request:  true,
		Response: true,
		Host:     "localhost",
		Port:     6379,
		Db:       0,
	}
	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	runner.client = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%v", runner.Host, runner.Port),
		Password: runner.Password,
		DB:       runner.Db,
	})

	var ctx = context.Background()
	_, err := runner.client.Ping(ctx).Result()
	if err != nil {
		panic(err)
	}

	return &runner, nil
}
