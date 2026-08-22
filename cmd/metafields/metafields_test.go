package metafields

import (
	"reflect"
	"testing"
)

func TestParseOrderArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    orderFilter
		wantErr bool
	}{
		{
			name: "empty",
			args: nil,
			want: orderFilter{},
		},
		{
			name: "ids only",
			args: []string{"123", "456"},
			want: orderFilter{IDs: []int64{123, 456}},
		},
		{
			name: "name only",
			args: []string{"name:FOO"},
			want: orderFilter{Names: []string{"FOO"}},
		},
		{
			name: "sku only",
			args: []string{"sku:BAR"},
			want: orderFilter{SKUs: []string{"BAR"}},
		},
		{
			name: "mixed",
			args: []string{"123", "name:FOO", "sku:BAR", "456"},
			want: orderFilter{IDs: []int64{123, 456}, Names: []string{"FOO"}, SKUs: []string{"BAR"}},
		},
		{
			name: "uppercase name prefix",
			args: []string{"NAME:FOO"},
			want: orderFilter{Names: []string{"FOO"}},
		},
		{
			name:    "empty name",
			args:    []string{"name:"},
			wantErr: true,
		},
		{
			name:    "empty sku",
			args:    []string{"sku:"},
			wantErr: true,
		},
		{
			name:    "invalid arg",
			args:    []string{"abc"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseOrderArgs(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Errorf("parseOrderArgs(%v) = %v, want error", tt.args, got)
				}
				return
			}
			if err != nil {
				t.Errorf("parseOrderArgs(%v) unexpected error: %v", tt.args, err)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseOrderArgs(%v) = %v, want %v", tt.args, got, tt.want)
			}
		})
	}
}

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
