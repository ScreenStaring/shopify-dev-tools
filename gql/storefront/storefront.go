package storefront

import (
	"fmt"
	"strings"
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

const enableMutation = `
  mutation Enable($namespace: String! $key: String! $owner: MetafieldOwnerType!) {
    metafieldStorefrontVisibilityCreate(
      input: {
	namespace: $namespace
	key: $key
	ownerType: $owner
      }
    ) {
      metafieldStorefrontVisibility {
	id
      }
      userErrors {
	field
	message
      }
    }
  }
`

func New(shop, token string) *Storefront {
	client := gql.NewClient(shop, token, "")
	return &Storefront{client}
}


func stringifyUserErrors(errors []interface{}) string {
	var userError []string

	for _, error := range errors {
		error := error.(map[string]interface{})
		userError = append(userError, fmt.Sprint(error["message"]))
	}

	return strings.Join(userError, ", ")
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

func (sf *Storefront) Enable(name, owner string) (string, error)  {
	var result string

	key := strings.SplitN(name, ".", 2)
	if len(key) < 2 {
		return result, fmt.Errorf("Metafield key %s invalid: must be in namespace.key format", name)
	}

	data, err := sf.client.Mutation(enableMutation, map[string]interface{}{"namespace": key[0], "key": key[1], "owner": strings.ToUpper(owner)})
	if err != nil {
		return result, fmt.Errorf("Failed to enable storefront metafield %s: %s", name, err)
	}

	// TODO: collect 'em all!
	message, _ := data.ValueForPathString("errors[0].message")
	if len(message) > 0 {
		return result, fmt.Errorf("Request failed: %s", message)
	}

	messages, _ := data.ValuesForPath("data.metafieldStorefrontVisibilityCreate.userErrors")
	if len(messages) > 0 {
		return result, fmt.Errorf("Request failed: %s", stringifyUserErrors(messages))
	}

	result, err = data.ValueForPathString("data.metafieldStorefrontVisibilityCreate.metafieldStorefrontVisibility.id")
	if err != nil {
		return result, fmt.Errorf("Failed to extract storefront metafield visibility id from response: %s", err)
	}

	return result, nil
}
