package gql

import (
	"encoding/json"
	"fmt"

	gqlclient "github.com/ScreenStaring/shopify-dev-tools/gql"
)

type ProductSetResult struct {
	ProductID  string
	UserErrors []string
}

type productSetResponse struct {
	Data struct {
		ProductSet struct {
			Product struct {
				ID string `json:"id"`
			} `json:"product"`
			UserErrors []struct {
				Field   []string `json:"field"`
				Message string   `json:"message"`
			} `json:"userErrors"`
		} `json:"productSet"`
	} `json:"data"`
}

func ProductSet(shop, token string, variables map[string]interface{}, options map[string]interface{}) (*ProductSetResult, error) {
	client := gqlclient.NewClient(shop, token, options)

	data, err := client.Execute(productSetMutation, variables)
	if err != nil {
		return nil, fmt.Errorf("productSet mutation failed: %s", err)
	}

	b, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("Cannot re-encode productSet response: %s", err)
	}

	var response productSetResponse
	if err := json.Unmarshal(b, &response); err != nil {
		return nil, fmt.Errorf("Cannot parse productSet response: %s", err)
	}

	result := &ProductSetResult{
		ProductID: response.Data.ProductSet.Product.ID,
	}

	for _, ue := range response.Data.ProductSet.UserErrors {
		result.UserErrors = append(result.UserErrors, ue.Message)
	}

	return result, nil
}
