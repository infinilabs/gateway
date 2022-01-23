package proxy

import (
	"infini.sh/framework/core/api"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/util"
	api2 "infini.sh/gateway/api"
	"net/http"
	"path"
)

func (this *GatewayModule) registerAPI(prefix string) {
	if prefix == "" {
		prefix = api2.DefaultAPIPrefix
	}

	api.HandleAPIMethod(api.GET, path.Join("/", prefix, "/entry/stats"), this.getEntries)
	api.HandleAPIMethod(api.POST, path.Join("/", prefix, "/entry/:id/_start"), this.startEntry)
	api.HandleAPIMethod(api.POST, path.Join("/", prefix, "/entry/:id/_stop"), this.stopEntry)
}
func (this *GatewayModule) getEntries(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	data := util.MapStr{}
	for k, v := range this.entryPoints {
		data[k] = v.Stats()
	}
	this.WriteJSON(w, data, 200)
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
