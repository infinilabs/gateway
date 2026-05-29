package managed

import (
	"strings"
	"testing"

	"infini.sh/framework/core/global"
	"infini.sh/framework/core/model"
	"infini.sh/framework/core/util"
	ucfg "infini.sh/framework/lib/go-ucfg"
)

func TestExchangeTokensSkipsWhenBootstrapTokenMissing(t *testing.T) {
	oldManaged := global.Env().SystemConfig.Configs.Managed
	oldServers := global.Env().SystemConfig.Configs.Servers
	oldAccessToken := global.Env().SystemConfig.Configs.ManagerConfig.AccessToken
	oldEnsure := ensureGatewayAccessTokenFunc
	oldDoManagerRequest := doManagerRequestFunc
	t.Cleanup(func() {
		global.Env().SystemConfig.Configs.Managed = oldManaged
		global.Env().SystemConfig.Configs.Servers = oldServers
		global.Env().SystemConfig.Configs.ManagerConfig.AccessToken = oldAccessToken
		ensureGatewayAccessTokenFunc = oldEnsure
		doManagerRequestFunc = oldDoManagerRequest
	})

	global.Env().SystemConfig.Configs.Managed = true
	global.Env().SystemConfig.Configs.Servers = []string{"https://console.local"}
	global.Env().SystemConfig.Configs.ManagerConfig.AccessToken = ""

	called := false
	ensureGatewayAccessTokenFunc = func() (string, error) {
		called = true
		return "gateway-api-token", nil
	}
	doManagerRequestFunc = func(req *util.Request) (string, *util.Result, error) {
		called = true
		return "", nil, nil
	}

	if err := ExchangeTokens("https://console.local", &util.Result{StatusCode: 200}); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if called {
		t.Fatal("expected exchange to short-circuit before requesting tokens")
	}
}

func TestExchangeTokensStoresManagerAccessToken(t *testing.T) {
	oldManaged := global.Env().SystemConfig.Configs.Managed
	oldServers := global.Env().SystemConfig.Configs.Servers
	oldAccessToken := global.Env().SystemConfig.Configs.ManagerConfig.AccessToken
	oldEnsure := ensureGatewayAccessTokenFunc
	oldDoManagerRequest := doManagerRequestFunc
	oldSave := saveManagerAccessTokenFunc
	oldGetInstanceInfo := getInstanceInfoFunc
	t.Cleanup(func() {
		global.Env().SystemConfig.Configs.Managed = oldManaged
		global.Env().SystemConfig.Configs.Servers = oldServers
		global.Env().SystemConfig.Configs.ManagerConfig.AccessToken = oldAccessToken
		ensureGatewayAccessTokenFunc = oldEnsure
		doManagerRequestFunc = oldDoManagerRequest
		saveManagerAccessTokenFunc = oldSave
		getInstanceInfoFunc = oldGetInstanceInfo
	})

	global.Env().SystemConfig.Configs.Managed = true
	global.Env().SystemConfig.Configs.Servers = []string{"https://console.local"}
	global.Env().SystemConfig.Configs.ManagerConfig.AccessToken = ucfg.SecretString("bootstrap-token")
	getInstanceInfoFunc = func() model.Instance {
		instance := model.Instance{}
		instance.ID = "gateway-1"
		return instance
	}
	ensureGatewayAccessTokenFunc = func() (string, error) {
		return "gateway-api-token", nil
	}

	savedToken := ""
	doManagerRequestFunc = func(req *util.Request) (string, *util.Result, error) {
		if req == nil {
			t.Fatal("expected request")
		}
		if req.Path != tokenExchangeAPI {
			t.Fatalf("unexpected request path: %s", req.Path)
		}
		body := string(req.Body)
		if !strings.Contains(body, `"instance_id":"gateway-1"`) || !strings.Contains(body, `"agent_api_token":"gateway-api-token"`) {
			t.Fatalf("unexpected request body: %s", body)
		}
		return "https://console.local", &util.Result{
			StatusCode: 200,
			Body:       []byte(`{"manager_api_token":"manager-api-token"}`),
		}, nil
	}
	saveManagerAccessTokenFunc = func(tokenValue string) error {
		savedToken = tokenValue
		return nil
	}

	if err := ExchangeTokens("https://console.local", &util.Result{StatusCode: 200}); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if savedToken != "manager-api-token" {
		t.Fatalf("unexpected saved token: %q", savedToken)
	}
	if got := global.Env().SystemConfig.Configs.ManagerConfig.AccessToken.Get(); got != "manager-api-token" {
		t.Fatalf("unexpected manager access token in config: %q", got)
	}
}
