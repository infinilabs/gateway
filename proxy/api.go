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

package proxy

import (
	"infini.sh/framework/core/api"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/util"
	"infini.sh/gateway/common"
	"net/http"
	"path"
)

func (this *GatewayModule) registerAPI(prefix string) {
	if prefix == "" {
		prefix = global.Env().GetAppLowercaseName()
	}

	api.HandleAPIMethod(api.POST, path.Join("/", prefix, "/entry/:id/_start"), this.startEntry)
	api.HandleAPIMethod(api.POST, path.Join("/", prefix, "/entry/:id/_stop"), this.stopEntry)
	api.HandleAPIMethod(api.GET, path.Join("/", prefix, "/entry/:id"), this.getConfig)
}

func (this *GatewayModule) getConfig(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.ByName("id")
	v, ok := this.entryPoints[id]
	if ok {
		cfg := v.GetConfig()
		data:=util.MapStr{
			"entry":cfg,
			"router":v.GetRouterConfig(),
			"flows":common.GetAllFlows(),
		}

		this.WriteJSON(w, data, 200)
	} else {
		this.Error404(w)
	}
}

func (this *GatewayModule) startEntry(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.ByName("id")
	v, ok := this.entryPoints[id]
	if ok {
		err := v.Start()
		if err != nil {
			this.Error500(w, err.Error())
			return
		}
		this.WriteAckJSON(w, true, 200, nil)
		return
	} else {
		this.Error404(w)
	}
}

func (this *GatewayModule) stopEntry(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.ByName("id")
	v, ok := this.entryPoints[id]
	if ok {
		err := v.Stop()
		if err != nil {
			this.Error500(w, err.Error())
			return
		}
		this.WriteAckJSON(w, true, 200, nil)
		return
	} else {
		this.Error404(w)
	}
}
