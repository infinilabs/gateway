package api

import (
	"fmt"
	"infini.sh/framework/core/api/router"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
	"infini.sh/gateway/common"
	"net/http"
	"strconv"
	"strings"
)

func (h *GatewayAPI) createFlow(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	var obj = &common.FlowConfig{}
	err := h.DecodeJSON(req, obj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = orm.Create(obj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.WriteJSON(w, util.MapStr{
		"_id":    obj.ID,
		"result": "created",
	}, 200)

}

func (h *GatewayAPI) getFlow(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.MustGetParameter("flow_id")

	obj := common.FlowConfig{}
	obj.ID = id

	exists, err := orm.Get(&obj)
	if !exists || err != nil {
		h.WriteJSON(w, util.MapStr{
			"_id":   id,
			"found": false,
		}, http.StatusNotFound)
		return
	}

	h.WriteJSON(w, util.MapStr{
		"found":   true,
		"_id":     id,
		"_source": obj,
	}, 200)
}

func (h *GatewayAPI) updateFlow(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.MustGetParameter("flow_id")
	obj := common.FlowConfig{}

	obj.ID = id
	exists, err := orm.Get(&obj)
	if !exists || err != nil {
		h.WriteJSON(w, util.MapStr{
			"_id":    id,
			"result": "not_found",
		}, http.StatusNotFound)
		return
	}

	id = obj.ID
	create := obj.Created
	err = h.DecodeJSON(req, &obj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//protect
	obj.ID = id
	obj.Created = create

	err = orm.Update(&obj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.WriteJSON(w, util.MapStr{
		"_id":    obj.ID,
		"result": "updated",
	}, 200)
}

func (h *GatewayAPI) deleteFlow(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.MustGetParameter("flow_id")

	obj := common.FlowConfig{}
	obj.ID = id

	exists, err := orm.Get(&obj)
	if !exists || err != nil {
		h.WriteJSON(w, util.MapStr{
			"_id":    id,
			"result": "not_found",
		}, http.StatusNotFound)
		return
	}

	err = orm.Delete(&obj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.WriteJSON(w, util.MapStr{
		"_id":    obj.ID,
		"result": "deleted",
	}, 200)
}

func (h *GatewayAPI) searchFlow(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {

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

	q := orm.Query{}
	queryDSL = fmt.Sprintf(queryDSL, mustBuilder.String(), size, from)
	q.RawQuery = []byte(queryDSL)

	err, res := orm.Search(&common.FlowConfig{}, &q)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.Write(w, res.Raw)
}

func (h *GatewayAPI) getFlowFilters(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {

	//var (
	//	name        = h.GetParameterOrDefault(req, "name", "")
	//	queryDSL    = `{"query":{"bool":{"must":[%s]}}, "size": %d, "from": %d}`
	//	strSize     = h.GetParameterOrDefault(req, "size", "20")
	//	strFrom     = h.GetParameterOrDefault(req, "from", "0")
	//	mustBuilder = &strings.Builder{}
	//)
	//if name != "" {
	//	mustBuilder.WriteString(fmt.Sprintf(`{"prefix":{"name.text": "%s"}}`, name))
	//}
	//size, _ := strconv.Atoi(strSize)
	//if size <= 0 {
	//	size = 20
	//}
	//from, _ := strconv.Atoi(strFrom)
	//if from < 0 {
	//	from = 0
	//}
	//
	//q := orm.Query{}
	//queryDSL = fmt.Sprintf(queryDSL, mustBuilder.String(), size, from)
	//q.RawQuery = []byte(queryDSL)
	//
	//err, res := orm.Search(&common.FlowConfig{}, &q)
	//if err != nil {
	//	h.WriteError(w, err.Error(), http.StatusInternalServerError)
	//	return
	//}

	data := []byte("{\n    \"request_body_json_del\": {\n        \"properties\": {\n            \"path\": {\n                \"type\": \"array\", \n                \"sub_type\": \"string\"\n            }\n        }\n    }, \n    \"request_body_json_set\": {\n        \"properties\": {\n            \"path\": {\n                \"type\": \"array\", \n                \"sub_type\": \"keyvalue\"\n            }\n        }\n    }, \n    \"ldap_auth\": {\n        \"properties\": {\n            \"host\": {\n                \"type\": \"string\", \n                \"default_value\": \"ldap.forumsys.com\"\n            }, \n            \"port\": {\n                \"type\": \"number\", \n                \"default_value\": 389\n            }, \n            \"bind_dn\": {\n                \"type\": \"string\"\n            }, \n            \"bind_password\": {\n                \"type\": \"string\"\n            }, \n            \"base_dn\": {\n                \"type\": \"string\"\n            }, \n            \"user_filter\": {\n                \"type\": \"string\"\n            }\n        }\n    }\n}")

	h.Write(w, data)
}
