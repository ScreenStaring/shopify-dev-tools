package exportformat

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ScreenStaring/shopify-dev-tools/cmd/products/gql"
)

func newTestJSON(t *testing.T, root string) *JSON[gql.Product] {
	t.Helper()

	file := filepath.Join(t.TempDir(), "out.json")

	j, err := NewJSON(file, root,
		JSONRootProperties,
		[]string{"variant_id", "sku", "barcode"},
		func(p gql.Product) map[string]interface{} {
			record := ProductMap(p)

			var variants []map[string]interface{}
			for _, v := range p.Variants {
				variants = append(variants, VariantMap(v))
			}

			record["variants"] = variants

			return record
		},
		func(p gql.Product, root string) []FlatRecord {
			var out []FlatRecord

			for _, v := range p.Variants {
				record := VariantMap(v)

				for k, value := range ProductMap(p) {
					record[k] = value
				}

				out = append(out, FlatRecord{Key: record[root].(string), Record: record})
			}

			return out
		})
	if err != nil {
		t.Fatalf("NewJSON failed: %s", err)
	}

	return j
}

func dumpProduct(j *JSON[gql.Product]) error {
	p := gql.Product{
		ID:          1,
		Title:       "Test Product",
		ProductType: "Shirt",
		Handle:      "test-product",
		Variants: []gql.Variant{
			{ID: 10, Title: "Red", SKU: "RED-1", Barcode: "111"},
			{ID: 11, Title: "Blue", SKU: "BLUE-1", Barcode: "222"},
		},
	}

	return j.Dump(p)
}

func readJSON(t *testing.T, j *JSON[gql.Product]) interface{} {
	t.Helper()

	if err := dumpProduct(j); err != nil {
		t.Fatalf("Dump failed: %s", err)
	}

	if err := j.Close(); err != nil {
		t.Fatalf("Close failed: %s", err)
	}

	b, err := os.ReadFile(j.out.Name())
	if err != nil {
		t.Fatalf("Cannot read JSON file: %s", err)
	}

	var out interface{}
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("Cannot parse JSON: %s", err)
	}

	return out
}

func TestJSONArray(t *testing.T) {
	j := newTestJSON(t, "")

	out := readJSON(t, j).([]interface{})

	if len(out) != 1 {
		t.Fatalf("len = %d, want 1", len(out))
	}

	record := out[0].(map[string]interface{})
	if record["product_id"] != "1" || record["handle"] != "test-product" {
		t.Errorf("record = %v, want product_id 1/handle test-product", record)
	}

	variants := record["variants"].([]interface{})
	if len(variants) != 2 {
		t.Fatalf("len(variants) = %d, want 2", len(variants))
	}
}

func TestJSONParentRoot(t *testing.T) {
	j := newTestJSON(t, "handle")

	out := readJSON(t, j).(map[string]interface{})

	if len(out) != 1 {
		t.Fatalf("len = %d, want 1", len(out))
	}

	if _, ok := out["test-product"]; !ok {
		t.Errorf("output = %v, want key test-product", out)
	}
}

func TestJSONFlatRoot(t *testing.T) {
	j := newTestJSON(t, "sku")

	out := readJSON(t, j).(map[string]interface{})

	if len(out) != 2 {
		t.Fatalf("len = %d, want 2", len(out))
	}

	record, ok := out["RED-1"].(map[string]interface{})
	if !ok {
		t.Fatalf("output = %v, want key RED-1", out)
	}

	if record["variant_id"] != "10" || record["product_id"] != "1" {
		t.Errorf("record = %v, want variant_id 10/product_id 1", record)
	}
}

func TestJSONInvalidRoot(t *testing.T) {
	_, err := NewJSON[gql.Product](filepath.Join(t.TempDir(), "out.json"), "bogus",
		JSONRootProperties, []string{"sku"}, nil, nil)
	if err == nil {
		t.Fatal("NewJSON with invalid root: want error, got nil")
	}
}
