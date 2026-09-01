package exportformat

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/ScreenStaring/shopify-dev-tools/cmd/products/gql"
)

// JSONRootProperties are the valid --json-root values for the product IDs
// export.
var JSONRootProperties = []string{
	"barcode",
	"product_id",
	"product_title",
	"handle",
	"variant_id",
	"sku",
}

// FlatRecord is a child-level record (e.g. a variant) keyed by a flat root
// property, with the parent context already merged in.
type FlatRecord struct {
	Key    string
	Record map[string]interface{}
}

// JSON writes records as a JSON array, or as a map keyed by a root property.
// parent returns the full record (children nested) used by the array and
// parent-level root modes. flatten returns the child-level records used by
// the flatRoots modes, given the root property being used as the key.
type JSON[T any] struct {
	out       *os.File
	root      string
	rootProps []string
	flatRoots []string
	items     []T
	parent    func(T) map[string]interface{}
	flatten   func(T, string) []FlatRecord
}

func NewJSON[T any](fileName, root string, rootProps, flatRoots []string, parent func(T) map[string]interface{}, flatten func(T, string) []FlatRecord) (*JSON[T], error) {
	if len(root) > 0 && !contains(rootProps, root) {
		return nil, fmt.Errorf("Invalid JSON root property: %s", root)
	}

	out, err := os.Create(fileName)
	if err != nil {
		return nil, fmt.Errorf("Failed to create JSON file: %s", err)
	}

	return &JSON[T]{out: out, root: root, rootProps: rootProps, flatRoots: flatRoots, parent: parent, flatten: flatten}, nil
}

func contains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}

	return false
}

func (j *JSON[T]) Dump(item T) error {
	j.items = append(j.items, item)

	return nil
}

func (j *JSON[T]) Close() error {
	defer j.out.Close()

	out, err := json.Marshal(j.formatForOutput())
	if err != nil {
		return err
	}

	n, err := j.out.Write(out)
	if err != nil {
		return err
	}

	if n != len(out) {
		return fmt.Errorf("Was only able to write %d/%d bytes to JSON file", n, len(out))
	}

	return nil
}

func (j *JSON[T]) formatForOutput() interface{} {
	if len(j.root) > 0 {
		if contains(j.flatRoots, j.root) {
			return j.formatWithFlatRoot()
		}

		return j.formatWithParentRoot()
	}

	return j.formatAsArray()
}

func (j *JSON[T]) formatAsArray() interface{} {
	output := make([]map[string]interface{}, 0, len(j.items))

	for _, item := range j.items {
		output = append(output, j.parent(item))
	}

	return output
}

func (j *JSON[T]) formatWithParentRoot() map[string]interface{} {
	output := make(map[string]interface{})

	for _, item := range j.items {
		record := j.parent(item)

		key, ok := record[j.root].(string)
		if !ok {
			panic(fmt.Sprintf("Cannot convert JSON root property '%s' to string for record with product_id '%v'", j.root, record["product_id"]))
		}

		output[key] = record
	}

	return output
}

func (j *JSON[T]) formatWithFlatRoot() map[string]interface{} {
	output := make(map[string]interface{})

	for _, item := range j.items {
		for _, flat := range j.flatten(item, j.root) {
			if len(flat.Key) == 0 {
				continue
			}

			output[flat.Key] = flat.Record
		}
	}

	return output
}

func ProductMap(product gql.Product) map[string]interface{} {
	return map[string]interface{}{
		"product_id":    strconv.FormatInt(product.ID, 10),
		"handle":        product.Handle,
		"product_title": product.Title,
		"product_type":  product.ProductType,
	}
}

func VariantMap(variant gql.Variant) map[string]interface{} {
	return map[string]interface{}{
		"barcode":       variant.Barcode,
		"variant_id":    strconv.FormatInt(variant.ID, 10),
		"variant_title": variant.Title,
		"sku":           variant.SKU,
	}
}
