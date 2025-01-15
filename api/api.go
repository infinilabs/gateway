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

package api

import (
	"infini.sh/framework/core/api"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/orm"
	"infini.sh/gateway/common"
	"path"
)

type GatewayAPI struct {
	api.Handler
}

func (this *GatewayAPI) RegisterSchema() {
	orm.MustRegisterSchemaWithIndexName(common.EntryConfig{}, "entry")
	orm.MustRegisterSchemaWithIndexName(common.RouterConfig{}, "router")
	orm.MustRegisterSchemaWithIndexName(common.FlowConfig{}, "flow")

}

func (this *GatewayAPI) RegisterAPI(prefix string) {

	if prefix == "" {
		prefix = global.Env().GetAppLowercaseName()
	}

	api.HandleAPIMethod(api.POST, path.Join("/", prefix, "/entry"), this.createEntry)
	api.HandleAPIMethod(api.GET, path.Join("/", prefix, "/entry/:entry_id"), this.getEntry)
	api.HandleAPIMethod(api.PUT, path.Join("/", prefix, "/entry/:entry_id"), this.updateEntry)
	api.HandleAPIMethod(api.DELETE, path.Join("/", prefix, "/entry/:entry_id"), this.deleteEntry)
	api.HandleAPIMethod(api.GET, path.Join("/", prefix, "/entry/_search"), this.searchEntry)

	api.HandleAPIMethod(api.POST, path.Join("/", prefix, "/router"), this.createRouter)
	api.HandleAPIMethod(api.GET, path.Join("/", prefix, "/router/:router_id"), this.getRouter)
	api.HandleAPIMethod(api.PUT, path.Join("/", prefix, "/router/:router_id"), this.updateRouter)
	api.HandleAPIMethod(api.DELETE, path.Join("/", prefix, "/router/:router_id"), this.deleteRouter)
	api.HandleAPIMethod(api.GET, path.Join("/", prefix, "/router/_search"), this.searchRouter)

	api.HandleAPIMethod(api.GET, path.Join("/", prefix, "/filter/metadata"), this.getFlowFilters)

	api.HandleAPIMethod(api.POST, path.Join("/", prefix, "/flow"), this.createFlow)
	api.HandleAPIMethod(api.GET, path.Join("/", prefix, "/flow/:flow_id"), this.getFlow)
	api.HandleAPIMethod(api.PUT, path.Join("/", prefix, "/flow/:flow_id"), this.updateFlow)
	api.HandleAPIMethod(api.DELETE, path.Join("/", prefix, "/flow/:flow_id"), this.deleteFlow)
	api.HandleAPIMethod(api.GET, path.Join("/", prefix, "/flow/_search"), this.searchFlow)

}
