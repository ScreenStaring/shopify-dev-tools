package metafields

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/cheynewallace/tabby"
	"github.com/urfave/cli/v2"

	"github.com/ScreenStaring/shopify-dev-tools/cmd"
	"github.com/ScreenStaring/shopify-dev-tools/gql/storefront"
)

type metafieldOptions struct {
	Namespace string `url:"namespace"`
	Key       string `url:"key"`
	JSONL     bool
}

var Cmd cli.Command

func contextToOptions(c *cli.Context) metafieldOptions {
	return metafieldOptions{
		Key:       c.String("key"),
		Namespace: c.String("namespace"),
		JSONL:     c.Bool("jsonl"),
	}
}

func printMetafields(metafields []Metafield, options metafieldOptions) {
	if options.JSONL {
		printJSONL(metafields)
	} else {
		printFormatted(metafields)
	}
}

func printJSONL(metafields []Metafield) {
	for _, metafield := range metafields {
		line, err := json.Marshal(metafield)
		if err != nil {
			panic(err)
		}

		fmt.Println(string(line))
	}
}

func printFormatted(metafields []Metafield) {
	items := make([]cmd.MetafieldPrintable, len(metafields))
	for i, mf := range metafields {
		items[i] = cmd.MetafieldPrintable{
			Gid:         mf.ID,
			Namespace:   mf.Namespace,
			Key:         mf.Key,
			Description: mf.Description,
			Value:       mf.Value,
			Type:        mf.Type,
			CreatedAt:   mf.CreatedAt,
			UpdatedAt:   mf.UpdatedAt,
		}
	}

	cmd.PrintMetafields(items)
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
	client := cmd.NewGraphQLClient(c)
	metafields, err := listCustomerMetafields(client, id, options.Namespace, options.Key, c.Bool("reverse"))
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

	ids, skus, err := cmd.ParseIDArgs(c, "an ID")
	if err != nil {
		return err
	}

	options := contextToOptions(c)
	client := cmd.NewGraphQLClient(c)

	var metafields []Metafield
	var failures []string
	for _, id := range ids {
		mfs, err := listProductMetafields(client, id, options.Namespace, options.Key, c.Bool("reverse"))
		if err != nil {
			failures = append(failures, fmt.Sprintf("%d: %s", id, err))
			continue
		}
		metafields = append(metafields, mfs...)
	}

	if len(skus) > 0 {
		mfs, foundSkus, err := listProductMetafieldsBySku(client, skus, options.Namespace, options.Key, c.Bool("reverse"))
		if err != nil {
			failures = append(failures, err.Error())
		} else {
			metafields = append(metafields, mfs...)
			for _, sku := range missingSkus(skus, foundSkus) {
				failures = append(failures, fmt.Sprintf("sku:%s: not found", sku))
			}
		}
	}

	printMetafields(metafields, options)

	if len(failures) > 0 {
		return fmt.Errorf("Cannot retrieve metafield(s): %s", strings.Join(failures, ", "))
	}

	return nil
}

func shopAction(c *cli.Context) error {
	options := contextToOptions(c)
	client := cmd.NewGraphQLClient(c)
	metafields, err := listShopMetafields(client, options.Namespace, options.Key, c.Bool("reverse"))
	if err != nil {
		return fmt.Errorf("Cannot list metafields for shop: %s", err)
	}

	printMetafields(metafields, options)

	return nil
}

func appAction(c *cli.Context) error {
	client := cmd.NewGraphQLClient(c)

	metafields, err := listAppInstallationMetafields(client, c.String("namespace"))
	if err != nil {
		return err
	}

	items := make([]cmd.MetafieldPrintable, len(metafields))
	for i, mf := range metafields {
		items[i] = cmd.MetafieldPrintable{
			Gid:         mf.ID,
			Namespace:   mf.Namespace,
			Key:         mf.Key,
			Description: mf.Description,
			Value:       mf.Value,
			Type:        mf.Type,
			CreatedAt:   mf.CreatedAt,
			UpdatedAt:   mf.UpdatedAt,
		}
	}

	cmd.PrintMetafields(items)

	return nil
}

