package orders

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ScreenStaring/shopify-dev-tools/gql"
)

const ordersQuery = `
query($query: String!, $first: Int!) {
  orders(first: $first, query: $query) {
    edges {
      node {
        legacyResourceId
        name
        createdAt
        updatedAt
        cancelledAt
        closedAt
        displayFinancialStatus
        displayFulfillmentStatus
        note
        lineItems(first: 250) {
          edges {
            node {
              id
              product { legacyResourceId }
              variant { legacyResourceId }
              sku
              name
              quantity
              fulfillmentStatus
            }
          }
        }
      }
    }
  }
}
`

type LineItem struct {
	ID                string
	ProductID         int64
	VariantID         int64
	SKU               string
	Name              string
	Quantity          int
	FulfillmentStatus string
}

type Order struct {
	ID                       int64
	Name                     string
	CreatedAt                string
	UpdatedAt                string
	CancelledAt              string
	ClosedAt                 string
	DisplayFinancialStatus   string
	DisplayFulfillmentStatus string
	Note                     string
	LineItems                []LineItem
}

type resourceRef struct {
	LegacyResourceId int64 `json:"legacyResourceId,string"`
}

type lineItemJSON struct {
	ID                string       `json:"id"`
	Product           *resourceRef `json:"product"`
	Variant           *resourceRef `json:"variant"`
	SKU               string       `json:"sku"`
	Name              string       `json:"name"`
	Quantity          int          `json:"quantity"`
	FulfillmentStatus string       `json:"fulfillmentStatus"`
}

type orderJSON struct {
	LegacyResourceId         int64  `json:"legacyResourceId,string"`
	Name                     string `json:"name"`
	CreatedAt                string `json:"createdAt"`
	UpdatedAt                string `json:"updatedAt"`
	CancelledAt              string `json:"cancelledAt"`
	ClosedAt                 string `json:"closedAt"`
	DisplayFinancialStatus   string `json:"displayFinancialStatus"`
	DisplayFulfillmentStatus string `json:"displayFulfillmentStatus"`
	Note                     string `json:"note"`
	LineItems                struct {
		Edges []struct {
			Node lineItemJSON `json:"node"`
		} `json:"edges"`
	} `json:"lineItems"`
}

type ordersResponse struct {
	Data struct {
		Orders struct {
			Edges []struct {
				Node orderJSON `json:"node"`
			} `json:"edges"`
		} `json:"orders"`
	} `json:"data"`
}

func buildQuery(ids []int64, status string) (string, int) {
	if len(ids) > 0 {
		parts := make([]string, len(ids))
		for i, id := range ids {
			parts[i] = fmt.Sprintf("id:%d", id)
		}
		return strings.Join(parts, " OR "), len(ids)
	}
	return "status:" + status, 0
}

func listOrders(shop, token string, ids []int64, status string, limit int) ([]Order, error) {
	client := gql.NewClient(shop, token)

	query, first := buildQuery(ids, status)
	if first == 0 {
		first = limit
	}

	data, err := client.Execute(ordersQuery, map[string]interface{}{"query": query, "first": first})
	if err != nil {
		return nil, fmt.Errorf("Cannot list orders: %s", err)
	}

	b, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("Cannot re-encode orders response: %s", err)
	}

	var response ordersResponse
	if err := json.Unmarshal(b, &response); err != nil {
		return nil, fmt.Errorf("Cannot parse orders response: %s", err)
	}

	var result []Order
	for _, edge := range response.Data.Orders.Edges {
		n := edge.Node
		order := Order{
			ID:                       n.LegacyResourceId,
			Name:                     n.Name,
			CreatedAt:                n.CreatedAt,
			UpdatedAt:                n.UpdatedAt,
			CancelledAt:              n.CancelledAt,
			ClosedAt:                 n.ClosedAt,
			DisplayFinancialStatus:   n.DisplayFinancialStatus,
			DisplayFulfillmentStatus: n.DisplayFulfillmentStatus,
			Note:                     n.Note,
		}

		for _, liEdge := range n.LineItems.Edges {
			li := liEdge.Node
			var productID, variantID int64
			if li.Product != nil {
				productID = li.Product.LegacyResourceId
			}
			if li.Variant != nil {
				variantID = li.Variant.LegacyResourceId
			}
			order.LineItems = append(order.LineItems, LineItem{
				ID:                li.ID,
				ProductID:         productID,
				VariantID:         variantID,
				SKU:               li.SKU,
				Name:              li.Name,
				Quantity:          li.Quantity,
				FulfillmentStatus: li.FulfillmentStatus,
			})
		}

		result = append(result, order)
	}

	return result, nil
}
