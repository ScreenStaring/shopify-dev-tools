package products

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/cheynewallace/tabby"
	"github.com/urfave/cli/v2"

	"github.com/ScreenStaring/shopify-dev-tools/cmd"
	"github.com/ScreenStaring/shopify-dev-tools/cmd/products/gql"
)

var Cmd cli.Command

func printJSONL(products []gql.Product) {
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

func printFormatted(products []gql.Product, fieldsToPrint []string) {
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

			if normalizedField == "options" || normalizedField == "variants" {
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

		if showAll || isFieldToPrint("options", normalizedFieldsToPrint) {
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

func listProducts(c *cli.Context) error {
	var ids []int64
	var fields []string

	for i := 0; i < c.NArg(); i++ {
		id, err := strconv.ParseInt(c.Args().Get(i), 10, 64)
		if err != nil {
			return fmt.Errorf("Product id '%s' invalid: must be an int", c.Args().Get(0))
		}

		ids = append(ids, id)
	}

	if len(c.String("fields")) > 0 {
		fields = strings.Split(c.String("fields"), ",")
	}

	shop := c.String("shop")
	products, err := gql.FetchProducts(shop, cmd.LookupAccessToken(shop, c.String("access-token")), ids, c.String("status"), int(c.Int64("limit")))
	if err != nil {
		return err
	}

	if c.Bool("jsonl") {
		printJSONL(products)
	} else {
		printFormatted(products, fields)
	}

	return nil
}

func init() {
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
				Usage:     "List some of a shop's products or the products given by the specified IDs",
				ArgsUsage: "[ID [ID ...]]",
				Flags:     append(cmd.Flags, productFlags...),
				Action:    listProducts,
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
				),
				Action: syncImportProducts,
			},
			{
				Name:    "bulk",
				Aliases: []string{"b"},
				Usage:     "Import products from a Shopify CSV file using the Bulk API",
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