func variantAction(c *cli.Context) error {
	if c.NArg() == 0 {
		return errors.New("Variant id required")
	}

	ids, skus, err := cmd.ParseIDArgs(c, "an ID")
	if err != nil {
		return err
	}

	options := contextToOptions(c)
	client := cmd.NewGraphQLClient(c)

	var metafields []Metafield
	var failures []string
	for _, id := range ids {
		mfs, err := listVariantMetafields(client, id, options.Namespace, options.Key, c.Bool("reverse"))
		if err != nil {
			failures = append(failures, fmt.Sprintf("%d: %s", id, err))
			continue
		}
		metafields = append(metafields, mfs...)
	}

	if len(skus) > 0 {
		mfs, foundSkus, err := listVariantMetafieldsBySku(client, skus, options.Namespace, options.Key, c.Bool("reverse"))
		if err != nil {
			failures = append(failures, err.Error())
		} else {
			metafields = append(metafields, mfs...)
			for _, sku := range missingSkus(skus, foundSkus) {
				failures = append(failures, fmt.Sprintf("sku:%s: not found", sku))
			}
		}
	}

	printMetafields(metafields, options)

	if len(failures) > 0 {
		return fmt.Errorf("Cannot retrieve metafield(s): %s", strings.Join(failures, ", "))
	}

	return nil
}

// missingSkus returns the requested SKUs that were not found.
func missingSkus(requested, found []string) []string {
	foundSet := make(map[string]bool, len(found))
	for _, s := range found {
		foundSet[s] = true
	}

	var missing []string
	for _, s := range requested {
		if !foundSet[s] {
			missing = append(missing, s)
		}
	}

	return missing
}

func storefrontListAction(c *cli.Context) error {
	shop := c.String("shop")
	token := cmd.LookupAccessToken(shop, c.String("access-token"))

	metafields, err := storefront.New(shop, token).List()
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

func storefrontEnableAction(c *cli.Context) error {
	shop := c.String("shop")
	token := cmd.LookupAccessToken(shop, c.String("access-token"))

	if c.Args().Len() < 2 {
		return fmt.Errorf("You must supply a key and owner")
	}

	id, err := storefront.New(shop, token).Enable(c.Args().Get(0), c.Args().Get(1))
	if err != nil {
		return err
	}

	fmt.Printf("Created %s \n", id)

	return nil
}

func parseMetafieldArg(arg string) (metafieldInput, error) {
	parts := strings.SplitN(arg, "@", 2)
	if len(parts) != 2 {
		return metafieldInput{}, fmt.Errorf("invalid metafield argument %q: must be in GID@namespace.key format", arg)
	}

	nk := strings.SplitN(parts[1], ".", 2)
	if len(nk) != 2 {
		return metafieldInput{}, fmt.Errorf("invalid metafield argument %q: namespace.key portion must contain a dot", arg)
	}

	return metafieldInput{OwnerID: parts[0], Namespace: nk[0], Key: nk[1]}, nil
}

func definitionsAction(c *cli.Context) error {
	if c.NArg() == 0 {
		return errors.New("Resource name required")
	}

	ownerType := strings.ToUpper(c.Args().Get(0))
	client := cmd.NewGraphQLClient(c)

	definitions, err := listMetafieldDefinitions(client, ownerType, c.String("namespace"))
	if err != nil {
		return err
	}

	t := tabby.New()
	for _, def := range definitions {
		t.AddLine("Gid", def.ID)
		t.AddLine("Name", def.Name)
		t.AddLine("Namespace", def.Namespace)
		t.AddLine("Key", def.Key)
		t.AddLine("Description", def.Description)
		t.AddLine("Type", def.Type)
		t.AddLine("Owner Type", def.OwnerType)
		t.Print()
		fmt.Printf("%s\n", strings.Repeat("-", 20))
	}

	return nil
}

func printDeleteResults(deleted []DeletedMetafield) {
	t := tabby.New()
	for _, mf := range deleted {
		t.AddLine("Gid", mf.OwnerID)
		t.AddLine("Key", mf.Key)
		t.AddLine("Namespace", mf.Namespace)
		if mf.Error != "" {
			t.AddLine("Result", mf.Error)
		} else {
			t.AddLine("Result", "Deleted")
		}
		t.Print()
		fmt.Printf("%s\n", strings.Repeat("-", 20))
	}
}

func deleteAction(c *cli.Context) error {
	shop := c.String("shop")
	token := cmd.LookupAccessToken(shop, c.String("access-token"))

	if c.NArg() > 0 {
		var inputs []metafieldInput
		for _, arg := range c.Args().Slice() {
			mf, err := parseMetafieldArg(arg)
			if err != nil {
				return err
			}
			inputs = append(inputs, mf)
		}

		deleted, err := deleteMetafields(shop, token, inputs)
		if err != nil {
			return err
		}

		printDeleteResults(deleted)
		return nil
	}

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		mf, err := parseMetafieldArg(line)
		if err != nil {
			return err
		}

		deleted, err := deleteMetafields(shop, token, []metafieldInput{mf})
		if err != nil {
			return err
		}

		printDeleteResults(deleted)
	}

	return scanner.Err()
}

