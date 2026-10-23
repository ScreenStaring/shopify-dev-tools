package inventory

import (
	"encoding/json"
	"testing"
)

func TestInventoryTransferSearchQuery(t *testing.T) {
	tests := []struct {
		name          string
		originID      string
		destinationID string
		status        string
		want          string
	}{
		{name: "no filters", want: ""},
		{name: "origin", originID: "123", want: "origin_id:123"},
		{name: "destination", destinationID: "456", want: "destination_id:456"},
		{name: "status is lowercased", status: "DRAFT", want: "status:draft"},
		{name: "status is trimmed", status: " ready_to_ship ", want: "status:ready_to_ship"},
		{name: "blank status", status: "  ", want: ""},
		{name: "origin and status", originID: "123", status: "IN_PROGRESS", want: "origin_id:123 AND status:in_progress"},
		{
			name:          "all filters",
			originID:      "123",
			destinationID: "456",
			status:        "TRANSFERRED",
			want:          "origin_id:123 AND destination_id:456 AND status:transferred",
		},
		{
			name:          "locations only",
			originID:      "123",
			destinationID: "456",
			want:          "origin_id:123 AND destination_id:456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := inventoryTransferSearchQuery(tt.originID, tt.destinationID, tt.status)
			if got != tt.want {
				t.Errorf(
					"inventoryTransferSearchQuery(%q, %q, %q) = %q, want %q",
					tt.originID, tt.destinationID, tt.status, got, tt.want,
				)
			}
		})
	}
}

func TestTransferFromNodeJSONOriginLocation(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		want     string
		wantID   string
		wantDisp string
	}{
		{
			name:     "live location",
			raw:      `{"id":"gid://shopify/InventoryTransfer/1","origin":{"name":"Warehouse","location":{"id":"gid://shopify/Location/2"}}}`,
			want:     "Warehouse",
			wantID:   "gid://shopify/Location/2",
			wantDisp: "Warehouse",
		},
		{
			name:     "deleted or inaccessible location",
			raw:      `{"id":"gid://shopify/InventoryTransfer/1","origin":{"name":"Warehouse","location":null}}`,
			want:     "Warehouse",
			wantID:   "",
			wantDisp: "Warehouse (No Location)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var node transferNodeJSON
			if err := json.Unmarshal([]byte(tt.raw), &node); err != nil {
				t.Fatal(err)
			}

			transfer := transferFromNodeJSON(node)

			if transfer.Origin != tt.want {
				t.Errorf("Origin = %q, want %q", transfer.Origin, tt.want)
			}
			if transfer.OriginLocationID != tt.wantID {
				t.Errorf("OriginLocationID = %q, want %q", transfer.OriginLocationID, tt.wantID)
			}
			if got := transferOrigin(transfer); got != tt.wantDisp {
				t.Errorf("transferOrigin() = %q, want %q", got, tt.wantDisp)
			}
		})
	}
}
