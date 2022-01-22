package proxy

import (
	"fmt"
	"infini.sh/framework/core/api"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
	common2 "infini.sh/gateway/common"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"
)

type GatewayAPI struct {
	api.Handler
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

func (h *GatewayAPI) createEntry(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	var conf =&common2.EntryConfig{}
	err := h.DecodeJSON(req, conf)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	conf.Id=util.GetUUID()
	conf.Created=time.Now()
	conf.Updated=time.Now()
	err=orm.Save(conf)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.WriteJSON(w,util.MapStr{
		"_id":conf.Id,
		"result": "created",
	},200)

}

func (h *GatewayAPI) getEntry(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.ByName("entry_id")

	if id==""{
		h.WriteError(w,"id was not set",500)
		return
	}

	obj:=common2.EntryConfig{}
	obj.Id=id

	err:=orm.Get(&obj)
	if err != nil {
		h.WriteJSON(w, util.MapStr{
			"_id":id,
			"found" : false,
		}, http.StatusNotFound)
		return
	}

	h.WriteJSON(w,util.MapStr{
		"found": true,
		"_id":id,
		"_source" : obj,

	},200)
}

func (h *GatewayAPI) updateEntry(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.ByName("entry_id")

	if id==""{
		h.WriteError(w,"id was not set",500)
		return
	}

	obj:=common2.EntryConfig{}
	err := h.DecodeJSON(req, &obj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	obj.Id=id
	//TODO handle created, version history
	obj.Updated=time.Now()
	err=orm.Save(&obj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.WriteJSON(w,util.MapStr{
		"_id":obj.Id,
		"result": "updated",
	},200)
}

func (h *GatewayAPI) deleteEntry(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.ByName("entry_id")

	if id==""{
		h.WriteError(w,"id was not set",500)
		return
	}

	obj:=common2.EntryConfig{}
	obj.Id=id

	err:=orm.Get(&obj)
	if err != nil {
		h.WriteJSON(w, util.MapStr{
			"_id":id,
			"result" : "not_found",
		}, http.StatusNotFound)
		return
	}

	err=orm.Delete(&obj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.WriteJSON(w,util.MapStr{
		"_id":obj.Id,
		"result": "deleted",
	},200)
}

func (h *GatewayAPI) searchEntry(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	var (
		name        = h.GetParameterOrDefault(req, "name", "")
		queryDSL    = `{"query":{"bool":{"must":[%s]}}, "size": %d, "from": %d}`
		strSize     = h.GetParameterOrDefault(req, "size", "20")
		strFrom     = h.GetParameterOrDefault(req, "from", "0")
		mustBuilder = &strings.Builder{}
	)
	if name != "" {
		mustBuilder.WriteString(fmt.Sprintf(`{"prefix":{"name.text": "%s"}}`, name))
	}
	size, _ := strconv.Atoi(strSize)
	if size <= 0 {
		size = 20
	}
	from, _ := strconv.Atoi(strFrom)
	if from < 0 {
		from = 0
	}

	q:=orm.Query{}
	queryDSL = fmt.Sprintf(queryDSL, mustBuilder.String(), size, from)
	q.RawQuery=[]byte(queryDSL)

	err,res := orm.Search(&common2.EntryConfig{},&q)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.Write(w, res.Raw)
}

const defaultPrefix = "gateway"

func (this *GatewayModule) registerAPI(prefix string) {
	if prefix == "" {
		prefix = defaultPrefix
	}

	api.HandleAPIMethod(api.GET, path.Join("/", prefix, "/entry/stats"), this.getEntries)
	api.HandleAPIMethod(api.POST, path.Join("/", prefix, "/entry/:id/_start"), this.startEntry)
	api.HandleAPIMethod(api.POST, path.Join("/", prefix, "/entry/:id/_stop"), this.stopEntry)
}

func (this *GatewayAPI) registerAPI(prefix string) {
	err:=orm.RegisterSchemaWithIndexName(common2.EntryConfig{},"entrypoint")
	if err!=nil{
		panic(err)
	}

	if prefix == "" {
		prefix = defaultPrefix
	}

	api.HandleAPIMethod(api.POST, path.Join("/", prefix, "/entry"), this.createEntry)
	api.HandleAPIMethod(api.GET, path.Join("/", prefix, "/entry/:entry_id"), this.getEntry)
	api.HandleAPIMethod(api.PUT, path.Join("/", prefix, "/entry/:entry_id"), this.updateEntry)
	api.HandleAPIMethod(api.DELETE, path.Join("/", prefix, "/entry/:entry_id"), this.deleteEntry)
	api.HandleAPIMethod(api.GET, path.Join("/", prefix, "/entry/_search"), this.searchEntry)
}
