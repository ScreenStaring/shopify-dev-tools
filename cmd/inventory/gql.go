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

const inventoryTransfersQuery = `
query InventoryTransfers($first: Int!, $after: String, $query: String, $sortKey: TransferSortKeys, $reverse: Boolean) {
  inventoryTransfers(first: $first, after: $after, query: $query, sortKey: $sortKey, reverse: $reverse) {
    nodes {
      id
      name
      status
      dateCreated
      origin {
        name
        location {
          id
        }
      }
      destination {
        name
      }
      lineItems(first: 250) {
        nodes {
          id
          title
          totalQuantity
          shippedQuantity
        }
        pageInfo {
          hasNextPage
          endCursor
        }
      }
    }
    pageInfo {
      hasNextPage
      endCursor
    }
  }
}
`

const inventoryTransfersByIDsQuery = `
query InventoryTransfersByIds($ids: [ID!]!) {
  nodes(ids: $ids) {
    ... on InventoryTransfer {
      id
      name
      status
      dateCreated
      origin {
        name
        location {
          id
        }
      }
      destination {
        name
      }
      lineItems(first: 250) {
        nodes {
          id
          title
          totalQuantity
          shippedQuantity
        }
        pageInfo {
          hasNextPage
          endCursor
        }
      }
    }
  }
}
`

