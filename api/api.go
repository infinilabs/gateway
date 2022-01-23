package api

import (
	"infini.sh/framework/core/api"
	"infini.sh/framework/core/orm"
	 "infini.sh/gateway/common"
	"path"
)

type GatewayAPI struct {
	api.Handler
}

const DefaultAPIPrefix = "gateway"


func (this *GatewayAPI) RegisterAPI(prefix string) {
	err:=orm.RegisterSchemaWithIndexName(common.EntryConfig{},"entrypoint")
	if err!=nil{
		panic(err)
	}

	if prefix == "" {
		prefix = DefaultAPIPrefix
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


}
