/* ©INFINI, All Rights Reserved.
 * mail: contact#infini.ltd */

package managed

import (
	"fmt"
	"strings"

	log "github.com/cihub/seelog"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/model"
	"infini.sh/framework/core/util"
	ucfg "infini.sh/framework/lib/go-ucfg"
	"infini.sh/framework/modules/configs/client"
	configcommon "infini.sh/framework/modules/configs/common"
)

const (
	tokenExchangeAPI      = "/instance/_exchange_token"
	managerAccessTokenKey = "CONFIGS_MANAGER_ACCESS_TOKEN"
)

var getInstanceInfoFunc = model.GetInstanceInfo
var ensureGatewayAccessTokenFunc = func() (string, error) {
	return configcommon.EnsureTokenInKeystore(configcommon.AgentAccessTokenKeystoreKey)
}
var doManagerRequestFunc = client.DoManagerRequest
var saveManagerAccessTokenFunc = func(tokenValue string) error {
	return configcommon.SaveTokenToKeystore(managerAccessTokenKey, tokenValue)
}

type tokenExchangeRequest struct {
	InstanceID    string `json:"instance_id,omitempty"`
	AgentAPIToken string `json:"agent_api_token,omitempty"`
}

type tokenExchangeResponse struct {
	ManagerAPIToken string `json:"manager_api_token,omitempty"`
}

func ExchangeTokens(server string, res *util.Result) error {
	if !global.Env().SystemConfig.Configs.Managed || len(global.Env().SystemConfig.Configs.Servers) == 0 {
		return nil
	}
	if res == nil || (res.StatusCode != 200 && !strings.Contains(string(res.Body), "exists")) {
		return nil
	}

	bootstrapToken := strings.TrimSpace(global.Env().SystemConfig.Configs.ManagerConfig.AccessToken.Get())
	if bootstrapToken == "" {
		return nil
	}

	gatewayAPIToken, err := ensureGatewayAccessTokenFunc()
	if err != nil {
		return err
	}
	instance := getInstanceInfoFunc()

	reqBody := tokenExchangeRequest{
		InstanceID:    instance.ID,
		AgentAPIToken: gatewayAPIToken,
	}
	req := util.Request{
		Method:      util.Verb_POST,
		Path:        tokenExchangeAPI,
		ContentType: "application/json",
		Body:        util.MustToJSONBytes(reqBody),
	}

	exchangeServer, exchangeRes, err := doManagerRequestFunc(&req)
	if err != nil {
		return err
	}
	if exchangeRes == nil {
		if exchangeServer == "" {
			exchangeServer = server
		}
		return fmt.Errorf("empty response from %s", exchangeServer)
	}
	if exchangeRes.StatusCode != 200 {
		return fmt.Errorf("token exchange failed on %s, status: %d, body: %s", exchangeServer, exchangeRes.StatusCode, string(exchangeRes.Body))
	}

	resp := tokenExchangeResponse{}
	if err := util.FromJSONBytes(exchangeRes.Body, &resp); err != nil {
		return err
	}
	if strings.TrimSpace(resp.ManagerAPIToken) == "" {
		return fmt.Errorf("manager api token is empty")
	}

	if err := saveManagerAccessTokenFunc(resp.ManagerAPIToken); err != nil {
		return err
	}
	global.Env().SystemConfig.Configs.ManagerConfig.AccessToken = ucfg.SecretString(resp.ManagerAPIToken)
	log.Infof("exchanged managed access token from %s", exchangeServer)
	return nil
}

func RegisterTokenExchangeCallback() {
	client.AddPostRegisterHook(ExchangeTokens)
}
