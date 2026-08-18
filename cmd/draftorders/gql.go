package draftorders

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ScreenStaring/shopify-dev-tools/gql"
)

const draftOrdersQuery = `
query($query: String!, $first: Int!, $sortKey: DraftOrderSortKeys!) {
  draftOrders(first: $first, query: $query, sortKey: $sortKey, reverse: true) {
    edges {
      node {
        legacyResourceId
        name
        status
        createdAt
        updatedAt
        completedAt
        invoiceSentAt
        reserveInventoryUntil
        note2
        order { legacyResourceId }
        lineItems(first: 250) {
          edges {
            node {
              id
              product { legacyResourceId }
              variant { legacyResourceId }
              sku
              title
              quantity
            }
          }
        }
      }
    }
  }
}
`

type LineItem struct {
	ID        string `json:"id,omitempty"`
	ProductID int64  `json:"product_id,omitempty"`
	VariantID int64  `json:"variant_id,omitempty"`
	SKU       string `json:"sku,omitempty"`
	Name      string `json:"title,omitempty"`
	Quantity  int    `json:"quantity,omitempty"`
}

type DraftOrder struct {
	ID                    int64      `json:"id,omitempty"`
	Name                  string     `json:"name,omitempty"`
	Status                string     `json:"status,omitempty"`
	CreatedAt             string     `json:"created_at,omitempty"`
	UpdatedAt             string     `json:"updated_at,omitempty"`
	CompletedAt           string     `json:"completed_at,omitempty"`
	InvoiceSentAt         string     `json:"invoice_sent_at,omitempty"`
	ReserveInventoryUntil string     `json:"reserve_inventory_until,omitempty"`
	Note                  string     `json:"note,omitempty"`
	OrderID               int64      `json:"order_id,omitempty"`
	LineItems             []LineItem `json:"line_items,omitempty"`
}

type resourceRef struct {
	LegacyResourceId int64 `json:"legacyResourceId,string"`
}

type lineItemJSON struct {
	ID       string       `json:"id"`
	Product  *resourceRef `json:"product"`
	Variant  *resourceRef `json:"variant"`
	SKU      string       `json:"sku"`
	Title    string       `json:"title"`
	Quantity int          `json:"quantity"`
}

type draftOrderJSON struct {
	LegacyResourceId      int64        `json:"legacyResourceId,string"`
	Name                  string       `json:"name"`
	Status                string       `json:"status"`
	CreatedAt             string       `json:"createdAt"`
	UpdatedAt             string       `json:"updatedAt"`
	CompletedAt           string       `json:"completedAt"`
	InvoiceSentAt         string       `json:"invoiceSentAt"`
	ReserveInventoryUntil string       `json:"reserveInventoryUntil"`
	Note                  string       `json:"note2"`
	Order                 *resourceRef `json:"order"`
	LineItems             struct {
		Edges []struct {
			Node lineItemJSON `json:"node"`
		} `json:"edges"`
	} `json:"lineItems"`
}

type draftOrdersResponse struct {
	Data struct {
		DraftOrders struct {
			Edges []struct {
				Node draftOrderJSON `json:"node"`
			} `json:"edges"`
		} `json:"draftOrders"`
	} `json:"data"`
}

var draftOrderSortKeys = map[string]string{
	"updated":       "UPDATED_AT",
	"updated_at":    "UPDATED_AT",
	"customer_name": "CUSTOMER_NAME",
	"id":            "ID",
	"number":        "NUMBER",
	"status":        "STATUS",
	"total_price":   "TOTAL_PRICE",
	"relevance":     "RELEVANCE",
}

func ResolveDraftOrderSortKey(value string) (string, error) {
	if len(value) == 0 {
		return "UPDATED_AT", nil
	}

	if key, ok := draftOrderSortKeys[strings.ToLower(value)]; ok {
		return key, nil
	}

	for _, key := range draftOrderSortKeys {
		if value == key {
			return key, nil
		}
	}

	return "", fmt.Errorf("Invalid --sort value '%s'", value)
}

func buildQuery(ids []int64, skus []string, status string) string {
	var parts []string

	for _, id := range ids {
		parts = append(parts, fmt.Sprintf("id:%d", id))
	}
	for _, sku := range skus {
		parts = append(parts, fmt.Sprintf("sku:%s", sku))
	}

	if len(parts) > 0 {
		return strings.Join(parts, " OR ")
	}
	return "status:" + status
}

func listDraftOrders(shop, token string, ids []int64, skus []string, status string, limit int, sortKey string) ([]DraftOrder, error) {
	client := gql.NewClient(shop, token)

	query := buildQuery(ids, skus, status)

	data, err := client.Execute(draftOrdersQuery, map[string]interface{}{"query": query, "first": limit, "sortKey": sortKey})
	if err != nil {
		return nil, fmt.Errorf("Cannot list draft orders: %s", err)
	}

	b, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("Cannot re-encode draft orders response: %s", err)
	}

	var response draftOrdersResponse
	if err := json.Unmarshal(b, &response); err != nil {
		return nil, fmt.Errorf("Cannot parse draft orders response: %s", err)
	}

	var result []DraftOrder
	for _, edge := range response.Data.DraftOrders.Edges {
		n := edge.Node
		order := DraftOrder{
			ID:                    n.LegacyResourceId,
			Name:                  n.Name,
			Status:                n.Status,
			CreatedAt:             n.CreatedAt,
			UpdatedAt:             n.UpdatedAt,
			CompletedAt:           n.CompletedAt,
			InvoiceSentAt:         n.InvoiceSentAt,
			ReserveInventoryUntil: n.ReserveInventoryUntil,
			Note:                  n.Note,
		}

		if n.Order != nil {
			order.OrderID = n.Order.LegacyResourceId
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
				ID:        strings.TrimPrefix(li.ID, "gid://shopify/DraftOrderLineItem/"),
				ProductID: productID,
				VariantID: variantID,
				SKU:       li.SKU,
				Name:      li.Title,
				Quantity:  li.Quantity,
			})
		}

		result = append(result, order)
	}

	return result, nil
}
