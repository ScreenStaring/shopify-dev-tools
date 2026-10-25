package inventory

import (
	"strings"
	"testing"
)

func TestTransferOrigin(t *testing.T) {
	tests := []struct {
		name     string
		transfer Transfer
		want     string
	}{
		{
			name:     "live location",
			transfer: Transfer{Origin: "Warehouse", OriginLocationID: "gid://shopify/Location/1"},
			want:     "Warehouse",
		},
		{
			name:     "no live location",
			transfer: Transfer{Origin: "Warehouse"},
			want:     "Warehouse (No Location)",
		},
		{name: "no origin", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := transferOrigin(tt.transfer); got != tt.want {
				t.Errorf("transferOrigin(%+v) = %q, want %q", tt.transfer, got, tt.want)
			}
		})
	}
}

func TestLineItemTitle(t *testing.T) {
	tests := []struct {
		name string
		line TransferLineItem
		want string
	}{
		{
			name: "default variant shows the product title only",
			line: TransferLineItem{Title: "T-Shirt", VariantTitle: "Default Title"},
			want: "T-Shirt",
		},
		{
			name: "no variant title",
			line: TransferLineItem{Title: "T-Shirt"},
			want: "T-Shirt",
		},
		{
			name: "non-default variant is appended",
			line: TransferLineItem{Title: "T-Shirt", VariantTitle: "Large"},
			want: "T-Shirt - Large",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := lineItemTitle(tt.line); got != tt.want {
				t.Errorf("lineItemTitle(%+v) = %q, want %q", tt.line, got, tt.want)
			}
		})
	}
}

func TestPrintTransferLineItems(t *testing.T) {
	lineItems := []TransferLineItem{
		{
			ID:              "gid://shopify/InventoryTransferLineItem/1",
			Title:           "T-Shirt",
			VariantTitle:    "Large",
			TotalQuantity:   5,
			ShippedQuantity: 2,
		},
		{
			ID:              "gid://shopify/InventoryTransferLineItem/2",
			Title:           "Mug",
			VariantTitle:    "Default Title",
			TotalQuantity:   3,
			ShippedQuantity: 3,
		},
	}

	out, _ := captureOutput(t, func() { printTransferLineItems(lineItems) })

	if !strings.Contains(out, "ID  Product          Total Quantity  Shipped Quantity") {
		t.Errorf("output missing the expected header:\n%s", out)
	}

	if strings.Contains(out, "Title") {
		t.Errorf("header must not use 'Title':\n%s", out)
	}

	if !strings.Contains(out, "T-Shirt - Large") {
		t.Errorf("output missing the variant title:\n%s", out)
	}

	if strings.Contains(out, "Mug - Default Title") {
		t.Errorf("output must not append the default variant title:\n%s", out)
	}

	if !strings.Contains(out, "Mug") {
		t.Errorf("output missing the default variant's product title:\n%s", out)
	}
}

func TestExitCodeFor(t *testing.T) {
	tests := []struct {
		name           string
		found, missing int
		want           int
	}{
		{name: "none requested", want: 0},
		{name: "all found", found: 3, want: 0},
		{name: "none found", missing: 2, want: 1},
		{name: "some found", found: 2, missing: 1, want: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := exitCodeFor(tt.found, tt.missing); got != tt.want {
				t.Errorf("exitCodeFor(%d, %d) = %d, want %d", tt.found, tt.missing, got, tt.want)
			}
		})
	}
}
