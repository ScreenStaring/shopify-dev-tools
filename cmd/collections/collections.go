package collections

import (
	"fmt"
	"strings"

	"github.com/ScreenStaring/shopify-dev-tools/cmd"
	"github.com/ScreenStaring/shopify-dev-tools/cmd/collections/gql"
	"github.com/cheynewallace/tabby"
	"github.com/urfave/cli/v2"
)

var Cmd cli.Command

func listAction(c *cli.Context) error {
	shop := c.String("shop")
	token := cmd.LookupAccessToken(shop, c.String("access-token"))

	if c.Args().Len() > 0 {
		collection, err := gql.GetCollection(shop, token, c.Args().Get(0))
		if err != nil {
			return err
		}

		printCollections([]gql.Collection{*collection})
		return nil
	}

	custom := c.Bool("custom")
	smart := c.Bool("smart")
	if custom && smart {
		return fmt.Errorf("--custom and --smart cannot be used together")
	}

	collections, err := gql.ListCollections(shop, token, c.Int("limit"), c.Int("page"), c.String("title"), custom, smart)
	if err != nil {
		return err
	}

	if len(collections) == 0 {
		fmt.Println("No collections")
		return nil
	}

	printCollections(collections)
	return nil
}

func printCollections(collections []gql.Collection) {
	t := tabby.New()
	for _, c := range collections {
		t.AddLine("ID", strings.TrimPrefix(c.ID, "gid://shopify/Collection/"))
		t.AddLine("Title", c.Title)
		t.AddLine("Handle", c.Handle)
		t.AddLine("Type", c.Type)
		t.AddLine("Products", c.ProductsCount)
		t.AddLine("Updated", c.UpdatedAt)
		t.Print()

		cmd.PrintSeparator()
	}
}

func init() {
	listFlags := []cli.Flag{
		&cli.IntFlag{
			Name:    "limit",
			Aliases: []string{"l"},
			Usage:   "Maximum number of collections to return, must be <= 250",
			Value:   10,
		},
		&cli.StringFlag{
			Name:    "title",
			Aliases: []string{"t"},
			Usage:   "Only show collections with the given title, wildcards (*) allowed",
		},
		&cli.IntFlag{
			Name:    "page",
			Aliases: []string{"p"},
			Usage:   "Page of results to return, pages are limit sized",
			Value:   1,
		},
		&cli.BoolFlag{
			Name:    "custom",
			Aliases: []string{"c"},
			Usage:   "Only show custom (manual) collections",
		},
		&cli.BoolFlag{
			Name:    "smart",
			Aliases: []string{"s"},
			Usage:   "Only show smart (automated) collections",
		},
	}

	Cmd = cli.Command{
		Name:    "collections",
		Aliases: []string{"col"},
		Usage:   "Do things with collections",

		Subcommands: []*cli.Command{
			{
				Name:      "ls",
				Aliases:   []string{"l"},
				ArgsUsage: "[ID]",
				Usage:     "List the shop's collections or a collection given by ID",
				Flags:     append(cmd.Flags, listFlags...),
				Action:    listAction,
			},
		},
	}
}
