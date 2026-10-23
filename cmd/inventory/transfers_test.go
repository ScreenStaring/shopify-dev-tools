package inventory

import "testing"

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
