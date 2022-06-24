package metafields

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	shopify "github.com/bold-commerce/go-shopify/v3"
	"github.com/cheynewallace/tabby"
	"github.com/urfave/cli/v2"

	"github.com/ScreenStaring/shopify-dev-tools/cmd"
	"github.com/ScreenStaring/shopify-dev-tools/gql/storefront"
)

type metafieldOptions struct {
	Namespace string `url:"namespace"`
	Key       string `url:"key"`
	JSONL     bool
	OrderBy   []string
}

var Cmd cli.Command

var sortByFieldFuncs = map[string]lessFunc{
	"namespace":      byNamespaceAsc,
	"namespace:asc":  byNamespaceAsc,
	"namespace:desc": byNamespaceDesc,
	"key":            byKeyAsc,
	"key:asc":        byKeyAsc,
	"key:desc":       byKeyDesc,
	"create":         byCreatedAtAsc,
	"create:asc":     byCreatedAtAsc,
	"create:desc":    byCreatedAtDesc,
	"created":        byCreatedAtAsc,
	"created:asc":    byCreatedAtAsc,
	"created:desc":   byCreatedAtDesc,
	"update":         byUpdatedAtAsc,
	"update:asc":     byUpdatedAtAsc,
	"update:desc":    byUpdatedAtDesc,
	"updated":        byUpdatedAtAsc,
	"updated:asc":    byUpdatedAtAsc,
	"updated:desc":   byUpdatedAtDesc,
}

func contextToOptions(c *cli.Context) metafieldOptions {
	return metafieldOptions{
		Key:       c.String("key"),
		Namespace: c.String("namespace"),
		OrderBy:   c.StringSlice("order"),
		JSONL:     c.Bool("jsonl"),
	}
}

func printMetafields(metafields []shopify.Metafield, options metafieldOptions) {
	if options.JSONL {
		printJSONL(metafields)
	} else {
		printFormatted(metafields, options)
	}
}

func printJSONL(metafields []shopify.Metafield) {
	for _, metafield := range metafields {
		line, err := json.Marshal(metafield)
		if err != nil {
			panic(err)
		}

		fmt.Println(string(line))
	}

}

func printFormatted(metafields []shopify.Metafield, options metafieldOptions) {
	sortMetafields(metafields, options)

	t := tabby.New()
	for _, metafield := range metafields {
		t.AddLine("Id", metafield.ID)
		t.AddLine("Gid", metafield.AdminGraphqlAPIID)
		t.AddLine("Namespace", metafield.Namespace)
		t.AddLine("Key", metafield.Key)
		t.AddLine("Description", metafield.Description)
		// format JSON strings
		// also check for string types that look like json: /\A\{"[^"]+":/ or /\A[/ and /\]\Z/
		t.AddLine("Value", metafield.Value)
		t.AddLine("Type", metafield.ValueType)
		t.AddLine("Created", metafield.CreatedAt)
		t.AddLine("Updated", metafield.UpdatedAt)
		t.Print()
		fmt.Printf("%s\n", strings.Repeat("-", 20))
	}
}

// Cannot sort storefront metafields from GQL
func sortMetafields(metafields []shopify.Metafield, options metafieldOptions) {
	var funcs []lessFunc

	if len(options.OrderBy) != 0 {
		for _, field := range options.OrderBy {
			funcs = append(funcs, sortByFieldFuncs[field])
		}
	} else {
		if options.Namespace != "" {
			funcs = []lessFunc{byKeyAsc}
		} else if options.Key != "" {
			funcs = []lessFunc{byNamespaceAsc}
		} else {
			funcs = []lessFunc{byNamespaceAsc, byKeyAsc}
		}
	}

	sorter := metafieldsSorter{less: funcs}
	sorter.Sort(metafields)
}

func customerAction(c *cli.Context) error {
	if c.NArg() == 0 {
		return errors.New("Customer id required")
	}

	id, err := strconv.ParseInt(c.Args().Get(0), 10, 64)
	if err != nil {
		return fmt.Errorf("Customer id '%s' invalid: must be an int", c.Args().Get(0))
	}

	options := contextToOptions(c)
	metafields, err := cmd.NewShopifyClient(c).Customer.ListMetafields(id, options)
	if err != nil {
		return fmt.Errorf("Cannot list metafields for customer: %s", err)
	}

	printMetafields(metafields, options)
	return nil
}

