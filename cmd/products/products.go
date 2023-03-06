package products

import (
	"encoding/json"

	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
	shopify "github.com/bold-commerce/go-shopify/v3"
	"github.com/cheynewallace/tabby"

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
	Limit int64 `url:"limit"`
	Status string `url:"status"`
	UpdatedAtMin time.Time `url:"updated_at_min"`
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

func printFormatted(products []shopify.Product)  {
	t := tabby.New()

	for _, product := range products {
		s := reflect.ValueOf(&product).Elem()

		for i := 0; i < s.NumField(); i++ {
			t.AddLine(s.Type().Field(i).Name, s.Field(i).Interface())
		}

		t.Print()
		fmt.Printf("%s\n", strings.Repeat("-", 20))
	}
}

func listProducts(c *cli.Context) error {
	options := listProductOptions{Limit: 10}

	if len(c.String("status")) > 0 {
		options.Status = c.String("status")
	}

	if c.Int64("limit") > 0 {
		options.Limit = c.Int64("limit")
	}

	products, err := cmd.NewShopifyClient(c).Product.List(options)
	if err != nil {
		return fmt.Errorf("Cannot list products: %s", err)
	}

	if c.Bool("jsonl") {
		printJSONL(products)
	} else {
		printFormatted(products)
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
		&cli.Int64Flag{
			Name: "limit",
			Aliases: []string{"l"},
		},
		&cli.StringFlag{
			Name: "status",
			Aliases: []string{"s"},
		},
		&cli.BoolFlag{
			Name:    "jsonl",
			Aliases: []string{"j"},
			Usage:   "Output the products in JSONL format",
		},
	}

	Cmd = cli.Command{
		Name:  "products",
		Aliases: []string{"p"},
		Usage:   "Do things with products",
		Subcommands: []*cli.Command{
			{
				Name: "ls",
				Aliases: []string{"l"},
				Usage:   "List some of a shop's products)",
				Flags: append(cmd.Flags, productFlags...),
				Action: listProducts,
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
