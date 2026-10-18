package purchaseorders

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/ScreenStaring/shopify-dev-tools/gql"
)

const purchaseOrderGIDPrefix = "gid://shopify/InventoryPurchaseOrder/"

const purchaseOrdersListQuery = `
query($first: Int!, $after: String, $query: String) {
  inventoryPurchaseOrders(first: $first, after: $after, query: $query) {
    nodes {
      id
      name
      currency
      origin {
        supplierName
      }
      destination {
        name
      }
      status
      lineItems(first: 250) {
        nodes {
          totalCost {
            amount
            currencyCode
          }
        }
      }
      transfers(first: 10) {
        nodes {
          name
          receivedQuantity
          totalQuantity
          shipments(first: 1) {
            nodes {
              tracking {
                arrivesAt
              }
            }
          }
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

const purchaseOrderTransfersQuery = `
query($id: ID!, $after: String) {
  inventoryPurchaseOrder(id: $id) {
    id
    name
    status
    origin {
      supplierName
    }
    destination {
      name
    }
    transfers(first: 10, after: $after) {
      nodes {
        id
        name
      }
      pageInfo {
        hasNextPage
        endCursor
      }
    }
  }
}
`

const purchaseOrderLineItemsQuery = `
query($id: ID!, $after: String) {
  inventoryPurchaseOrder(id: $id) {
    lineItems(first: 50, after: $after) {
      nodes {
        title
        supplierSku
        totalQuantity
        unitCost {
          amount
          currencyCode
        }
        totalCost {
          amount
          currencyCode
        }
      }
      pageInfo {
        hasNextPage
        endCursor
      }
    }
  }
}
`

type moneyJSON struct {
	Amount       string `json:"amount"`
	CurrencyCode string `json:"currencyCode"`
}

type pageInfoJSON struct {
	HasNextPage bool   `json:"hasNextPage"`
	EndCursor   string `json:"endCursor"`
}

type purchaseOrderTransferJSON struct {
	Name             string `json:"name"`
	ReceivedQuantity int    `json:"receivedQuantity"`
	TotalQuantity    int    `json:"totalQuantity"`
	Shipments        struct {
		Nodes []struct {
			Tracking *struct {
				ArrivesAt string `json:"arrivesAt"`
			} `json:"tracking"`
		} `json:"nodes"`
	} `json:"shipments"`
}

type purchaseOrderNodeJSON struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Currency string `json:"currency"`
	Status   string `json:"status"`
	Origin   *struct {
		SupplierName string `json:"supplierName"`
	} `json:"origin"`
	Destination *struct {
		Name string `json:"name"`
	} `json:"destination"`
	LineItems struct {
		Nodes []struct {
			TotalCost *moneyJSON `json:"totalCost"`
		} `json:"nodes"`
	} `json:"lineItems"`
	Transfers struct {
		Nodes []purchaseOrderTransferJSON `json:"nodes"`
	} `json:"transfers"`
}

type purchaseOrdersResponse struct {
	Data struct {
		InventoryPurchaseOrders struct {
			Nodes    []purchaseOrderNodeJSON `json:"nodes"`
			PageInfo pageInfoJSON            `json:"pageInfo"`
		} `json:"inventoryPurchaseOrders"`
	} `json:"data"`
}

// Transfer is a purchase order's linked inventory transfer.
type Transfer struct {
	Name             string `json:"name"`
	ReceivedQuantity int    `json:"receivedQuantity"`
	TotalQuantity    int    `json:"totalQuantity"`
}

// PurchaseOrder is the package's native purchase order shape used by
// `purchaseorders ls`.
type PurchaseOrder struct {
	ID              string     `json:"id"`
	Name            string     `json:"name"`
	Supplier        string     `json:"supplier,omitempty"`
	Destination     string     `json:"destination,omitempty"`
	Status          string     `json:"status"`
	Currency        string     `json:"currency,omitempty"`
	Transfers       []Transfer `json:"transfers,omitempty"`
	Total           string     `json:"total,omitempty"`
	ExpectedArrival string     `json:"expectedArrival,omitempty"`
}

// ExportLineItem is a purchase order line item as written to the export CSV.
type ExportLineItem struct {
	Title         string
	SKU           string
	Quantity      int
	UnitCost      string
	UnitCurrency  string
	TotalCost     string
	TotalCurrency string
}

// ExportPurchaseOrder is a purchase order with all of its line items and
// linked transfers, as written to the export CSV.
type ExportPurchaseOrder struct {
	ID          string
	Name        string
	Supplier    string
	Destination string
	Status      string
	Transfers   []Transfer
	LineItems   []ExportLineItem
}

type purchaseOrderTransfersResponse struct {
	Data struct {
		InventoryPurchaseOrder *struct {
			ID     string `json:"id"`
			Name   string `json:"name"`
			Status string `json:"status"`
			Origin *struct {
				SupplierName string `json:"supplierName"`
			} `json:"origin"`
			Destination *struct {
				Name string `json:"name"`
			} `json:"destination"`
			Transfers struct {
				Nodes []struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"nodes"`
				PageInfo pageInfoJSON `json:"pageInfo"`
			} `json:"transfers"`
		} `json:"inventoryPurchaseOrder"`
	} `json:"data"`
}

