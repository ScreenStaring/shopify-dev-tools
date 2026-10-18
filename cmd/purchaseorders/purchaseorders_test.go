package purchaseorders

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"strings"
	"testing"
)

const purchaseOrdersJSON = `{
  "data": {
    "inventoryPurchaseOrders": {
      "pageInfo": { "hasNextPage": true, "endCursor": "cursor-1" },
      "nodes": [
        {
          "id": "gid://shopify/InventoryPurchaseOrder/1",
          "name": "#PO1001",
          "currency": "USD",
          "status": "ORDERED",
          "origin": { "supplierName": "Acme Supply Co." },
          "destination": { "name": "Warehouse" },
          "lineItems": {
            "nodes": [
              { "totalCost": { "amount": "20.50", "currencyCode": "USD" } },
              { "totalCost": { "amount": "4.50", "currencyCode": "USD" } }
            ]
          },
          "transfers": {
            "nodes": [
              {
                "name": "TR-1",
                "receivedQuantity": 5,
                "totalQuantity": 10,
                "shipments": {
                  "nodes": [
                    { "tracking": { "arrivesAt": "2026-11-02T10:00:00Z" } }
                  ]
                }
              },
              {
                "name": "TR-2",
                "receivedQuantity": 3,
                "totalQuantity": 10,
                "shipments": {
                  "nodes": [
                    { "tracking": { "arrivesAt": "2026-10-25T10:00:00Z" } },
                    { "tracking": null }
                  ]
                }
              }
            ]
          }
        },
        {
          "id": "gid://shopify/InventoryPurchaseOrder/2",
          "name": "#PO1002",
          "currency": "USD",
          "status": "DRAFT",
          "origin": null,
          "destination": null,
          "lineItems": { "nodes": [] },
          "transfers": { "nodes": [] }
        }
      ]
    }
  }
}`

func TestParsePurchaseOrders(t *testing.T) {
	orders, pageInfo, err := parsePurchaseOrders([]byte(purchaseOrdersJSON))
	if err != nil {
		t.Fatalf("parse failed: %s", err)
	}

	if len(orders) != 2 {
		t.Fatalf("want 2 purchase orders, got %d", len(orders))
	}

	if !pageInfo.HasNextPage || pageInfo.EndCursor != "cursor-1" {
		t.Errorf("pageInfo = %+v, want hasNextPage true and cursor-1", pageInfo)
	}

	first := orders[0]
	if first.Name != "#PO1001" {
		t.Errorf("Name = %q, want %q", first.Name, "#PO1001")
	}
	if first.Supplier != "Acme Supply Co." {
		t.Errorf("Supplier = %q, want %q", first.Supplier, "Acme Supply Co.")
	}
	if first.Destination != "Warehouse" {
		t.Errorf("Destination = %q, want %q", first.Destination, "Warehouse")
	}
	if first.Status != "ORDERED" {
		t.Errorf("Status = %q, want %q", first.Status, "ORDERED")
	}
	if first.Total != "25 USD" {
		t.Errorf("Total = %q, want %q", first.Total, "25 USD")
	}
	if first.ExpectedArrival != "2026-10-25" {
		t.Errorf("ExpectedArrival = %q, want %q", first.ExpectedArrival, "2026-10-25")
	}
	if len(first.Transfers) != 2 {
		t.Fatalf("want 2 transfers, got %d", len(first.Transfers))
	}
	if got := transferNames(first.Transfers); got != "TR-1, TR-2" {
		t.Errorf("transferNames = %q, want %q", got, "TR-1, TR-2")
	}
	if got, total, ok := receivedQuantity(first.Transfers); !ok || got != 8 || total != 20 {
		t.Errorf("receivedQuantity = %d, %d, %t; want 8, 20, true", got, total, ok)
	}

	second := orders[1]
	if second.Supplier != "" || second.Destination != "" {
		t.Errorf("expected empty supplier and destination, got %q and %q", second.Supplier, second.Destination)
	}
	if second.Total != "" {
		t.Errorf("Total = %q, want empty", second.Total)
	}
	if second.ExpectedArrival != "" {
		t.Errorf("ExpectedArrival = %q, want empty", second.ExpectedArrival)
	}
	if _, _, ok := receivedQuantity(second.Transfers); ok {
		t.Error("receivedQuantity reported a value for a purchase order with no transfers")
	}
}

