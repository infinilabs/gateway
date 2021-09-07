/* ©INFINI, All Rights Reserved.
 * mail: contact#infini.ltd */

package rbac

import (
	"infini.sh/framework/core/param"
	"infini.sh/framework/lib/fasthttp"
)

type RBACFilter struct {
	param.Parameters
}

func (filter RBACFilter) Name() string {
	return "rbac"
}

const AccessDeniedMessage = "Access denied"
const NoUserMessage ="No user found"
const NoRoleMessage ="No roles found"

func (filter RBACFilter) Process(ctx *fasthttp.RequestCtx) {

	action := filter.GetStringOrDefault("action", "allow")

	users,definedUsers := filter.GetStringArray("users")
	if definedUsers{
		currentUsername,ok1:=ctx.GetString("user_name")
		if !ok1{
			ctx.Error(NoUserMessage,401)
			ctx.Finished()
			return
		}
		for _, v := range users {
			if v==currentUsername{
				if action=="allow"{
					filter.checkES(ctx)
					return
				}else{
					ctx.Error(AccessDeniedMessage,403)
					ctx.Finished()
					return
				}
			}
		}
	}

	roles,definedRoles := filter.GetStringArray("roles")
	if definedRoles{
		currentRoles,ok2:=ctx.GetStringArray("user_roles")
		if !ok2{
			ctx.Error(NoRoleMessage,401)
			ctx.Finished()
			return
		}
		for _, v := range roles {
			for _,z:=range currentRoles{
				if v==z{
					if action=="allow"{
						filter.checkES(ctx)
						return
					}else{
						ctx.Error(AccessDeniedMessage,403)
						ctx.Finished()
						return
					}
				}
			}
		}
	}

	defaultAction := filter.GetStringOrDefault("default_action", "deny")
	if defaultAction=="deny"{
		ctx.Error(AccessDeniedMessage,403)
		ctx.Finished()
		return
	}else{
		filter.checkES(ctx)
	}
}

func (filter RBACFilter) checkES(ctx *fasthttp.RequestCtx){
	//check clusters

}

func (filter RBACFilter) checkClusters(ctx *fasthttp.RequestCtx){
	//check indices

}

func (filter RBACFilter) checkIndices(ctx *fasthttp.RequestCtx){
	//check actions

}

func (filter RBACFilter) checkActions(ctx *fasthttp.RequestCtx){

}
