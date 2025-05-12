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

/* ©INFINI.LTD, All Rights Reserved.
 * mail: hello#infini.ltd */

package elastic

import (
	"fmt"
	"github.com/OneOfOne/xxhash"
	log "github.com/cihub/seelog"
	"infini.sh/framework/lib/fasttemplate"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/param"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
	"io"
	"strings"
)

type HashModFilter struct {
	Source              string `config:"source" `
	TargetContextKey    string `config:"target_context_name" `
	PartitionSize       int    `config:"mod"`
	template            *fasttemplate.Template
	partitionSizeStr    string
	AddToRequestHeader  bool `config:"add_to_request_header" type:"bool" default_value:"true"`
	AddToResponseHeader bool `config:"add_to_response_header" type:"bool" default_value:"true"`
}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("hash_mod", NewHashModFilter, &HashModFilter{})
}

func NewHashModFilter(c *config.Config) (pipeline.Filter, error) {

	runner := HashModFilter{
		TargetContextKey:   "partition_id",
		AddToRequestHeader: true,
	}

	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	var err error
	if strings.Contains(runner.Source, "$[[") {
		runner.template, err = fasttemplate.NewTemplate(runner.Source, "$[[", "]]")
		if err != nil {
			panic(err)
		}
	}

	runner.partitionSizeStr = fmt.Sprintf("%d", runner.PartitionSize)

	return &runner, nil
}

func (filter *HashModFilter) Name() string {
	return "hash_mod"
}

func (filter *HashModFilter) Filter(ctx *fasthttp.RequestCtx) {

	str := filter.Source

	var idStr string
	if filter.PartitionSize == 1 {
		idStr = "0"
	} else {
		if filter.template != nil {
			str = filter.template.ExecuteFuncString(func(w io.Writer, tag string) (int, error) {
				variable, err := ctx.GetValue(tag)
				if err == nil {
					return w.Write([]byte(util.ToString(variable)))
				}
				return -1, err
			})
		}

		if str != "" {

			xxHash := xxHashPool.Get().(*xxhash.XXHash32)
			xxHash.Reset()
			xxHash.WriteString(str)
			partitionID := int(xxHash.Sum32()) % filter.PartitionSize

			idStr = fmt.Sprintf("%d", partitionID)
			xxHashPool.Put(xxHash)

			if global.Env().IsDebug {
				log.Debug("hash_mod filter: ", filter.Name(), ", input:", str, ", partition_id: ", idStr, ", partition_size: ", filter.partitionSizeStr)
			}
		}
	}

	ctx.Set(param.ParaKey(filter.TargetContextKey), idStr)

	if filter.AddToRequestHeader {
		ctx.Request.Header.Set("X-Partition-ID", idStr)
		ctx.Request.Header.Set("X-Partition-Size", filter.partitionSizeStr)
	}

	if filter.AddToResponseHeader {
		ctx.Response.Header.Set("X-Partition-ID", idStr)
		ctx.Response.Header.Set("X-Partition-Size", filter.partitionSizeStr)
	}

}