type purchaseOrderLineItemsResponse struct {
	Data struct {
		InventoryPurchaseOrder *struct {
			LineItems struct {
				Nodes []struct {
					Title         string     `json:"title"`
					SupplierSKU   string     `json:"supplierSku"`
					TotalQuantity int        `json:"totalQuantity"`
					UnitCost      *moneyJSON `json:"unitCost"`
					TotalCost     *moneyJSON `json:"totalCost"`
				} `json:"nodes"`
				PageInfo pageInfoJSON `json:"pageInfo"`
			} `json:"lineItems"`
		} `json:"inventoryPurchaseOrder"`
	} `json:"data"`
}

// purchaseOrderQuery builds the inventoryPurchaseOrders search query for the
// given numeric purchase order IDs and/or status.
func purchaseOrderQuery(ids []int64, status string) string {
	var parts []string

	if len(ids) > 0 {
		idParts := make([]string, 0, len(ids))
		for _, id := range ids {
			idParts = append(idParts, fmt.Sprintf("id:%d", id))
		}

		// Parentheses keep the ORs together when a status filter is ANDed on.
		query := strings.Join(idParts, " OR ")
		if len(ids) > 1 {
			query = "(" + query + ")"
		}

		parts = append(parts, query)
	}

	if len(status) > 0 {
		parts = append(parts, "status:"+strings.ToUpper(strings.TrimSpace(status)))
	}

	return strings.Join(parts, " AND ")
}

func parsePurchaseOrders(b []byte) ([]PurchaseOrder, pageInfoJSON, error) {
	var response purchaseOrdersResponse
	if err := json.Unmarshal(b, &response); err != nil {
		return nil, pageInfoJSON{}, fmt.Errorf("Cannot parse purchase orders response: %s", err)
	}

	var result []PurchaseOrder
	for _, node := range response.Data.InventoryPurchaseOrders.Nodes {
		result = append(result, purchaseOrderFromNode(node))
	}

	return result, response.Data.InventoryPurchaseOrders.PageInfo, nil
}

func purchaseOrderFromNode(node purchaseOrderNodeJSON) PurchaseOrder {
	po := PurchaseOrder{
		ID:       node.ID,
		Name:     node.Name,
		Status:   node.Status,
		Currency: node.Currency,
	}

	if node.Origin != nil {
		po.Supplier = node.Origin.SupplierName
	}
	if node.Destination != nil {
		po.Destination = node.Destination.Name
	}

	for _, t := range node.Transfers.Nodes {
		po.Transfers = append(po.Transfers, Transfer{
			Name:             t.Name,
			ReceivedQuantity: t.ReceivedQuantity,
			TotalQuantity:    t.TotalQuantity,
		})
	}

	po.Total = totalCost(node)
	po.ExpectedArrival = expectedArrival(node.Transfers.Nodes)

	return po
}

// totalCost sums the purchase order's line item costs and appends the
// purchase order's currency, e.g. "40.00 USD". It's empty when the purchase
// order has no line item costs.
func totalCost(node purchaseOrderNodeJSON) string {
	sum := decimal.Zero
	currency := node.Currency
	found := false

	for _, line := range node.LineItems.Nodes {
		if line.TotalCost == nil {
			continue
		}

		amount, err := decimal.NewFromString(line.TotalCost.Amount)
		if err != nil {
			continue
		}

		sum = sum.Add(amount)
		found = true

		if len(currency) == 0 {
			currency = line.TotalCost.CurrencyCode
		}
	}

	if !found {
		return ""
	}

	if len(currency) > 0 {
		return fmt.Sprintf("%s %s", sum.String(), currency)
	}

	return sum.String()
}

