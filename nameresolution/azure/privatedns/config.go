package privatedns

import (
	"encoding/json"
	"fmt"

	"github.com/dapr/kit/config"
)

type configSpec struct {
	ClientId       string
	ClientSecret   string
	TenantId       string
	ZoneName       string
	AppId          string
	SubscriptionId string
	ResourceGroup  string
}

func parseConfig(rawConfig interface{}) (configSpec, error) {
	var result configSpec
	rawConfig, err := config.Normalize(rawConfig)
	if err != nil {
		return result, err
	}

	data, err := json.Marshal(rawConfig)
	if err != nil {
		return result, fmt.Errorf("Error serializing to json: %w", err)
	}

	err = json.Unmarshal(data, &result)
	if err != nil {
		return result, fmt.Errorf("Error deserializing to configSpec: %w", err)
	}

	return result, nil
}
