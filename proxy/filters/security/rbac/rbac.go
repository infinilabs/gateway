/* ©INFINI, All Rights Reserved.
 * mail: contact#infini.ltd */

package rbac

import (
	"fmt"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/lib/fasthttp"
)

type RBACFilter struct {
	Action        string   `config:"action"`
	Users         []string `config:"users"`
	Roles         []string `config:"roles"`
	DefaultAction string   `config:"default_action"`
}

func (filter *RBACFilter) Name() string {
	return "rbac"
}

const AccessDeniedMessage = "Access denied"
const NoUserMessage = "No user found"
const NoRoleMessage = "No roles found"

func (filter *RBACFilter) Filter(ctx *fasthttp.RequestCtx) {

	if len(filter.Users) > 0 {
		currentUsername, ok1 := ctx.GetString("user_name")
		if !ok1 {
			ctx.Error(NoUserMessage, 401)
			ctx.Finished()
			return
		}
		for _, v := range filter.Users {
			if v == currentUsername {
				if filter.Action == "allow" {
					filter.checkES(ctx)
					return
				} else {
					ctx.Error(AccessDeniedMessage, 403)
					ctx.Finished()
					return
				}
			}
		}
	}

	if len(filter.Roles) > 0 {
		currentRoles, ok2 := ctx.GetStringArray("user_roles")
		if !ok2 {
			ctx.Error(NoRoleMessage, 401)
			ctx.Finished()
			return
		}
		for _, v := range filter.Roles {
			for _, z := range currentRoles {
				if v == z {
					if filter.Action == "allow" {
						filter.checkES(ctx)
						return
					} else {
						ctx.Error(AccessDeniedMessage, 403)
						ctx.Finished()
						return
					}
				}
			}
		}
	}

	if filter.DefaultAction == "deny" {
		ctx.Error(AccessDeniedMessage, 403)
		ctx.Finished()
		return
	} else {
		filter.checkES(ctx)
	}
}

func (filter *RBACFilter) checkES(ctx *fasthttp.RequestCtx) {
	//check clusters

}

func (filter *RBACFilter) checkClusters(ctx *fasthttp.RequestCtx) {
	//check indices

}

func (filter *RBACFilter) checkIndices(ctx *fasthttp.RequestCtx) {
	//check actions

}

func (filter *RBACFilter) checkActions(ctx *fasthttp.RequestCtx) {

}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("rbac",NewRBACFilter,&RBACFilter{})
}

func NewRBACFilter(c *config.Config) (pipeline.Filter, error) {

	runner := RBACFilter{
		Action: "allow",
	}

	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	return &runner, nil
}