// expectedArrival returns the date of the earliest estimated shipment arrival
// across the purchase order's transfers. It's empty when no shipment has an
// estimated arrival.
func expectedArrival(transfers []purchaseOrderTransferJSON) string {
	var earliest time.Time
	var fallback string
	haveTime := false

	for _, transfer := range transfers {
		for _, shipment := range transfer.Shipments.Nodes {
			if shipment.Tracking == nil || len(shipment.Tracking.ArrivesAt) == 0 {
				continue
			}

			arrival := shipment.Tracking.ArrivesAt

			t, err := time.Parse(time.RFC3339, arrival)
			if err != nil {
				if len(fallback) == 0 {
					fallback = arrival
				}
				continue
			}

			if !haveTime || t.Before(earliest) {
				earliest = t
				haveTime = true
			}
		}
	}

	if haveTime {
		return earliest.UTC().Format("2006-01-02")
	}

	return fallback
}

// purchaseOrderID returns the bare numeric id of a purchase order GID.
func purchaseOrderID(gid string) string {
	return strings.TrimPrefix(gid, purchaseOrderGIDPrefix)
}

func transferNames(transfers []Transfer) string {
	names := make([]string, 0, len(transfers))
	for _, t := range transfers {
		names = append(names, t.Name)
	}

	return strings.Join(names, ", ")
}

// receivedQuantity returns the total quantity received and the total quantity
// ordered across the purchase order's transfers. The boolean is false when
// there are no transfers, so callers can distinguish "nothing received" from
// "no transfer linked".
func receivedQuantity(transfers []Transfer) (int, int, bool) {
	if len(transfers) == 0 {
		return 0, 0, false
	}

	received, total := 0, 0
	for _, t := range transfers {
		received += t.ReceivedQuantity
		total += t.TotalQuantity
	}

	return received, total, true
}

// listPurchaseOrders returns the purchase orders of the given 1-based page.
// Each page holds limit purchase orders; pages before it are skipped by
// following the connection's cursor.
func listPurchaseOrders(client *gql.Client, ids []int64, status string, limit, page int) ([]PurchaseOrder, error) {
	if page < 1 {
		return nil, fmt.Errorf("--page must be 1 or greater")
	}

	variables := map[string]interface{}{"first": limit}
	if query := purchaseOrderQuery(ids, status); len(query) > 0 {
		variables["query"] = query
	}

	after := ""
	for current := 1; ; current++ {
		if len(after) > 0 {
			variables["after"] = after
		}

		data, err := client.Execute(purchaseOrdersListQuery, variables)
		if err != nil {
			return nil, fmt.Errorf("Cannot list purchase orders: %s", err)
		}

		b, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("Cannot re-encode purchase orders response: %s", err)
		}

		orders, pageInfo, err := parsePurchaseOrders(b)
		if err != nil {
			return nil, err
		}

		if current == page {
			return orders, nil
		}

		if !pageInfo.HasNextPage {
			return nil, nil
		}

		after = pageInfo.EndCursor
	}
}

