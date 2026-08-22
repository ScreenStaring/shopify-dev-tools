package cmd

import (
	"reflect"
	"testing"
)

func TestParseIDArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		idDesc   string
		wantIDs  []int64
		wantSKUs []string
		wantErr  string
	}{
		{
			name:     "mixed ids and skus",
			args:     []string{"123", "sku:FOO", "456", "sku:BAR"},
			idDesc:   "a product id",
			wantIDs:  []int64{123, 456},
			wantSKUs: []string{"FOO", "BAR"},
		},
		{
			name:     "sku only",
			args:     []string{"sku:FOO"},
			idDesc:   "an order id",
			wantSKUs: []string{"FOO"},
		},
		{
			name:    "id only",
			args:    []string{"789"},
			idDesc:  "an ID",
			wantIDs: []int64{789},
		},
		{
			name:   "no args",
			idDesc: "a product id",
		},
		{
			name:     "uppercase sku prefix",
			args:     []string{"SKU:foo"},
			idDesc:   "a product id",
			wantSKUs: []string{"foo"},
		},
		{
			name:     "sku containing colon",
			args:     []string{"sku:FOO:BAR"},
			idDesc:   "a product id",
			wantSKUs: []string{"FOO:BAR"},
		},
		{
			name:    "empty sku",
			args:    []string{"sku:"},
			idDesc:  "a product id",
			wantErr: "SKU value missing after 'sku:'",
		},
		{
			name:    "invalid arg",
			args:    []string{"abc"},
			idDesc:  "a product id",
			wantErr: "Argument 'abc' invalid: must be a product id or 'sku:VALUE'",
		},
		{
			name:    "idDesc appears in error",
			args:    []string{"abc"},
			idDesc:  "an order id",
			wantErr: "Argument 'abc' invalid: must be an order id or 'sku:VALUE'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ids, skus, err := ParseIDArgs(tt.args, tt.idDesc)

			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("want error %q, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !reflect.DeepEqual(ids, tt.wantIDs) {
				t.Errorf("ids = %v, want %v", ids, tt.wantIDs)
			}
			if !reflect.DeepEqual(skus, tt.wantSKUs) {
				t.Errorf("skus = %v, want %v", skus, tt.wantSKUs)
			}
		})
	}
}
