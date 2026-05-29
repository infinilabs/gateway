package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/model"
	configcommon "infini.sh/framework/modules/configs/common"
)

func TestInitSelfAPIEnsuresAccessToken(t *testing.T) {
	t.Setenv("KEYSTORE_PATH", t.TempDir())

	InitSelfAPI()

	value, err := configcommon.LoadTokenFromKeystore(configcommon.AgentAccessTokenKeystoreKey)
	if err != nil {
		t.Fatalf("load access token: %v", err)
	}
	if value == "" {
		t.Fatal("expected gateway access token to be initialized")
	}
}

func TestGatewaySelfAPIRequiresManagedAccessToken(t *testing.T) {
	t.Setenv("KEYSTORE_PATH", t.TempDir())

	token, err := configcommon.EnsureTokenInKeystore(configcommon.AgentAccessTokenKeystoreKey)
	if err != nil {
		t.Fatalf("ensure access token: %v", err)
	}

	originalWebConfig := global.Env().SystemConfig.WebAppConfig
	t.Cleanup(func() {
		global.Env().SystemConfig.WebAppConfig = originalWebConfig
	})
	global.Env().SystemConfig.WebAppConfig.Security.Enabled = false

	handler := selfAPIHandler{}
	protected := handler.requireLoginOrAccessToken(func(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
		w.WriteHeader(http.StatusAccepted)
	})

	req := httptest.NewRequest(http.MethodGet, "/queue/stats", nil)
	req.Header.Set(model.API_TOKEN, token)
	recorder := httptest.NewRecorder()
	protected(recorder, req, nil)
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("expected token-authenticated request to succeed, got %d", recorder.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/queue/stats", nil)
	recorder = httptest.NewRecorder()
	protected(recorder, req, nil)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected missing token to be rejected, got %d", recorder.Code)
	}
}

func TestApplyGatewayLocalAPIAuth(t *testing.T) {
	originalCfg := global.Env().SystemConfig.APIConfig
	t.Cleanup(func() {
		global.Env().SystemConfig.APIConfig = originalCfg
	})

	global.Env().SystemConfig.APIConfig.Security.Enabled = true
	global.Env().SystemConfig.APIConfig.Security.Username = "api-user"
	global.Env().SystemConfig.APIConfig.Security.Password = "api-pass"

	req := httptest.NewRequest(http.MethodGet, "/stats", nil)
	applyGatewayLocalAPIAuth(req)

	username, password, ok := req.BasicAuth()
	if !ok || username != "api-user" || password != "api-pass" {
		t.Fatalf("unexpected basic auth: %v %s %s", ok, username, password)
	}
}
