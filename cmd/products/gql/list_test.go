package gql

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestToProductWithInventoryLevels(t *testing.T) {
	body := `{
		"legacyResourceId": "123",
		"title": "Test Product",
		"hasOnlyDefaultVariant": true,
		"variants": {
			"edges": [{
				"node": {
					"legacyResourceId": "456",
					"title": "Red",
					"sku": "RED-1",
					"inventoryQuantity": 5,
					"inventoryItem": {
						"inventoryLevels": {
							"edges": [{
								"node": {
									"location": {"name": "Main Warehouse"},
									"quantities": [
										{"name": "available", "quantity": 5},
										{"name": "committed", "quantity": 2},
										{"name": "on_hand", "quantity": 10}
									]
								}
							}, {
								"node": {
									"location": {"name": "Retail"},
									"quantities": [
										{"name": "available", "quantity": 1},
										{"name": "on_hand", "quantity": 1}
									]
								}
							}]
						}
					}
				}
			}]
		}
	}`

	var n productJSON
	if err := json.Unmarshal([]byte(body), &n); err != nil {
		t.Fatalf("Cannot parse test JSON: %s", err)
	}

	product := toProduct(n)

	if product.ID != 123 || product.Title != "Test Product" {
		t.Errorf("product = %d/%q, want 123/Test Product", product.ID, product.Title)
	}

	if len(product.Variants) != 1 {
		t.Fatalf("len(variants) = %d, want 1", len(product.Variants))
	}

	v := product.Variants[0]
	if v.ID != 456 || v.Title != "Red" || v.SKU != "RED-1" || v.InventoryQuantity != 5 {
		t.Errorf("variant = %+v, want id 456/title Red/sku RED-1/quantity 5", v)
	}

	want := []InventoryLevel{
		{Location: "Main Warehouse", Available: 5, Committed: 2, OnHand: 10},
		{Location: "Retail", Available: 1, Committed: 0, OnHand: 1},
	}
	if !reflect.DeepEqual(v.InventoryLevels, want) {
		t.Errorf("inventory levels = %+v, want %+v", v.InventoryLevels, want)
	}
}

func TestToProductNoInventory(t *testing.T) {
	body := `{
		"legacyResourceId": "1",
		"title": "Plain",
		"variants": {
			"edges": [{
				"node": {
					"legacyResourceId": "2",
					"title": "Default",
					"inventoryQuantity": 3
				}
			}]
		}
	}`

	var n productJSON
	if err := json.Unmarshal([]byte(body), &n); err != nil {
		t.Fatalf("Cannot parse test JSON: %s", err)
	}

	product := toProduct(n)

	if len(product.Variants) != 1 {
		t.Fatalf("len(variants) = %d, want 1", len(product.Variants))
	}
	if len(product.Variants[0].InventoryLevels) != 0 {
		t.Errorf("inventory levels = %+v, want none", product.Variants[0].InventoryLevels)
	}
}
