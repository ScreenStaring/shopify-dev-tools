package exportformat

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"

	"github.com/ScreenStaring/shopify-dev-tools/cmd/products/gql"
)

type CSV struct {
	out           *csv.Writer
	file          *os.File
	header        []string
	headerWritten bool
}

func NewCSV(shop string) (*CSV, error) {
	c := new(CSV)

	file, err := os.Create(shop + ".csv")
	if err != nil {
		return nil, fmt.Errorf("Failed to create CSV file: %s", err)
	}

	c.file = file
	c.out = csv.NewWriter(c.file)
	c.header = []string{
		"Product ID",
		"Product Title",
		"Product Type",
		"Variant ID",
		"Variant Title",
		"SKU",
		"Barcode",
		"Handle",
	}

	return c, nil
}

func (c *CSV) Dump(product gql.Product) error {
	if !c.headerWritten {
		c.out.Write(c.header)
		c.headerWritten = true
	}

	for _, variant := range product.Variants {
		row := []string{
			strconv.FormatInt(product.ID, 10),
			product.Title,
			product.ProductType,
			strconv.FormatInt(variant.ID, 10),
			variant.Title,
			variant.SKU,
			variant.Barcode,
			product.Handle,
		}

		if err := c.out.Write(row); err != nil {
			return err
		}
	}

	return nil
}

func (c *CSV) Close() error {
	defer c.file.Close()

	c.out.Flush()

	return c.out.Error()
}