func productAction(c *cli.Context) error {
	if c.NArg() == 0 {
		return errors.New("Product id required")
	}

	// TODO: accept handle too (maybe use regex to detect? But handle can be all digits too)
	id, err := strconv.ParseInt(c.Args().Get(0), 10, 64)
	if err != nil {
		return fmt.Errorf("Product id '%s' invalid: must be an int", c.Args().Get(0))
	}

	options := contextToOptions(c)
	metafields, err := cmd.NewShopifyClient(c).Product.ListMetafields(id, options)
	if err != nil {
		return fmt.Errorf("Cannot list metafields for product %d: %s", id, err)
	}

	printMetafields(metafields, options)
	return nil
}

func shopAction(c *cli.Context) error {
	options := contextToOptions(c)
	metafields, err := cmd.NewShopifyClient(c).Metafield.List(options)
	if err != nil {
		return fmt.Errorf("Cannot list metafields for shop: %s", err)
	}

	printFormatted(metafields, options)

	return nil
}

func storefrontAction(c *cli.Context) error {
	metafields, err := storefront.New(c.String("shop"), c.String("access-token")).List()
	if err != nil {
		return err
	}

	//fmt.Printf("%+v\n", metafields)

	t := tabby.New()
	for _, metafield := range metafields {
		t.AddLine("Id", metafield["legacyResourceId"])
		t.AddLine("Gid", metafield["id"])
		t.AddLine("Namespace", metafield["namespace"])
		t.AddLine("Key", metafield["key"])
		t.AddLine("Owner Type", metafield["ownerType"])
		t.AddLine("Created", metafield["createdAt"])
		t.AddLine("Updated", metafield["updatedAt"])
		t.Print()
		fmt.Printf("%s\n", strings.Repeat("-", 20))
	}

	return nil
}

func variantAction(c *cli.Context) error {
	if c.NArg() == 0 {
		return errors.New("Variant id required")
	}

	id, err := strconv.ParseInt(c.Args().Get(0), 10, 64)
	if err != nil {
		return fmt.Errorf("Variant id '%s' invalid: must be an int", c.Args().Get(0))
	}

	options := contextToOptions(c)
	metafields, err := cmd.NewShopifyClient(c).Variant.ListMetafields(id, options)
	if err != nil {
		return fmt.Errorf("Cannot list metafields for variant %d: %s", id, err)
	}

	printMetafields(metafields, options)

	return nil
}

func init() {
	metafieldFlags := []cli.Flag{
		&cli.StringFlag{
			Name:    "key",
			Aliases: []string{"k"},
			Usage:   "Find metafields with the given key",
		},
		&cli.StringFlag{
			Name:    "namespace",
			Aliases: []string{"n"},
			Usage:   "Find metafields with the given namespace",
		},
		&cli.StringSliceFlag{
			Name:    "order",
			Aliases: []string{"o"},
			Usage:   "Order metafields by the given properties",
		},
		&cli.BoolFlag{
			Name:    "jsonl",
			Aliases: []string{"j"},
			Usage:   "Output the metafields in JSONL format",
		},
	}

	// create!
	Cmd = cli.Command{
		Name:    "metafield",
		Aliases: []string{"m", "meta"},
		Usage:   "Metafield utilities",
		Subcommands: []*cli.Command{
			{
				Name:    "customer",
				Flags:   append(cmd.Flags, metafieldFlags...),
				Aliases: []string{"c"},
				Action:  customerAction,
				Usage:   "List metafields for the given customer",
			},
			{
				Name:    "product",
				Flags:   append(cmd.Flags, metafieldFlags...),
				Aliases: []string{"products", "prod", "p"},
				Action:  productAction,
				Usage:   "List metafields for the given product",
			},
			{
				Name:    "shop",
				Flags:   append(cmd.Flags, metafieldFlags...),
				Aliases: []string{"s"},
				Action:  shopAction,
				Usage:   "List metafields for the given shop",
			},
			{
				Name:    "storefront",
				Aliases: []string{"sf"},
				Usage:   "Storefront API utilities",
				Subcommands: []*cli.Command{
					{
						Name:   "ls",
						Flags:  append(cmd.Flags, metafieldFlags...),
						Action: storefrontAction,
					},
					// {
					// 	// --key, --namespace --owner
					// 	Name: "create",
					// 	Flags: append(cmd.Flags, metafieldFlags...),
					// 	Action: storefrontAction,
					// },
				},
			},
			{
				Name:    "variant",
				Aliases: []string{"var", "v"},
				Flags:   append(cmd.Flags, metafieldFlags...),
				Action:  variantAction,
				Usage:   "List metafields for the given variant",
			},
		},
	}

}
