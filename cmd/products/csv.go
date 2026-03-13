package products

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

type optionValueInput struct {
	Name string `json:"name"`
}

type optionCreateInput struct {
	Name   string             `json:"name"`
	Values []optionValueInput `json:"values"`
}

type variantOptionValue struct {
	OptionName string `json:"optionName"`
	Name       string `json:"name"`
}

type inventoryItemInput struct {
	Tracked          *bool `json:"tracked,omitempty"`
	RequiresShipping *bool `json:"requiresShipping,omitempty"`
}

type inventoryQuantityInput struct {
	LocationID string `json:"locationId"`
	Name       string `json:"name"`
	Quantity   int    `json:"quantity"`
}

type importVariant struct {
	OptionValues        []variantOptionValue     `json:"optionValues,omitempty"`
	SKU                 string                   `json:"sku,omitempty"`
	Price               string                   `json:"price,omitempty"`
	CompareAtPrice      string                   `json:"compareAtPrice,omitempty"`
	Barcode             string                   `json:"barcode,omitempty"`
	Taxable             *bool                    `json:"taxable,omitempty"`
	InventoryPolicy     string                   `json:"inventoryPolicy,omitempty"`
	InventoryItem       *inventoryItemInput      `json:"inventoryItem,omitempty"`
	InventoryQuantities []inventoryQuantityInput `json:"inventoryQuantities,omitempty"`
}

type fileInput struct {
	OriginalSource string `json:"originalSource"`
	ContentType    string `json:"contentType"`
}

type importProduct struct {
	ID              string              `json:"-"`
	Handle          string              `json:"handle,omitempty"`
	Title           string              `json:"title,omitempty"`
	DescriptionHTML string              `json:"descriptionHtml,omitempty"`
	Vendor          string              `json:"vendor,omitempty"`
	ProductType     string              `json:"productType,omitempty"`
	Tags            []string            `json:"tags,omitempty"`
	Status          string              `json:"status,omitempty"`
	Files           []fileInput         `json:"files,omitempty"`
	ProductOptions  []optionCreateInput `json:"productOptions,omitempty"`
	Variants        []importVariant     `json:"variants,omitempty"`
}

type productSetIdentifier struct {
	ID     string `json:"id,omitempty"`
	Handle string `json:"handle,omitempty"`
}

type importProductInput struct {
	Input      importProduct        `json:"input"`
	Identifier *productSetIdentifier `json:"identifier,omitempty"`
}

func buildColumnIndex(header []string) map[string]int {
	idx := make(map[string]int)
	for i, h := range header {
		idx[strings.ToLower(strings.TrimSpace(h))] = i
	}
	return idx
}

func colVal(row []string, idx int) string {
	if idx < 0 || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}

func buildProductOptions(optionNames []string, optionValues [][]string) []optionCreateInput {
	var opts []optionCreateInput
	for i, name := range optionNames {
		var values []optionValueInput
		if i < len(optionValues) {
			for _, v := range optionValues[i] {
				values = append(values, optionValueInput{Name: v})
			}
		}
		if len(values) == 0 {
			continue
		}
		opts = append(opts, optionCreateInput{Name: name, Values: values})
	}
	return opts
}

func parseBoolPtr(s string) *bool {
	if s == "" {
		return nil
	}
	v := strings.EqualFold(s, "true")
	return &v
}

