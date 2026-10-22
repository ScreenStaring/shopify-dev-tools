package inventory

import "testing"

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