const inventoryTransferLineItemsQuery = `
query InventoryTransferLineItems($id: ID!, $after: String) {
  inventoryTransfer(id: $id) {
    lineItems(first: 250, after: $after) {
      nodes {
        id
        title
        totalQuantity
        shippedQuantity
      }
      pageInfo {
        hasNextPage
        endCursor
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

type pageInfoJSON struct {
	HasNextPage bool   `json:"hasNextPage"`
	EndCursor   string `json:"endCursor"`
}

// TransferLineItem is an item transferred between locations.
type TransferLineItem struct {
	ID              string `json:"id"`
	Title           string `json:"title"`
	TotalQuantity   int    `json:"totalQuantity"`
	ShippedQuantity int    `json:"shippedQuantity"`
}

// Transfer is the movement of inventory items from an origin location to a
// destination location.
type Transfer struct {
	ID               string             `json:"id"`
	Name             string             `json:"name"`
	Status           string             `json:"status"`
	DateCreated      string             `json:"dateCreated,omitempty"`
	Origin           string             `json:"origin,omitempty"`
	OriginLocationID string             `json:"originLocationId,omitempty"`
	Destination      string             `json:"destination,omitempty"`
	LineItems        []TransferLineItem `json:"lineItems,omitempty"`
}

type transferLineItemNodeJSON struct {
	ID              string `json:"id"`
	Title           string `json:"title"`
	TotalQuantity   int    `json:"totalQuantity"`
	ShippedQuantity int    `json:"shippedQuantity"`
}

type transferNodeJSON struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	DateCreated string `json:"dateCreated"`
	Origin      *struct {
		Name     string `json:"name"`
		Location *struct {
			ID string `json:"id"`
		} `json:"location"`
	} `json:"origin"`
	Destination *struct {
		Name string `json:"name"`
	} `json:"destination"`
	LineItems struct {
		Nodes    []transferLineItemNodeJSON `json:"nodes"`
		PageInfo pageInfoJSON               `json:"pageInfo"`
	} `json:"lineItems"`
}

type inventoryTransfersResponse struct {
	Data struct {
		InventoryTransfers struct {
			Nodes    []transferNodeJSON `json:"nodes"`
			PageInfo pageInfoJSON       `json:"pageInfo"`
		} `json:"inventoryTransfers"`
	} `json:"data"`
}

type inventoryTransfersByIDsResponse struct {
	Data struct {
		Nodes []*transferNodeJSON `json:"nodes"`
	} `json:"data"`
}

type inventoryTransferLineItemsResponse struct {
	Data struct {
		InventoryTransfer *struct {
			LineItems struct {
				Nodes    []transferLineItemNodeJSON `json:"nodes"`
				PageInfo pageInfoJSON               `json:"pageInfo"`
			} `json:"lineItems"`
		} `json:"inventoryTransfer"`
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

// inventoryTransferSearchQuery builds the inventoryTransfers search query for
// the given origin and destination location ids, and/or transfer status.
func inventoryTransferSearchQuery(originID, destinationID, status string) string {
	var parts []string

	if len(originID) > 0 {
		parts = append(parts, "origin_id:"+originID)
	}

	if len(destinationID) > 0 {
		parts = append(parts, "destination_id:"+destinationID)
	}

	if len(strings.TrimSpace(status)) > 0 {
		parts = append(parts, "status:"+strings.ToLower(strings.TrimSpace(status)))
	}

	return strings.Join(parts, " AND ")
}

// FetchInventoryTransfersByIDs returns the transfers with the given
// InventoryTransfer GIDs. The second return value holds the ids for which no
// transfer was found.
func FetchInventoryTransfersByIDs(client *gqlclient.Client, ids []string) ([]Transfer, []string, error) {
	data, err := client.Execute(inventoryTransfersByIDsQuery, map[string]interface{}{"ids": ids})
	if err != nil {
		return nil, nil, fmt.Errorf("Cannot fetch inventory transfers: %s", err)
	}

	b, err := json.Marshal(data)
	if err != nil {
		return nil, nil, fmt.Errorf("Cannot re-encode inventory transfers response: %s", err)
	}

	var response inventoryTransfersByIDsResponse
	if err := json.Unmarshal(b, &response); err != nil {
		return nil, nil, fmt.Errorf("Cannot parse inventory transfers response: %s", err)
	}

	return transfersFromNodes(client, response.Data.Nodes, ids)
}

// transfersFromNodes converts the given nodes into transfers. The second
// return value holds the ids whose node is nil, i.e. the transfers that
// don't exist.
func transfersFromNodes(client *gqlclient.Client, nodes []*transferNodeJSON, ids []string) ([]Transfer, []string, error) {
	transfers := make([]Transfer, 0, len(nodes))
	var missing []string

	for i, node := range nodes {
		if node == nil {
			missing = append(missing, ids[i])
			continue
		}

		transfer, err := transferFromNode(client, *node)
		if err != nil {
			return nil, nil, err
		}

		transfers = append(transfers, transfer)
	}

	return transfers, missing, nil
}

// listInventoryTransfers returns the inventory transfers of the given 1-based
// page, newest first, optionally restricted by a search query. Each page holds
// limit transfers; pages before it are skipped by following the connection's
// cursor.
func listInventoryTransfers(client *gqlclient.Client, search string, limit, page int) ([]Transfer, error) {
	if page < 1 {
		return nil, fmt.Errorf("--page must be 1 or greater")
	}

	if limit < 1 {
		return nil, fmt.Errorf("--limit must be 1 or greater")
	}

	variables := map[string]interface{}{
		"first":   limit,
		"sortKey": "CREATED_AT",
		"reverse": true,
	}
	if len(search) > 0 {
		variables["query"] = search
	}

	after := ""
	for current := 1; ; current++ {
		if len(after) > 0 {
			variables["after"] = after
		}

		data, err := client.Execute(inventoryTransfersQuery, variables)
		if err != nil {
			return nil, fmt.Errorf("Cannot list inventory transfers: %s", err)
		}

		b, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("Cannot re-encode inventory transfers response: %s", err)
		}

		var response inventoryTransfersResponse
		if err := json.Unmarshal(b, &response); err != nil {
			return nil, fmt.Errorf("Cannot parse inventory transfers response: %s", err)
		}

		result := response.Data.InventoryTransfers

		if current == page {
			transfers := make([]Transfer, 0, len(result.Nodes))
			for _, node := range result.Nodes {
				transfer, err := transferFromNode(client, node)
				if err != nil {
					return nil, err
				}

				transfers = append(transfers, transfer)
			}

			return transfers, nil
		}

		if !result.PageInfo.HasNextPage {
			return nil, nil
		}

		after = result.PageInfo.EndCursor
	}
}

// transferFromNode converts a transfer node, fetching the remaining line items
// when the transfer has more than one page of them.
func transferFromNode(client *gqlclient.Client, node transferNodeJSON) (Transfer, error) {
	transfer := transferFromNodeJSON(node)

	if node.LineItems.PageInfo.HasNextPage {
		lineItems, err := fetchTransferLineItems(client, node.ID, node.LineItems.PageInfo.EndCursor)
		if err != nil {
			return transfer, err
		}

		transfer.LineItems = append(transfer.LineItems, lineItems...)
	}

	return transfer, nil
}

func transferFromNodeJSON(node transferNodeJSON) Transfer {
	transfer := Transfer{
		ID:          node.ID,
		Name:        node.Name,
		Status:      node.Status,
		DateCreated: node.DateCreated,
	}

	if node.Origin != nil {
		transfer.Origin = node.Origin.Name

		if node.Origin.Location != nil {
			transfer.OriginLocationID = node.Origin.Location.ID
		}
	}
	if node.Destination != nil {
		transfer.Destination = node.Destination.Name
	}

	for _, line := range node.LineItems.Nodes {
		transfer.LineItems = append(transfer.LineItems, transferLineItemFromNode(line))
	}

	return transfer
}

func transferLineItemFromNode(node transferLineItemNodeJSON) TransferLineItem {
	return TransferLineItem{
		ID:              node.ID,
		Title:           node.Title,
		TotalQuantity:   node.TotalQuantity,
		ShippedQuantity: node.ShippedQuantity,
	}
}

// fetchTransferLineItems returns the transfer's line items after the given
// cursor.
func fetchTransferLineItems(client *gqlclient.Client, id, after string) ([]TransferLineItem, error) {
	var lineItems []TransferLineItem

	for len(after) > 0 {
		data, err := client.Execute(inventoryTransferLineItemsQuery, map[string]interface{}{
			"id":    id,
			"after": after,
		})
		if err != nil {
			return nil, fmt.Errorf("Cannot get inventory transfer %s line items: %s", id, err)
		}

		b, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("Cannot re-encode inventory transfer line items response: %s", err)
		}

		var response inventoryTransferLineItemsResponse
		if err := json.Unmarshal(b, &response); err != nil {
			return nil, fmt.Errorf("Cannot parse inventory transfer line items response: %s", err)
		}

		node := response.Data.InventoryTransfer
		if node == nil {
			return nil, fmt.Errorf("Cannot get inventory transfer %s: not found or access denied", id)
		}

		for _, line := range node.LineItems.Nodes {
			lineItems = append(lineItems, transferLineItemFromNode(line))
		}

		if !node.LineItems.PageInfo.HasNextPage {
			break
		}

		after = node.LineItems.PageInfo.EndCursor
	}

	return lineItems, nil
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
