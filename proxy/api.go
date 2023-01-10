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
