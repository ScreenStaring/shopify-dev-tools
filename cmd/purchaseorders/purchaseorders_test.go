package purchaseorders

import (
	"bytes"
	"encoding/csv"
	"strings"
	"testing"
)

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
