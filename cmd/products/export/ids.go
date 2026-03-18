package export

import (
	"fmt"
	"os"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/ScreenStaring/shopify-dev-tools/cmd"
	"github.com/ScreenStaring/shopify-dev-tools/cmd/products/exportformat"
	"github.com/ScreenStaring/shopify-dev-tools/cmd/products/gql"
)

type dumper interface {
	Dump(gql.Product) error
	Close() error
}

func shopBaseName(shop string) string {
	return strings.SplitN(shop, ".", 2)[0]
}

func IDs(c *cli.Context) error {
	shop := c.String("shop")
	token := cmd.LookupAccessToken(shop, c.String("access-token"))
	status := c.String("status")
	baseName := shopBaseName(shop)

	exportFormat := "CSV"
	if c.Bool("json") {
		exportFormat = "JSON"
	}

	options := map[string]interface{}{"version": c.String("api-version")}

	total, err := gql.FetchProductCount(shop, token, status, options)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Exporting %d products to %s...\n", total, exportFormat)

	var d dumper

	if c.Bool("json") {
		d, err = exportformat.NewJSON(baseName, c.String("json-root"))
	} else {
		d, err = exportformat.NewCSV(baseName)
	}

	if err != nil {
		return err
	}

	count := 0
	err = gql.FetchAllProducts(shop, token, status, func(product gql.Product) error {
		if err := d.Dump(product); err != nil {
			return fmt.Errorf("Cannot write product %d: %s", product.ID, err)
		}

		count++
		fmt.Fprintf(os.Stderr, "\rProcessing %d/%d", count, total)

		return nil
	}, options)

	if err != nil {
		return err
	}

	if err := d.Close(); err != nil {
		return err
	}

	fmt.Fprintln(os.Stderr, "\nComplete!")

	return nil
}
