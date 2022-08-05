package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"channeld.clewcat.com/channeld/pkg/channeld"
	"channeld.clewcat.com/channeld/pkg/channeldpb"
	"go.uber.org/zap"
)

type APIServerAuthProvider struct {
	BaseURL string
}

var logger *zap.Logger

// PIT = Username, LT = AccessToken
func (provider *APIServerAuthProvider) DoAuth(pit string, lt string) (channeldpb.AuthResultMessage_AuthResult, error) {
	url := fmt.Sprintf("%s/users/%s", provider.BaseURL, pit)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		logger.Error("failed to create HTTP request", zap.Error(err))
		return 0, err
	}
	req.Header.Set("Authorization", lt)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.Error("failed to do HTTP request", zap.Error(err), zap.String("url", url))
		return 0, err
	}

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Error("failed to read HTTP response body", zap.Error(err), zap.String("url", url))
		return 0, err
	}

	jsonObj := new(map[string]json.RawMessage)
	jsonErr := json.Unmarshal(respBytes, jsonObj)
	if jsonErr != nil {
		logger.Error("failed to parse response JSON", zap.Error(err), zap.String("url", url), zap.ByteString("body", respBytes))
		return 0, jsonErr
	}

	if (*jsonObj)["code"] != nil {
		var code string
		err := json.Unmarshal((*jsonObj)["code"], &code)
		if err != nil {
			logger.Error("failed to parse 'code' from response", zap.Error(err), zap.String("url", url), zap.ByteString("body", respBytes))
			return 0, err
		}

		if code == "000000" {
			return channeldpb.AuthResultMessage_SUCCESSFUL, nil
		}
	}

	return channeldpb.AuthResultMessage_INVALID_LT, nil
}

func SetupAuth() {
	if channeld.GlobalSettings.Development {
		logger, _ = zap.NewDevelopment()
	} else {
		logger, _ = zap.NewProduction()
	}
	defer logger.Sync()

	authProvider := &APIServerAuthProvider{}

	if arg := flag.Lookup("--apiBaseUrl"); arg != nil {
		authProvider.BaseURL = arg.Value.String()
	} else if env, exists := os.LookupEnv("APISERVER_SERVICE_HOST"); exists {
		authProvider.BaseURL = fmt.Sprintf("https://%s:8443/api/v1", env)
	} else {
		authProvider.BaseURL = "https://api-test.kooola.com:8443/api/v1"
	}

	channeld.SetAuthProvider(authProvider)
}
