package orders

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/ScreenStaring/shopify-dev-tools/gql"
)

const fulfillmentsQuery = `
query($id: ID!) {
  order(id: $id) {
    fulfillments {
      id
      name
      displayStatus
      createdAt
      updatedAt
      service {
        serviceName
        type
      }
      location {
        name
      }
      trackingInfo {
        company
        number
        url
      }
      fulfillmentLineItems(first: 250) {
        edges {
          node {
            lineItem {
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

const ordersQuery = `
query($query: String!, $first: Int!) {
  orders(first: $first, query: $query, sortKey: CREATED_AT, reverse: true) {
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

type TrackingInfo struct {
	Company string
	Number  string
	URL     string
}

type Fulfillment struct {
	ID            string
	Name          string
	DisplayStatus string
	CreatedAt     string
	UpdatedAt     string
	ServiceName   string
	ServiceType   string
	LocationName  string
	TrackingInfo  []TrackingInfo
	LineItems     []LineItem
}

type trackingInfoJSON struct {
	Company string `json:"company"`
	Number  string `json:"number"`
	URL     string `json:"url"`
}

type fulfillmentJSON struct {
	ID            string             `json:"id"`
	Name          string             `json:"name"`
	DisplayStatus string             `json:"displayStatus"`
	CreatedAt     string             `json:"createdAt"`
	UpdatedAt     string             `json:"updatedAt"`
	Service       *struct {
		ServiceName string `json:"serviceName"`
		Type        string `json:"type"`
	} `json:"service"`
	Location *struct {
		Name string `json:"name"`
	} `json:"location"`
	TrackingInfo []trackingInfoJSON `json:"trackingInfo"`
	FulfillmentLineItems struct {
		Edges []struct {
			Node struct {
				LineItem lineItemJSON `json:"lineItem"`
			} `json:"node"`
		} `json:"edges"`
	} `json:"fulfillmentLineItems"`
}

type fulfillmentsResponse struct {
	Data struct {
		Order struct {
			Fulfillments []fulfillmentJSON `json:"fulfillments"`
		} `json:"order"`
	} `json:"data"`
}

func listFulfillments(shop, token, orderID string) ([]Fulfillment, error) {
	client := gql.NewClient(shop, token)

	if !strings.HasPrefix(orderID, "gid://") {
		orderID = "gid://shopify/Order/" + orderID
	}

	data, err := client.Execute(fulfillmentsQuery, map[string]interface{}{"id": orderID})
	if err != nil {
		return nil, fmt.Errorf("Cannot list fulfillments: %s", err)
	}

	b, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("Cannot re-encode fulfillments response: %s", err)
	}

	var response fulfillmentsResponse
	if err := json.Unmarshal(b, &response); err != nil {
		return nil, fmt.Errorf("Cannot parse fulfillments response: %s", err)
	}

	var result []Fulfillment
	for _, f := range response.Data.Order.Fulfillments {
		ff := Fulfillment{
			ID:            f.ID,
			Name:          f.Name,
			DisplayStatus: f.DisplayStatus,
			CreatedAt:     f.CreatedAt,
			UpdatedAt:     f.UpdatedAt,
		}

		if f.Service != nil {
			ff.ServiceName = f.Service.ServiceName
			ff.ServiceType = f.Service.Type
		}

		if f.Location != nil {
			ff.LocationName = f.Location.Name
		}

		for _, ti := range f.TrackingInfo {
			ff.TrackingInfo = append(ff.TrackingInfo, TrackingInfo{
				Company: ti.Company,
				Number:  ti.Number,
				URL:     ti.URL,
			})
		}

		for _, edge := range f.FulfillmentLineItems.Edges {
			li := edge.Node.LineItem
			var productID, variantID int64
			if li.Product != nil {
				productID = li.Product.LegacyResourceId
			}
			if li.Variant != nil {
				variantID = li.Variant.LegacyResourceId
			}
			ff.LineItems = append(ff.LineItems, LineItem{
				ID:                li.ID,
				ProductID:         productID,
				VariantID:         variantID,
				SKU:               li.SKU,
				Name:              li.Name,
				Quantity:          li.Quantity,
				FulfillmentStatus: li.FulfillmentStatus,
			})
		}

		result = append(result, ff)
	}

	// Sort by UpdatedAt descending
	sort.Slice(result, func(i, j int) bool {
		return result[i].UpdatedAt > result[j].UpdatedAt
	})

	return result, nil
}

const fulfillmentEventCreateMutation = `
mutation($fulfillmentEvent: FulfillmentEventInput!) {
  fulfillmentEventCreate(fulfillmentEvent: $fulfillmentEvent) {
    fulfillmentEvent {
      id
      status
      happenedAt
      message
    }
    userErrors {
      field
      message
    }
  }
}
`

func createFulfillmentDeliveredEvent(shop, token, fulfillmentID, happenedAt, message string) (string, error) {
	client := gql.NewClient(shop, token)

	if !strings.HasPrefix(fulfillmentID, "gid://") {
		fulfillmentID = "gid://shopify/Fulfillment/" + fulfillmentID
	}

	event := map[string]interface{}{
		"fulfillmentId": fulfillmentID,
		"status":        "DELIVERED",
		"happenedAt":    happenedAt,
	}

	if len(message) > 0 {
		event["message"] = message
	}

	data, err := client.Execute(fulfillmentEventCreateMutation, map[string]interface{}{"fulfillmentEvent": event})
	if err != nil {
		return "", fmt.Errorf("Cannot create fulfillment event: %s", err)
	}

	b, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("Cannot re-encode fulfillment event response: %s", err)
	}

	var response struct {
		Data struct {
			FulfillmentEventCreate struct {
				FulfillmentEvent *struct {
					ID string `json:"id"`
				} `json:"fulfillmentEvent"`
				UserErrors []struct {
					Field   []string `json:"field"`
					Message string   `json:"message"`
				} `json:"userErrors"`
			} `json:"fulfillmentEventCreate"`
		} `json:"data"`
	}

	if err := json.Unmarshal(b, &response); err != nil {
		return "", fmt.Errorf("Cannot parse fulfillment event response: %s", err)
	}

	if errs := response.Data.FulfillmentEventCreate.UserErrors; len(errs) > 0 {
		var messages []string
		for _, e := range errs {
			messages = append(messages, e.Message)
		}
		return "", fmt.Errorf("Cannot create fulfillment event: %s", strings.Join(messages, ", "))
	}

	return response.Data.FulfillmentEventCreate.FulfillmentEvent.ID, nil
}