// fetchPurchaseOrderExport fetches a purchase order's header, all of its
// linked transfers, and all of its line items. Transfers and line items are
// paginated separately since each nested connection has its own cursor.
func fetchPurchaseOrderExport(client *gql.Client, id string) (ExportPurchaseOrder, error) {
	po := ExportPurchaseOrder{ID: id}

	after := ""
	for {
		variables := map[string]interface{}{"id": id}
		if len(after) > 0 {
			variables["after"] = after
		}

		data, err := client.Execute(purchaseOrderTransfersQuery, variables)
		if err != nil {
			return po, fmt.Errorf("Cannot get purchase order %s: %s", id, err)
		}

		b, err := json.Marshal(data)
		if err != nil {
			return po, fmt.Errorf("Cannot re-encode purchase order response: %s", err)
		}

		var response purchaseOrderTransfersResponse
		if err := json.Unmarshal(b, &response); err != nil {
			return po, fmt.Errorf("Cannot parse purchase order response: %s", err)
		}

		node := response.Data.InventoryPurchaseOrder
		if node == nil {
			return po, fmt.Errorf("Cannot get purchase order %s: not found or access denied", id)
		}

		if len(po.Name) == 0 {
			po.Name = node.Name
			po.Status = node.Status
			if node.Origin != nil {
				po.Supplier = node.Origin.SupplierName
			}
			if node.Destination != nil {
				po.Destination = node.Destination.Name
			}
		}

		for _, t := range node.Transfers.Nodes {
			po.Transfers = append(po.Transfers, Transfer{Name: t.Name})
		}

		if !node.Transfers.PageInfo.HasNextPage {
			break
		}
		after = node.Transfers.PageInfo.EndCursor
	}

	after = ""
	for {
		variables := map[string]interface{}{"id": id}
		if len(after) > 0 {
			variables["after"] = after
		}

		data, err := client.Execute(purchaseOrderLineItemsQuery, variables)
		if err != nil {
			return po, fmt.Errorf("Cannot get purchase order %s line items: %s", id, err)
		}

		b, err := json.Marshal(data)
		if err != nil {
			return po, fmt.Errorf("Cannot re-encode purchase order line items response: %s", err)
		}

		var response purchaseOrderLineItemsResponse
		if err := json.Unmarshal(b, &response); err != nil {
			return po, fmt.Errorf("Cannot parse purchase order line items response: %s", err)
		}

		node := response.Data.InventoryPurchaseOrder
		if node == nil {
			return po, fmt.Errorf("Cannot get purchase order %s: not found or access denied", id)
		}

		for _, line := range node.LineItems.Nodes {
			item := ExportLineItem{
				Title:    line.Title,
				SKU:      line.SupplierSKU,
				Quantity: line.TotalQuantity,
			}

			if line.UnitCost != nil {
				item.UnitCost = line.UnitCost.Amount
				item.UnitCurrency = line.UnitCost.CurrencyCode
			}
			if line.TotalCost != nil {
				item.TotalCost = line.TotalCost.Amount
				item.TotalCurrency = line.TotalCost.CurrencyCode
			}

			po.LineItems = append(po.LineItems, item)
		}

		if !node.LineItems.PageInfo.HasNextPage {
			break
		}
		after = node.LineItems.PageInfo.EndCursor
	}

	return po, nil
}

// fetchPurchaseOrderIDs returns the GIDs of the purchase orders of the given
// 1-based page. Each page holds limit purchase orders; pages before it are
// skipped by following the connection's cursor.
func fetchPurchaseOrderIDs(client *gql.Client, limit, page int) ([]string, error) {
	if page < 1 {
		return nil, fmt.Errorf("--page must be 1 or greater")
	}

	after := ""
	for current := 1; ; current++ {
		variables := map[string]interface{}{"first": limit}
		if len(after) > 0 {
			variables["after"] = after
		}

		data, err := client.Execute(purchaseOrderIDsQuery, variables)
		if err != nil {
			return nil, fmt.Errorf("Cannot list purchase orders: %s", err)
		}

		b, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("Cannot re-encode purchase orders response: %s", err)
		}

		var response purchaseOrderIDsResponse
		if err := json.Unmarshal(b, &response); err != nil {
			return nil, fmt.Errorf("Cannot parse purchase orders response: %s", err)
		}

		result := response.Data.InventoryPurchaseOrders
		ids := make([]string, 0, len(result.Nodes))
		for _, node := range result.Nodes {
			ids = append(ids, node.ID)
		}

		if current == page {
			return ids, nil
		}

		if !result.PageInfo.HasNextPage {
			return nil, nil
		}
		after = result.PageInfo.EndCursor
	}
}

const purchaseOrderIDsQuery = `
query($first: Int!, $after: String) {
  inventoryPurchaseOrders(first: $first, after: $after) {
    nodes {
      id
    }
    pageInfo {
      hasNextPage
      endCursor
    }
  }
}
`

type purchaseOrderIDsResponse struct {
	Data struct {
		InventoryPurchaseOrders struct {
			Nodes []struct {
				ID string `json:"id"`
			} `json:"nodes"`
			PageInfo pageInfoJSON `json:"pageInfo"`
		} `json:"inventoryPurchaseOrders"`
	} `json:"data"`
}
