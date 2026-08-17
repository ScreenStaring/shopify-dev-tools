package products

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"

	"github.com/cheynewallace/tabby"
	"github.com/urfave/cli/v2"

	"github.com/ScreenStaring/shopify-dev-tools/cmd"
	"github.com/ScreenStaring/shopify-dev-tools/cmd/products/export"
	"github.com/ScreenStaring/shopify-dev-tools/cmd/products/exportformat"
	"github.com/ScreenStaring/shopify-dev-tools/cmd/products/gql"
)

var Cmd cli.Command

// PrintJSONL outputs the products one JSON object per line.
func PrintJSONL(products []gql.Product) {
	for _, product := range products {
		line, err := json.Marshal(product)
		if err != nil {
			panic(err)
		}

		fmt.Println(string(line))
	}
}

func displayFieldName(name string) string {
	var result strings.Builder
	for i, r := range name {
		if i > 0 && r >= 'A' && r <= 'Z' {
			prev := name[i-1]
			if prev >= 'a' && prev <= 'z' {
				result.WriteByte(' ')
			} else if prev >= 'A' && prev <= 'Z' && i+1 < len(name) && name[i+1] >= 'a' && name[i+1] <= 'z' {
				result.WriteByte(' ')
			}
		}
		result.WriteRune(r)
	}
	return result.String()
}

func normalizeField(name string) string {
	return strings.ReplaceAll(strings.ToLower(name), " ", "")
}

func isFieldToPrint(field string, selectedFields []string) bool {
	for _, f := range selectedFields {
		if f == field {
			return true
		}
	}

	return false
}

func printOptions(options []gql.ProductOption) {
	t := tabby.New()
	t.AddHeader("Name", "Values")

	for _, opt := range options {
		t.AddLine(opt.Name, strings.Join(opt.Values, ", "))
	}

	t.Print()
}

func printVariants(variants []gql.Variant) {
	t := tabby.New()
	t.AddHeader("ID", "Title", "SKU", "Barcode", "Price", "Compare At Price", "Inventory")

	for _, v := range variants {
		t.AddLine(v.ID, v.Title, v.SKU, v.Barcode, v.Price, v.CompareAtPrice, v.InventoryQuantity)
	}

	t.Print()
}

// PrintFormatted outputs the products in the tabular format used by
// `products ls`: one block per product with its fields, options, and
// variants. fieldsToPrint selects which product fields to print; an
// empty slice prints all of them.
func PrintFormatted(products []gql.Product, fieldsToPrint []string) {
	t := tabby.New()
	normalizedFieldsToPrint := []string{}

	for _, field := range fieldsToPrint {
		normalizedFieldsToPrint = append(normalizedFieldsToPrint, normalizeField(field))
	}

	for _, product := range products {
		s := reflect.ValueOf(&product).Elem()

		for i := 0; i < s.NumField(); i++ {
			field := s.Type().Field(i).Name
			normalizedField := normalizeField(field)
			label := displayFieldName(field)

			if normalizedField == "options" || normalizedField == "variants" || normalizedField == "hasonlydefaultvariant" {
				continue
			}

			if len(fieldsToPrint) > 0 {
				if isFieldToPrint(normalizedField, normalizedFieldsToPrint) {
					t.AddLine(label, s.Field(i).Interface())
				}
			} else {
				t.AddLine(label, s.Field(i).Interface())
			}
		}

		t.Print()

		showAll := len(fieldsToPrint) == 0

		if (showAll && !product.HasOnlyDefaultVariant && len(product.Options) > 0) || (!showAll && isFieldToPrint("options", normalizedFieldsToPrint)) {
			fmt.Println("Options")
			printOptions(product.Options)
			fmt.Print("\n")
		}

		if showAll || isFieldToPrint("variants", normalizedFieldsToPrint) {
			fmt.Println("Variants")
			printVariants(product.Variants)
			fmt.Print("\n")
		}

		cmd.PrintSeparator()
	}
}

