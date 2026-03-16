package gql

import (
	"encoding/json"
	"fmt"

	gqlclient "github.com/ScreenStaring/shopify-dev-tools/gql"
)

const productDeleteMutation = `
mutation($id: ID!) {
  productDelete(input: {id: $id}) {
    deletedProductId
    userErrors {
      field
      message
    }
  }
}
`

type ProductDeleteResult struct {
	DeletedProductID string
	UserErrors       []string
}

type productDeleteResponse struct {
	Data struct {
		ProductDelete struct {
			DeletedProductID string `json:"deletedProductId"`
			UserErrors       []struct {
				Field   []string `json:"field"`
				Message string   `json:"message"`
			} `json:"userErrors"`
		} `json:"productDelete"`
	} `json:"data"`
}

func ProductDelete(shop, token, id string, options map[string]interface{}) (*ProductDeleteResult, error) {
	client := gqlclient.NewClient(shop, token, options)

	data, err := client.Execute(productDeleteMutation, map[string]interface{}{"id": id})
	if err != nil {
		return nil, fmt.Errorf("productDelete mutation failed: %s", err)
	}

	b, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("Cannot re-encode productDelete response: %s", err)
	}

	var response productDeleteResponse
	if err := json.Unmarshal(b, &response); err != nil {
		return nil, fmt.Errorf("Cannot parse productDelete response: %s", err)
	}

	result := &ProductDeleteResult{
		DeletedProductID: response.Data.ProductDelete.DeletedProductID,
	}

	for _, ue := range response.Data.ProductDelete.UserErrors {
		result.UserErrors = append(result.UserErrors, ue.Message)
	}

	return result, nil
}