func parseCSV(filename string, locations map[string]string) ([]importProductInput, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("Cannot open CSV file: %s", err)
	}
	defer f.Close()

	reader := csv.NewReader(f)

	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("Cannot read CSV header: %s", err)
	}

	ci := buildColumnIndex(header)

	get := func(row []string, name string) string {
		if idx, ok := ci[name]; ok {
			return colVal(row, idx)
		}
		return ""
	}

	var products []importProductInput
	var current *importProduct
	var optionNames []string
	var optionValueCols []string
	var optionValues [][]string
	var optionSeen []map[string]bool

	finalize := func() {
		if current == nil {
			return
		}

		if len(optionNames) > 0 {
			current.ProductOptions = buildProductOptions(optionNames, optionValues)
		} else {
			current.ProductOptions = []optionCreateInput{
				{Name: "Title", Values: []optionValueInput{{Name: "Default Title"}}},
			}

			if len(current.Variants) == 0 {
				current.Variants = []importVariant{
					{OptionValues: []variantOptionValue{{OptionName: "Title", Name: "Default Title"}}},
				}
			} else {
				for i := range current.Variants {
					current.Variants[i].OptionValues = append(
						[]variantOptionValue{{OptionName: "Title", Name: "Default Title"}},
						current.Variants[i].OptionValues...,
					)
				}
			}
		}

		products = append(products, importProductInput{Input: *current})
	}

	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("Cannot read CSV row: %s", err)
		}

		handle := get(row, "handle")

		if handle != "" {
			finalize()

			status := "DRAFT"
			if s := get(row, "status"); s != "" {
				status = strings.ToUpper(s)
			}

			var tags []string
			if t := get(row, "tags"); t != "" {
				for _, tag := range strings.Split(t, ",") {
					tag = strings.TrimSpace(tag)
					if tag != "" {
						tags = append(tags, tag)
					}
				}
			}

			optionNames = nil
			optionValueCols = nil
			optionValues = nil
			optionSeen = nil
			nameColumns := []string{"option1 name", "option2 name", "option3 name"}
			valueColumns := []string{"option1 value", "option2 value", "option3 value"}
			for idx, nameCol := range nameColumns {
				if v := get(row, nameCol); v != "" {
					optionNames = append(optionNames, v)
					optionValueCols = append(optionValueCols, valueColumns[idx])
					optionValues = append(optionValues, nil)
					optionSeen = append(optionSeen, make(map[string]bool))
				}
			}

			var files []fileInput
			if u := get(row, "product image url"); u != "" {
				files = append(files, fileInput{OriginalSource: u, ContentType: "IMAGE"})
			}

			current = &importProduct{
				ID:              get(row, "id"),
				Handle:          handle,
				Title:           get(row, "title"),
				DescriptionHTML: get(row, "body (html)"),
				Vendor:          get(row, "vendor"),
				ProductType:     get(row, "type"),
				Tags:            tags,
				Status:          status,
				Files:           files,
			}

		}

		if current == nil {
			continue
		}

		// Collect option values for dedup and build variant
		var variantOpts []variantOptionValue
		for i, valCol := range optionValueCols {
			if val := get(row, valCol); val != "" {
				if !optionSeen[i][val] {
					optionSeen[i][val] = true
					optionValues[i] = append(optionValues[i], val)
				}
				variantOpts = append(variantOpts, variantOptionValue{
					OptionName: optionNames[i],
					Name:       val,
				})
			}
		}

		sku := get(row, "variant sku")
		price := get(row, "variant price")
		compareAt := get(row, "variant compare at price")
		barcode := get(row, "variant barcode")
		taxable := get(row, "variant taxable")
		inventoryPolicy := get(row, "variant inventory policy")

		if inventoryPolicy == "" {
			if v := get(row, "continue selling when out of stock"); v != "" {
				if strings.EqualFold(v, "true") {
					inventoryPolicy = "CONTINUE"
				} else {
					inventoryPolicy = "DENY"
				}
			}
		}

		var inventoryItem *inventoryItemInput
		if v := get(row, "requires shipping"); v != "" {
			inventoryItem = &inventoryItemInput{RequiresShipping: parseBoolPtr(v)}
		}

		var inventoryQuantities []inventoryQuantityInput
		location := get(row, "location")
		if location != "" {
			locationID, ok := locations[location]
			if !ok {
				return nil, fmt.Errorf("Unknown location %q", location)
			}

			for _, iqType := range []struct {
				column string
				name   string
			}{
				{"available", "available"},
				{"on hand", "on_hand"},
			} {
				if qtyStr := get(row, iqType.column); qtyStr != "" {
					qty, err := strconv.Atoi(qtyStr)
					if err != nil {
						return nil, fmt.Errorf("Invalid %s value %q: %s", iqType.column, qtyStr, err)
					}
					inventoryQuantities = []inventoryQuantityInput{
						{
							LocationID: locationID,
							Name:       iqType.name,
							Quantity:   qty,
						},
					}
					break
				}
			}
		}

		if len(inventoryQuantities) > 0 {
			tracked := true
			if inventoryItem == nil {
				inventoryItem = &inventoryItemInput{}
			}
			inventoryItem.Tracked = &tracked
		}

		hasVariantData := len(variantOpts) > 0 || sku != "" || price != ""
		if hasVariantData {
			v := importVariant{
				OptionValues:        variantOpts,
				SKU:                 sku,
				Price:               price,
				CompareAtPrice:      compareAt,
				Barcode:             barcode,
				Taxable:             parseBoolPtr(taxable),
				InventoryPolicy:     inventoryPolicy,
				InventoryItem:       inventoryItem,
				InventoryQuantities: inventoryQuantities,
			}
			current.Variants = append(current.Variants, v)
		}
	}

	finalize()

	return products, nil
}