// PrintInventory shows each product's ID and title, then an "Inventories"
// section with one block per variant: variant ID and title, then a horizontal
// table of inventory quantities per location.
func PrintInventory(products []gql.Product) {
	for _, product := range products {
		t := tabby.New()
		t.AddLine("ID", product.ID)
		t.AddLine("Title", product.Title)
		t.Print()

		fmt.Println("Inventories")

		for _, v := range product.Variants {
			t := tabby.New()
			t.AddLine("Variant ID", v.ID)
			t.AddLine("Variant Title", v.Title)
			t.AddLine("SKU", v.SKU)
			t.AddLine("Barcode", v.Barcode)
			t.Print()

			t = tabby.New()
			t.AddHeader("Location", "Unavailable", "Committed", "Available", "On Hand")
			for _, level := range v.InventoryLevels {
				unavailable := level.OnHand - level.Available - level.Committed
				t.AddLine(level.Location, unavailable, level.Committed, level.Available, level.OnHand)
			}
			t.Print()
			fmt.Print("\n")
		}

		cmd.PrintSeparator()
	}
}

func toProductGID(id string) string {
	if strings.HasPrefix(id, "gid://") {
		return id
	}
	return "gid://shopify/Product/" + id
}

func deleteProducts(c *cli.Context) error {
	var ids []string

	if c.NArg() > 0 {
		for i := 0; i < c.NArg(); i++ {
			ids = append(ids, c.Args().Get(i))
		}
	} else {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("Cannot read from stdin: %s", err)
		}

		for _, line := range strings.Split(strings.TrimRight(string(data), "\n"), "\n") {
			if line != "" {
				ids = append(ids, line)
			}
		}
	}

	shop := c.String("shop")
	token := cmd.LookupAccessToken(shop, c.String("access-token"))
	options := map[string]interface{}{"version": c.String("api-version")}

	for _, id := range ids {
		gid := toProductGID(id)
		result, err := gql.ProductDelete(shop, token, gid, options)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error deleting product %s: %s\n", id, err)
			continue
		}

		if len(result.UserErrors) > 0 {
			fmt.Fprintf(os.Stderr, "Error deleting product %s: %s\n", id, strings.Join(result.UserErrors, ", "))
			continue
		}

		fmt.Println("Deleted %s\n", result.DeletedProductID)
	}

	return nil
}

func listProducts(c *cli.Context) error {
	ids, skus, err := cmd.ParseIDArgs(c, "a product id")
	if err != nil {
		return err
	}

	var fields []string

	if len(c.String("fields")) > 0 {
		fields = strings.Split(c.String("fields"), ",")
	}

	shop := c.String("shop")
	options := map[string]interface{}{"version": c.String("api-version")}
	products, err := gql.FetchProducts(shop, cmd.LookupAccessToken(shop, c.String("access-token")), ids, skus, c.String("status"), int(c.Int64("limit")), options)
	if err != nil {
		return err
	}

	if c.Bool("jsonl") {
		PrintJSONL(products)
	} else {
		PrintFormatted(products, fields)
	}

	return nil
}

func inventoryProducts(c *cli.Context) error {
	ids, skus, err := cmd.ParseIDArgs(c, "a product id")
	if err != nil {
		return err
	}

	if len(ids) == 0 && len(skus) == 0 {
		return fmt.Errorf("No product IDs or 'sku:VALUE' arguments given")
	}

	shop := c.String("shop")
	token := cmd.LookupAccessToken(shop, c.String("access-token"))
	options := map[string]interface{}{"version": c.String("api-version")}

	// Resolve sku: arguments to product IDs first; the inventory query is per product.
	if len(skus) > 0 {
		products, err := gql.FetchProducts(shop, token, nil, skus, "", 250, options)
		if err != nil {
			return err
		}
		for _, p := range products {
			ids = append(ids, p.ID)
		}
	}

	var result []gql.Product
	for _, id := range ids {
		product, err := gql.FetchProductInventory(shop, token, id, options)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching inventory for product %d: %s\n", id, err)
			continue
		}
		result = append(result, *product)
	}

	if len(result) > 0 {
		PrintInventory(result)
	}

	return nil
}

