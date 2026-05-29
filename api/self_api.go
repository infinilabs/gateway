package api

import (
	"net/http"
	"strings"

	log "github.com/cihub/seelog"
	frameworkapi "infini.sh/framework/core/api"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/global"
	frameworksecurity "infini.sh/framework/core/security"
	configcommon "infini.sh/framework/modules/configs/common"
)

var gatewayAdditionalProtectedWebAPIRoutes = []frameworkapi.ProtectedAPIRoute{
	{Method: frameworkapi.GET, Path: "/stats/prometheus"},
	{Method: frameworkapi.GET, Path: "/debug/goroutines"},
	{Method: frameworkapi.GET, Path: "/debug/pool/bytes"},
	{Method: frameworkapi.GET, Path: "/_local/files/_list"},
	{Method: frameworkapi.GET, Path: "/_local/files/:file/_list"},
	{Method: frameworkapi.DELETE, Path: "/_local/files/:file"},
}

type selfAPIHandler struct {
	frameworkapi.Handler
}

func InitSelfAPI() {
	if _, err := configcommon.EnsureTokenInKeystore(configcommon.AgentAccessTokenKeystoreKey); err != nil {
		log.Errorf("failed to initialize gateway access token: %v", err)
	}

	handler := selfAPIHandler{}
	routes := append([]frameworkapi.ProtectedAPIRoute{}, frameworkapi.DefaultProtectedAPIRoutes...)
	routes = append(routes, gatewayAdditionalProtectedWebAPIRoutes...)
	frameworkapi.RegisterProtectedUIRoutes(routes, handler.requireLoginOrAccessToken(handler.proxyLocalAPI), frameworkapi.AllowOPTIONSS(), frameworkapi.Feature(frameworkapi.FeatureCORS))
}

func (h selfAPIHandler) proxyLocalAPI(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	proxyReq := req.Clone(req.Context())
	applyGatewayLocalAPIAuth(proxyReq)
	frameworkapi.ServeRegisteredAPIRequest(w, proxyReq)
}

func (h selfAPIHandler) requireLoginOrAccessToken(next httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
		if frameworkapi.ValidateManagedAccessTokenRequest(req) {
			next(w, req, ps)
			return
		}
		if frameworkapi.IsAuthEnable() {
			user, err := frameworksecurity.ValidateLogin(w, req)
			if err != nil {
				h.WriteError(w, err.Error(), http.StatusUnauthorized)
				return
			}
			req = req.WithContext(frameworksecurity.AddUserToContext(req.Context(), user))
			next(w, req, ps)
			return
		}
		h.WriteError(w, "unauthorized", http.StatusUnauthorized)
	}
}

func applyGatewayLocalAPIAuth(req *http.Request) {
	if req == nil {
		return
	}
	apiCfg := global.Env().SystemConfig.APIConfig
	if !apiCfg.Security.Enabled {
		return
	}
	username := strings.TrimSpace(apiCfg.Security.Username)
	if username == "" {
		return
	}
	req.SetBasicAuth(username, apiCfg.Security.Password)
}
