package products

import (
	"encoding/json"

	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	shopify "github.com/bold-commerce/go-shopify/v3"
	"github.com/cheynewallace/tabby"
	"github.com/urfave/cli/v2"

	"github.com/ScreenStaring/shopify-dev-tools/cmd"
)

var Cmd cli.Command

// func createProducts(c *cli.Context) error {
// 	if c.NArg() == 0 {
// 		return errors.New("CSV required")
// 	}

// 	return nil
// }

type listProductOptions struct {
	Fields       []string  `url:"fields,comma,omitempty"`
	Ids          []int64   `url:"ids,comma,omitempty"`
	Limit        int64     `url:"limit,omitempty"`
	Status       string    `url:"status,omitempty"`
	UpdatedAtMin time.Time `url:"updated_at_min,omitempty"`
}

func printJSONL(products []shopify.Product) {
	for _, product := range products {
		line, err := json.Marshal(product)
		if err != nil {
			panic(err)
		}

		fmt.Println(string(line))
	}
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

func printFormatted(products []shopify.Product, fieldsToPrint []string) {
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

			if len(fieldsToPrint) > 0 {
				if isFieldToPrint(normalizedField, normalizedFieldsToPrint) {
					t.AddLine(field, s.Field(i).Interface())
				}
			} else {
				t.AddLine(field, s.Field(i).Interface())
			}
		}

		t.Print()
		fmt.Printf("%s\n", strings.Repeat("-", 20))
	}
}

func listProducts(c *cli.Context) error {
	var products []shopify.Product
	var options listProductOptions

	if c.NArg() > 0 {
		for i := 0; i < c.NArg(); i++ {
			id, err := strconv.ParseInt(c.Args().Get(i), 10, 64)
			if err != nil {
				return fmt.Errorf("Product id '%s' invalid: must be an int", c.Args().Get(0))
			}

			options.Ids = append(options.Ids, id)
		}

	} else {
		options.Limit = 10

		if len(c.String("status")) > 0 {
			options.Status = c.String("status")
		}

		if c.Int64("limit") > 0 {
			options.Limit = c.Int64("limit")
		}
	}

	if len(c.String("fields")) > 0 {
		options.Fields = strings.Split(c.String("fields"), ",")
	}

	products, err := cmd.NewShopifyClient(c).Product.List(options)
	if err != nil {
		return fmt.Errorf("Cannot list products: %s", err)
	}

	if c.Bool("jsonl") {
		printJSONL(products)
	} else {
		printFormatted(products, options.Fields)
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
		},
		&cli.Int64Flag{
			Name:    "limit",
			Aliases: []string{"l"},
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
			// {
			// 	Name: "create",
			// 	Aliases: []string{"c"},
			// 	Usage:   "Create products from the give Shopify CSV",
			// 	Flags: cmd.Flags,
			// 	Action: createProducts,
			// },
		},
	}
}