func init() {
	apiVersionFlag := &cli.StringFlag{
		Name:  "api-version",
		Usage: "API version to use; default is a versionless call",
	}

	productFlags := []cli.Flag{
		// &cli.StringSliceFlag{
		// 	Name:    "order",
		// 	Aliases: []string{"o"},
		// 	Usage:   "Order products by the given properties",
		// },
		&cli.StringFlag{
			Name:    "fields",
			Aliases: []string{"f"},
			Usage:   "Comma separated list of fields to output",
			EnvVars: []string{"SHOPIFY_PRODUCT_FIELDS"},
		},
		&cli.Int64Flag{
			Name:    "limit",
			Aliases: []string{"l"},
			Value:   10,
		},
		&cli.StringFlag{
			Name:    "status",
			Aliases: []string{"s"},
		},
		&cli.BoolFlag{
			Name:    "jsonl",
			Aliases: []string{"j"},
			Usage:   "Output the products in JSONL format",
		},
		apiVersionFlag,
	}

	identifyByFlag := &cli.StringFlag{
		Name:    "identify-by",
		Aliases: []string{"i"},
		Usage:   "Identifier property for productSet: 'id' or 'handle'",
	}

	Cmd = cli.Command{
		Name:    "products",
		Aliases: []string{"p"},
		Usage:   "Do things with products",
		Subcommands: []*cli.Command{
			{
				Name:      "ls",
				Aliases:   []string{"l"},
				Usage:     "List some of a shop's products or the products matching the given IDs and/or 'sku:VALUE' arguments",
				ArgsUsage: "[ID|sku:VALUE [ID|sku:VALUE ...]]",
				Flags:     append(cmd.Flags, productFlags...),
				Action:    listProducts,
			},
			{
				Name:      "inventory",
				Aliases:   []string{"inv"},
				Usage:     "List per-location inventory quantities for the given products",
				ArgsUsage: "[ID|sku:VALUE [ID|sku:VALUE ...]]",
				Flags:     append(cmd.Flags, apiVersionFlag),
				Action:    inventoryProducts,
			},
			{
				Name:        "delete",
				Aliases:     []string{"d"},
				Usage:       "Delete products by ID",
				ArgsUsage:   "[ID [ID ...]]",
				Description: "If IDs are not given they're read from stdin one per line",
				Flags:       append(cmd.Flags, apiVersionFlag),
				Action:      deleteProducts,
			},
			{
				Name:      "import",
				Aliases:   []string{"i"},
				Usage:     "Import products synchronously from a Shopify CSV file",
				ArgsUsage: "products.csv",
				Flags: append(cmd.Flags,
					identifyByFlag,
					&cli.IntFlag{
						Name:    "parallel",
						Aliases: []string{"p"},
						Value:   5,
						Usage:   "Number of parallel API calls to make",
					},
					&cli.BoolFlag{
						Name:    "json",
						Aliases: []string{"j"},
						Usage:   "Output the results in JSON format",
					},
					apiVersionFlag,
				),
				Action: syncImportProducts,
			},
			{
				Name:    "export",
				Aliases: []string{"e", "x"},
				Usage:   "Export product data",
				Subcommands: []*cli.Command{
					{
						Name:    "ids",
						Aliases: []string{"i"},
						Usage:   "Export product and variant IDs, and other identifiers, to a CSV or JSON file",
						Flags: append(cmd.Flags,
							&cli.StringFlag{
								Name:    "status",
								Aliases: []string{"s"},
							},
							&cli.BoolFlag{
								Name:    "json",
								Aliases: []string{"j"},
								Usage:   "Output in JSON format",
							},
							&cli.StringFlag{
								Name:    "json-root",
								Aliases: []string{"r"},
								Usage:   fmt.Sprintf("Top-level property for JSON output, one of: %s", strings.Join(exportformat.JSONRootProperties, ", ")),
							},
						),
						Action: export.IDs,
					},
					{
						Name:    "inventory",
						Aliases: []string{"inv"},
						Usage:   "Export inventory quantities by variant and location to a CSV file",
						Flags: append(cmd.Flags,
							apiVersionFlag,
							&cli.StringFlag{
								Name:    "identify-by",
								Aliases: []string{"i"},
								Usage:   "Read identifiers from stdin and only export inventory for matching variants; one of: id, sku, barcode",
							},
						),
						Action: export.Inventory,
					},
				},
			},
			{
				Name:    "bulk",
				Aliases: []string{"b"},
				Usage:   "Import products from a Shopify CSV file using the Bulk API",
				Subcommands: []*cli.Command{
					{
						Name:      "import",
						Aliases:   []string{"i"},
						Usage:     "Import a Shopify CSV file",
						ArgsUsage: "products.csv",
						Flags:     append(cmd.Flags, identifyByFlag),
						Action:    importProducts,
					},
					{
						Name:      "status",
						Aliases:   []string{"s"},
						Usage:     "Check the status of a bulk import operation",
						ArgsUsage: "<operation-id>",
						Flags:     cmd.Flags,
						Action:    importStatus,
					},
					{
						Name:      "cancel",
						Aliases:   []string{"c"},
						Usage:     "Cancel a running bulk import operation",
						ArgsUsage: "<operation-id>",
						Flags:     cmd.Flags,
						Action:    cancelBulkOperation,
					},
				},
			},
		},
	}
}
