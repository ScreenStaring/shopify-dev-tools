package inventory

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	productsgql "github.com/ScreenStaring/shopify-dev-tools/cmd/products/gql"
	gqlclient "github.com/ScreenStaring/shopify-dev-tools/gql"
)

const inventoryItemsQuery = `
query VariantsByInventoryItemIds($ids: [ID!]!) {
  nodes(ids: $ids) {
    ... on InventoryItem {
      id
      variant {
        id
        sku
        title
        product {
          id
          title
        }
      }
    }
  }
}
`

type inventoryItemJSON struct {
	ID      string `json:"id"`
	Variant *struct {
		ID      string `json:"id"`
		SKU     string `json:"sku"`
		Title   string `json:"title"`
		Product *struct {
			ID    string `json:"id"`
			Title string `json:"title"`
		} `json:"product"`
	} `json:"variant"`
}

type inventoryItemsResponse struct {
	Data struct {
		Nodes []*inventoryItemJSON `json:"nodes"`
	} `json:"data"`
}

// FetchProductsByInventoryItemIDs returns the products (with their
// variants) associated with the given inventory item GIDs. Variants that
// share a product are merged into a single product. The second return
// value holds the ids for which no inventory item was found.
func FetchProductsByInventoryItemIDs(client *gqlclient.Client, ids []string) ([]productsgql.Product, []string, error) {
	data, err := client.Execute(inventoryItemsQuery, map[string]interface{}{"ids": ids})
	if err != nil {
		return nil, nil, fmt.Errorf("Cannot fetch inventory items: %s", err)
	}

	b, err := json.Marshal(data)
	if err != nil {
		return nil, nil, fmt.Errorf("Cannot re-encode inventory items response: %s", err)
	}

	var response inventoryItemsResponse
	if err := json.Unmarshal(b, &response); err != nil {
		return nil, nil, fmt.Errorf("Cannot parse inventory items response: %s", err)
	}

	return productsFromInventoryItems(response.Data.Nodes, ids)
}

// productsFromInventoryItems converts the inventory item nodes into the
// products (with their variants) associated with them. Variants that
// share a product are merged into a single product. The second return
// value holds the ids for which no inventory item was found.
func productsFromInventoryItems(nodes []*inventoryItemJSON, ids []string) ([]productsgql.Product, []string, error) {
	var order []*productsgql.Product
	byID := map[string]*productsgql.Product{}
	var missing []string

	for i, node := range nodes {
		if node == nil {
			missing = append(missing, ids[i])
			continue
		}

		if node.Variant == nil {
			continue
		}

		v := node.Variant

		if v.Product == nil {
			continue
		}

		product := byID[v.Product.ID]
		if product == nil {
			product = &productsgql.Product{
				ID:    idFromGID(v.Product.ID),
				Title: v.Product.Title,
			}
			byID[v.Product.ID] = product
			order = append(order, product)
		}

		product.Variants = append(product.Variants, productsgql.Variant{
			ID:    idFromGID(v.ID),
			Title: v.Title,
			SKU:   v.SKU,
		})
	}

	products := make([]productsgql.Product, len(order))
	for i, p := range order {
		products[i] = *p
	}

	return products, missing, nil
}

// idFromGID extracts the numeric id from a Shopify GID like
// gid://shopify/InventoryItem/123456.
func idFromGID(gid string) int64 {
	parts := strings.Split(gid, "/")
	if len(parts) == 0 {
		return 0
	}

	id, err := strconv.ParseInt(parts[len(parts)-1], 10, 64)
	if err != nil {
		return 0
	}

	return id
}