func init() {
	storefrontFlags := []cli.Flag{
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

	apiVersionFlag := &cli.StringFlag{
		Name:    "api-version",
		Aliases: []string{"a"},
		Usage:   "API version to use; default is a versionless call",
	}

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
		&cli.BoolFlag{
			Name:    "jsonl",
			Aliases: []string{"j"},
			Usage:   "Output the metafields in JSONL format",
		},
		&cli.BoolFlag{
			Name:    "reverse",
			Aliases: []string{"r"},
			Usage:   "Reverse the order of the results",
		},
	}

	Cmd = cli.Command{
		Name:    "metafield",
		Aliases: []string{"m", "meta"},
		Usage:   "Metafield utilities",
		Subcommands: []*cli.Command{
			{
				Name:    "definitions",
				Aliases: []string{"def"},
				Usage:   "Metafield definition utilities",
				Subcommands: []*cli.Command{
					{
						Name:      "ls",
						ArgsUsage: "resource",
						Flags: append(cmd.Flags, apiVersionFlag, &cli.StringFlag{
							Name:    "namespace",
							Aliases: []string{"n"},
							Usage:   "Filter by namespace",
						}),
						Action: definitionsAction,
						Usage:  "List metafield definitions for the given resource",
					},
				},
			},
			{
				Name:        "delete",
				Aliases:     []string{"d"},
				ArgsUsage:   "GID@namespace.key [GID@namespace.key ...]",
				Description: "If IDs are not given they're read from stdin one per line",
				Flags:       cmd.Flags,
				Action:      deleteAction,
				Usage:       "Delete one or more metafields",
			},
			{
				Name: "app",
				Flags: append(cmd.Flags, apiVersionFlag, &cli.StringFlag{
					Name:    "namespace",
					Aliases: []string{"n"},
					Usage:   "Filter by namespace",
				}),
				Action: appAction,
				Usage:  "List metafields for the app installation associated with the credentials",
			},
			{
				Name:    "customer",
				Flags:   append(append(cmd.Flags, metafieldFlags...), apiVersionFlag),
				Aliases: []string{"c"},
				Action:  customerAction,
				Usage:   "List metafields for the given customer",
			},
			{
				Name:      "product",
				Flags:     append(append(cmd.Flags, metafieldFlags...), apiVersionFlag),
				Aliases:   []string{"products", "prod", "p"},
				Action:    productAction,
				Usage:     "List metafields for the given product(s)",
				ArgsUsage: "[ID|sku:VALUE [ID|sku:VALUE ...]]",
			},
			{
				Name:    "shop",
				Flags:   append(append(cmd.Flags, metafieldFlags...), apiVersionFlag),
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
						Flags:  append(cmd.Flags, storefrontFlags...),
						Usage:  "List accessible metafields",
						Action: storefrontListAction,
					},
					{
						Name:      "enable",
						Aliases:   []string{"e"},
						Usage:     "Make a metafield accessible",
						ArgsUsage: "NAMESPACE.KEY OWNER",
						Flags:     cmd.Flags,
						Action:    storefrontEnableAction,
					},
				},
			},
			{
				Name:      "variant",
				Aliases:   []string{"var", "v"},
				Flags:     append(append(cmd.Flags, metafieldFlags...), apiVersionFlag),
				Action:    variantAction,
				Usage:     "List metafields for the given variant(s)",
				ArgsUsage: "[ID|sku:VALUE [ID|sku:VALUE ...]]",
			},
		},
	}

}
