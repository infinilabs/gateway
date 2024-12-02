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

/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package elastic

import (
	"fmt"
	"github.com/OneOfOne/xxhash"
	"github.com/savsgio/gotils/bytes"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
	"strings"
	"sync"
)

type ElasticsearchRequestReshuffle struct {
	Elasticsearch          string   `config:"elasticsearch"`
	TagsOnSuccess          []string `config:"tag_on_success"`
	SkipBulk               bool     `config:"skip_bulk"`
	PartitionSize          int      `config:"partition_size"`
	QueuePrefix            string   `config:"queue_name_prefix"`
	HashFactor             string   `config:"hash_factor"`
	ContinueAfterReshuffle bool     `config:"continue_after_reshuffle"`
	esConfig               *elastic.ElasticsearchConfig
}

func (filter *ElasticsearchRequestReshuffle) Name() string {
	return "request_reshuffle"
}

func (this *ElasticsearchRequestReshuffle) Filter(ctx *fasthttp.RequestCtx) {

	pathStr := util.UnsafeBytesToString(ctx.PhantomURI().Path())

	if this.SkipBulk && util.SuffixStr(pathStr, "/_bulk") {
		return
	}

	path := strings.Split(pathStr, "/")
	if len(path) > 1 {

		qName := this.QueuePrefix
		labels := util.MapStr{}
		labels["type"] = "request_reshuffle"
		labels["elasticsearch"] = this.esConfig.ID

		if this.PartitionSize > 1 {
			xxHash := xxHashPool.Get().(*xxhash.XXHash32)
			defer xxHashPool.Put(xxHash)

			xxHash.Reset()
			xxHash.WriteString(path[1])

			partitionID := int(xxHash.Sum32()) % this.PartitionSize
			qName = fmt.Sprintf("%v##cluster##%v##%v", qName, this.esConfig.ID, partitionID)
			labels["partition"] = partitionID
			labels["partition_size"] = this.PartitionSize
		}

		cfg := queue.AdvancedGetOrInitConfig("", qName, labels)
		data := ctx.Request.Encode()
		err := queue.Push(cfg, bytes.Copy(data))
		if err != nil {
			panic(err)
		}
		ctx.SetDestination(fmt.Sprintf("%v:%v", "queue", qName))

		if len(this.TagsOnSuccess) > 0 {
			ctx.UpdateTags(this.TagsOnSuccess, nil)
		}

		if !this.ContinueAfterReshuffle {
			ctx.Response.Header.Set("X-Request-Reshuffled", "true")
			ctx.Response.SetStatusCode(200)
			ctx.Finished()
		}

	}

}

var xxHashPool = sync.Pool{
	New: func() interface{} {
		return xxhash.New32()
	},
}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("request_reshuffle", NewElasticsearchRequestReshuffleFilter, &ElasticsearchRequestReshuffle{})
}

func NewElasticsearchRequestReshuffleFilter(c *config.Config) (pipeline.Filter, error) {

	runner := ElasticsearchRequestReshuffle{
		SkipBulk:      true,
		PartitionSize: 10,
		QueuePrefix:   "request_reshuffle",
	}

	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	if runner.Elasticsearch == "" {
		panic(errors.New("elasticsearch is required"))
	}

	runner.esConfig = elastic.GetConfig(runner.Elasticsearch)

	if runner.PartitionSize > 1 {

		for i := 0; i < runner.PartitionSize; i++ {
			qName := fmt.Sprintf("%v##cluster##%v##%v", runner.QueuePrefix, runner.esConfig.ID, i)
			labels := util.MapStr{}
			labels["type"] = "request_reshuffle"
			labels["elasticsearch"] = runner.esConfig.ID
			labels["partition"] = i
			labels["partition_size"] = runner.PartitionSize
			queue.RegisterConfig(queue.AdvancedGetOrInitConfig("", qName, labels))
		}
	}

	return &runner, nil
}
