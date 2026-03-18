package export

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/ScreenStaring/shopify-dev-tools/cmd"
	"github.com/ScreenStaring/shopify-dev-tools/cmd/products/gql"
)

var validIdentifyBy = []string{"id", "sku", "barcode"}

func readIdentifiers() ([]string, error) {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, fmt.Errorf("Cannot read from stdin: %s", err)
	}

	var ids []string
	for _, line := range strings.Split(strings.TrimRight(string(data), "\n"), "\n") {
		if line != "" {
			ids = append(ids, line)
		}
	}

	return ids, nil
}

func Inventory(c *cli.Context) error {
	shop := c.String("shop")
	token := cmd.LookupAccessToken(shop, c.String("access-token"))
	options := map[string]interface{}{"version": c.String("api-version")}
	baseName := shopBaseName(shop)

	identifyBy := c.String("identify-by")

	if identifyBy != "" {
		valid := false
		for _, v := range validIdentifyBy {
			if identifyBy == v {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("--identify-by must be one of: %s", strings.Join(validIdentifyBy, ", "))
		}
	}

	file, err := os.Create(baseName + "-inventory.csv")
	if err != nil {
		return fmt.Errorf("Failed to create CSV file: %s", err)
	}
	defer file.Close()

	w := csv.NewWriter(file)

	if err := w.Write([]string{
		"Variant ID",
		"Variant Title",
		"Product ID",
		"Product Title",
		"SKU",
		"Barcode",
		"Location",
		"Available",
		"On Hand",
	}); err != nil {
		return err
	}

	writeRow := func(pi gql.ProductInventory) error {
		for _, vi := range pi.Variants {
			for _, level := range vi.InventoryLevels {
				row := []string{
					strconv.FormatInt(vi.VariantID, 10),
					vi.VariantTitle,
					strconv.FormatInt(pi.ProductID, 10),
					pi.ProductTitle,
					vi.SKU,
					vi.Barcode,
					level.Location,
					strconv.Itoa(level.Available),
					strconv.Itoa(level.OnHand),
				}

				if err := w.Write(row); err != nil {
					return fmt.Errorf("Cannot write variant %d: %s", vi.VariantID, err)
				}
			}
		}

		return nil
	}

	count := 0

	if identifyBy != "" {
		identifiers, err := readIdentifiers()
		if err != nil {
			return err
		}

		fmt.Fprintf(os.Stderr, "Exporting inventory for %d identifiers to CSV...\n", len(identifiers))

		err = gql.FetchInventoryByIdentifiers(shop, token, identifyBy, identifiers, func(pi gql.ProductInventory) error {
			if err := writeRow(pi); err != nil {
				return err
			}

			count++
			fmt.Fprintf(os.Stderr, "\rProcessing %d", count)

			return nil
		}, options)
	} else {
		total, err := gql.FetchProductCount(shop, token, "", options)
		if err != nil {
			return err
		}

		fmt.Fprintf(os.Stderr, "Exporting inventory for %d products to CSV...\n", total)

		err = gql.FetchAllInventory(shop, token, func(pi gql.ProductInventory) error {
			if err := writeRow(pi); err != nil {
				return err
			}

			count++
			fmt.Fprintf(os.Stderr, "\rProcessing %d/%d", count, total)

			return nil
		}, options)
	}

	if err != nil {
		return err
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return err
	}

	fmt.Fprintln(os.Stderr, "\nComplete!")

	return nil
}
