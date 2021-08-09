package storefront

import (
	"fmt"
	"github.com/ScreenStaring/shopify-dev-tools/gql"
)

type Storefront struct {
	client *gql.Client
}

const listQuery = `
{
  metafieldStorefrontVisibilities(first: 250) {
    pageInfo {
      hasNextPage
    }
    edges {
      cursor
      node {
        id
        key
        namespace
        createdAt
        updatedAt
        legacyResourceId
        namespace
        ownerType
      }
    }
  }
}
`


func New(shop, token string) *Storefront {
	client := gql.NewClient(shop, token)
	return &Storefront{client}
}

// no pagination...
func (sf *Storefront) List() ([]map[string]interface{}, error)  {
	var result []map[string]interface{}

	data, err := sf.client.Query(listQuery)
	if err != nil {
		return result, fmt.Errorf("Failed to retrieve storefront metafields: %s", err)
	}

	nodes, err := data.ValuesForPath("data.metafieldStorefrontVisibilities.edges.node")
	if err != nil {
		return result, fmt.Errorf("Failed to extract storefront metafields from response: %s", err)
	}

	for _, node := range nodes {
		result = append(result, node.(map[string]interface{}))
	}

	return result, nil
}
