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

package date_range_precision_tuning

import (
	"fmt"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
	log "src/github.com/cihub/seelog"
)

type DatePrecisionTuning struct {
	config *Config
}

type Config struct {
	PathKeywords  []string `config:"path_keywords"`
	TimePrecision int      `config:"time_precision"`
}

var defaultConfig = Config{
	PathKeywords:  builtinKeywords,
	TimePrecision: 4,
}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("date_range_precision_tuning", New, &defaultConfig)
}

func New(c *config.Config) (pipeline.Filter, error) {
	cfg := defaultConfig
	if err := c.Unpack(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	if cfg.TimePrecision > 9 {
		cfg.TimePrecision = 9
	}

	if cfg.TimePrecision < 0 {
		cfg.TimePrecision = 0
	}

	runner := DatePrecisionTuning{config: &cfg}

	return &runner, nil
}

func (this *DatePrecisionTuning) Name() string {
	return "date_range_precision_tuning"
}

var builtinKeywords = []string{"_search", "_async_search"}

func (this *DatePrecisionTuning) Filter(ctx *fasthttp.RequestCtx) {

	if ctx.Request.GetBodyLength() <= 0 {
		return
	}

	path := string(ctx.RequestURI())

	if util.ContainsAnyInArray(path, this.config.PathKeywords) {
		//request normalization
		body := ctx.Request.GetRawBody()
		//0-9: 时分秒微妙 00:00:00:000
		//TODO get time field from index pattern settings
		ok := util.ProcessJsonData(&body, []byte("range"), 150, [][]byte{[]byte("\"gte\""), []byte("\"lte\"")}, false, []byte("\"gte\""), []byte("}"), 128, func(data []byte, start, end int) {
			startProcess := false
			precisionOffset := 0
			matchCount := 0
			block := body[start:end]
			if global.Env().IsDebug {
				log.Debug("body[start:end]: ", string(body[start:end]))
			}

			len := len(block) - 1
			for i, v := range block {
				if i > 1 && i < len {
					left := block[i-1]
					right := block[i+1]

					if global.Env().IsDebug {
						log.Debug(i, ",", string(v), ",", block[i-1], ",", block[i+1])
					}
					if v == 84 && left > 47 && left < 58 && right > 47 && right < 58 { //T
						startProcess = true
						precisionOffset = 0
						matchCount++
						continue
					}
				}

				if startProcess && v > 47 && v < 58 {
					precisionOffset++
					if precisionOffset <= this.config.TimePrecision {
						continue
					} else if precisionOffset > 9 {
						startProcess = false
						continue
					}
					if matchCount == 1 {
						body[start+i] = 48
					} else if matchCount == 2 {
						prev := body[start+i-1]

						if precisionOffset == 1 {
							body[start+i] = 50
							continue
						}

						if precisionOffset == 2 {
							if prev == 48 { //int:0
								body[start+i] = 57
								continue
							}
							if prev == 49 { //int:1
								body[start+i] = 57
								continue
							}
							if prev == 50 { //int:2
								body[start+i] = 51
								continue
							}
						}
						if precisionOffset == 3 {
							body[start+i] = 53
							continue
						}
						if precisionOffset == 4 {
							if global.Env().IsDebug {
								log.Debug("prev: ", prev, ",", prev != 54)
							}
							if prev != 54 { //int:6
								body[start+i] = 57
								continue
							}
						}
						if precisionOffset == 5 {
							body[start+i] = 53
							continue
						}
						if precisionOffset >= 6 {
							body[start+i] = 57
							continue
						}

					}

				}

			}
		})

		if global.Env().IsDebug {
			log.Debug("rewrite success: ", ok, ",", string(body), ",", this.config.TimePrecision)
		}

		if ok {
			ctx.Request.SetRawBody(body)
		}
	}
}
