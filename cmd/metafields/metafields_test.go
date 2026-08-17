package metafields

import (
	"reflect"
	"testing"
)

func TestMissingSkus(t *testing.T) {
	tests := []struct {
		name      string
		requested []string
		found     []string
		want      []string
	}{
		{
			name:      "all found",
			requested: []string{"A", "B"},
			found:     []string{"B", "A"},
			want:      nil,
		},
		{
			name:      "some missing",
			requested: []string{"A", "B", "C"},
			found:     []string{"A"},
			want:      []string{"B", "C"},
		},
		{
			name:      "all missing",
			requested: []string{"A", "B"},
			found:     []string{"C"},
			want:      []string{"A", "B"},
		},
		{
			name:      "duplicate requested",
			requested: []string{"A", "A"},
			found:     []string{"A"},
			want:      nil,
		},
		{
			name:      "empty requested",
			requested: nil,
			found:     []string{"A"},
			want:      nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := missingSkus(tt.requested, tt.found)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("missingSkus(%v, %v) = %v, want %v", tt.requested, tt.found, got, tt.want)
			}
		})
	}
}
