package exportformat

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/ScreenStaring/shopify-dev-tools/cmd/products/gql"
)

type JSON struct {
	out      *os.File
	root     string
	products []gql.Product
}

var JSONRootProperties = []string{
	"barcode",
	"product_id",
	"product_title",
	"handle",
	"variant_id",
	"sku",
}

func isValidJSONRootProperty(name string) bool {
	for _, property := range JSONRootProperties {
		if name == property {
			return true
		}
	}

	return false
}

func productMap(product gql.Product) map[string]interface{} {
	return map[string]interface{}{
		"product_id":    strconv.FormatInt(product.ID, 10),
		"handle":        product.Handle,
		"product_title": product.Title,
		"product_type":  product.ProductType,
	}
}

func variantMap(variant gql.Variant) map[string]string {
	return map[string]string{
		"barcode":       variant.Barcode,
		"variant_id":    strconv.FormatInt(variant.ID, 10),
		"variant_title": variant.Title,
		"sku":           variant.SKU,
	}
}

func (j *JSON) formatForOutput() interface{} {
	if j.root == "variant_id" || j.root == "sku" || j.root == "barcode" {
		return j.formatWithVariantRoot()
	}

	return j.formatWithProduct()
}

func (j *JSON) formatWithVariantRoot() map[string]interface{} {
	output := make(map[string]interface{})

	for _, product := range j.products {
		for _, variant := range product.Variants {
			record := variantMap(variant)
			key := record[j.root]
			if len(key) == 0 {
				continue
			}

			for k, value := range productMap(product) {
				svalue, ok := value.(string)
				if !ok {
					panic(fmt.Sprintf("Cannot convert product property '%s' to string for product '%s'", k, product.Title))
				}

				record[k] = svalue
			}

			output[key] = record
		}
	}

	return output
}

func (j *JSON) formatWithProduct() interface{} {
	if len(j.root) > 0 {
		return j.formatWithProductRoot()
	}

	var output []map[string]interface{}

	for _, product := range j.products {
		record := productMap(product)

		var variants []map[string]string
		for _, variant := range product.Variants {
			variants = append(variants, variantMap(variant))
		}

		record["variants"] = variants
		output = append(output, record)
	}

	return output
}

func (j *JSON) formatWithProductRoot() map[string]interface{} {
	output := make(map[string]interface{})

	for _, product := range j.products {
		record := productMap(product)

		var variants []map[string]string
		for _, variant := range product.Variants {
			variants = append(variants, variantMap(variant))
		}

		record["variants"] = variants

		key, ok := record[j.root].(string)
		if !ok {
			panic(fmt.Sprintf("Cannot convert JSON root property '%s' to string for product '%s'", j.root, product.Title))
		}

		output[key] = record
	}

	return output
}

func NewJSON(shop string, jsonRoot string) (*JSON, error) {
	if len(jsonRoot) > 0 && !isValidJSONRootProperty(jsonRoot) {
		return nil, fmt.Errorf("Invalid JSON root property: %s", jsonRoot)
	}

	out, err := os.Create(shop + ".json")
	if err != nil {
		return nil, fmt.Errorf("Failed to create JSON file: %s", err)
	}

	return &JSON{out: out, root: jsonRoot}, nil
}

func (j *JSON) Dump(product gql.Product) error {
	j.products = append(j.products, product)

	return nil
}

func (j *JSON) Close() error {
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
