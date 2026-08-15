package inventory

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/ScreenStaring/shopify-dev-tools/cmd"
	"github.com/ScreenStaring/shopify-dev-tools/cmd/products"
)

var Cmd cli.Command

// toInventoryItemGID returns the id as an InventoryItem GID, prepending
// the gid://shopify/InventoryItem/ prefix when given a bare id.
func toInventoryItemGID(id string) string {
	if strings.HasPrefix(id, "gid://") {
		return id
	}

	return "gid://shopify/InventoryItem/" + id
}

func items(c *cli.Context) error {
	args := make([]string, 0, c.NArg())
	for i := 0; i < c.NArg(); i++ {
		args = append(args, c.Args().Get(i))
	}

	if len(args) == 0 {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("Cannot read from stdin: %s", err)
		}

		for _, line := range strings.Split(strings.TrimRight(string(data), "\n"), "\n") {
			if line != "" {
				args = append(args, line)
			}
		}
	}

	if len(args) == 0 {
		return fmt.Errorf("You must supply at least one inventory item id")
	}

	ids := make([]string, 0, len(args))
	argForGID := make(map[string]string, len(args))
	for _, arg := range args {
		gid := toInventoryItemGID(arg)
		ids = append(ids, gid)
		argForGID[gid] = arg
	}

	productList, missing, err := FetchProductsByInventoryItemIDs(cmd.NewGraphQLClient(c), ids)
	if err != nil {
		return err
	}

	if c.Bool("jsonl") {
		products.PrintJSONL(productList)
	} else {
		products.PrintFormatted(productList, nil)
	}

	if len(missing) > 0 {
		missingArgs := make([]string, 0, len(missing))
		for _, gid := range missing {
			missingArgs = append(missingArgs, argForGID[gid])
		}

		notice := "Inventory Item(s) not found: " + strings.Join(missingArgs, ", ")
		if c.Bool("jsonl") {
			fmt.Fprintln(os.Stderr, notice)
		} else {
			fmt.Println(notice)
		}
	}

	return nil
}

func init() {
	itemsFlags := []cli.Flag{
		&cli.BoolFlag{
			Name:    "jsonl",
			Aliases: []string{"j"},
			Usage:   "Output the products in JSONL format",
		},
	}

	Cmd = cli.Command{
		Name:    "inventory",
		Aliases: []string{"inv"},
		Usage:   "Do things with inventory",
		Subcommands: []*cli.Command{
			{
				Name:        "items",
				Aliases:     []string{"i"},
				Usage:       "Look up the variants and products for the given inventory item IDs",
				ArgsUsage:   "[ID [ID ...]]",
				Description: "If IDs are not given they're read from stdin one per line",
				Flags:       append(cmd.Flags, itemsFlags...),
				Action:      items,
			},
		},
	}
}