func TestPurchaseOrderQuery(t *testing.T) {
	tests := []struct {
		name   string
		ids    []int64
		status string
		want   string
	}{
		{name: "no filters", want: ""},
		{name: "one id", ids: []int64{123}, want: "id:123"},
		{name: "many ids", ids: []int64{123, 456}, want: "(id:123 OR id:456)"},
		{name: "status", status: "ordered", want: "status:ORDERED"},
		{name: "status is trimmed", status: " ordered ", want: "status:ORDERED"},
		{name: "ids and status", ids: []int64{123, 456}, status: "draft", want: "(id:123 OR id:456) AND status:DRAFT"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := purchaseOrderQuery(tt.ids, tt.status); got != tt.want {
				t.Errorf("purchaseOrderQuery(%v, %q) = %q, want %q", tt.ids, tt.status, got, tt.want)
			}
		})
	}
}

func TestPurchaseOrderIDs(t *testing.T) {
	ids, err := purchaseOrderIDs([]string{"123", "456"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 2 || ids[0] != 123 || ids[1] != 456 {
		t.Errorf("ids = %v, want [123 456]", ids)
	}

	if _, err := purchaseOrderIDs([]string{"abc"}); err == nil {
		t.Error("want error for invalid id")
	}
}

func TestPurchaseOrderID(t *testing.T) {
	tests := []struct {
		gid  string
		want string
	}{
		{gid: "gid://shopify/InventoryPurchaseOrder/1234", want: "1234"},
		{gid: "1234", want: "1234"},
		{gid: "", want: ""},
	}

	for _, tt := range tests {
		if got := purchaseOrderID(tt.gid); got != tt.want {
			t.Errorf("purchaseOrderID(%q) = %q, want %q", tt.gid, got, tt.want)
		}
	}
}

func TestListPurchaseOrdersRejectsInvalidPage(t *testing.T) {
	if _, err := listPurchaseOrders(nil, nil, "", 10, 0); err == nil {
		t.Error("want error for page 0")
	}
}

func TestExpectedArrivalFallback(t *testing.T) {
	var transfers []purchaseOrderTransferJSON
	if err := json.Unmarshal([]byte(`[
		{ "shipments": { "nodes": [
			{ "tracking": { "arrivesAt": "not-a-date" } },
			{ "tracking": { "arrivesAt": "2026-10-25T10:00:00Z" } }
		] } }
	]`), &transfers); err != nil {
		t.Fatalf("parse failed: %s", err)
	}

	if got := expectedArrival(transfers); got != "2026-10-25" {
		t.Errorf("expectedArrival = %q, want %q", got, "2026-10-25")
	}

	transfers = nil
	if err := json.Unmarshal([]byte(`[
		{ "shipments": { "nodes": [ { "tracking": { "arrivesAt": "not-a-date" } } ] } }
	]`), &transfers); err != nil {
		t.Fatalf("parse failed: %s", err)
	}

	if got := expectedArrival(transfers); got != "not-a-date" {
		t.Errorf("expectedArrival = %q, want %q", got, "not-a-date")
	}
}

func TestWritePurchaseOrderRow(t *testing.T) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	order := ExportPurchaseOrder{
		ID:          "gid://shopify/InventoryPurchaseOrder/1234",
		Name:        "#PO1001",
		Supplier:    "Acme Supply Co.",
		Destination: "Warehouse",
		Status:      "ORDERED",
		Transfers:   []Transfer{{Name: "TR-1"}},
		LineItems: []ExportLineItem{
			{Title: "Widget", SKU: "ACME-01", Quantity: 10, UnitCost: "2.00", UnitCurrency: "USD", TotalCost: "20.00", TotalCurrency: "USD"},
			{Title: "Gadget", SKU: "ACME-02", Quantity: 5, UnitCost: "1.00", UnitCurrency: "USD", TotalCost: "5.00", TotalCurrency: "USD"},
		},
	}

	if err := writePurchaseOrderRow(w, order); err != nil {
		t.Fatalf("write failed: %s", err)
	}
	w.Flush()

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("want 2 rows, got %d: %q", len(lines), buf.String())
	}

	want := "1234,#PO1001,Acme Supply Co.,Warehouse,ORDERED,TR-1,Widget,ACME-01,10,2.00,USD,20.00,USD"
	if lines[0] != want {
		t.Errorf("row = %q, want %q", lines[0], want)
	}
}

func TestWritePurchaseOrderRowWithoutLineItems(t *testing.T) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	order := ExportPurchaseOrder{
		ID:     "gid://shopify/InventoryPurchaseOrder/2",
		Name:   "#PO1002",
		Status: "DRAFT",
	}

	if err := writePurchaseOrderRow(w, order); err != nil {
		t.Fatalf("write failed: %s", err)
	}
	w.Flush()

	want := "2,#PO1002,,,DRAFT,,,,,,,,\n"
	if buf.String() != want {
		t.Errorf("row = %q, want %q", buf.String(), want)
	}
}
