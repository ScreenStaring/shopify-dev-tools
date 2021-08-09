package admin

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/pkg/browser"
	"github.com/urfave/cli/v2"

	"github.com/ScreenStaring/shopify-dev-tools/cmd"
)

var Cmd cli.Command

type themeOptions struct {
	Fields []string `url:"fields[]"`
}

func findPublishedTheme(c *cli.Context) (int64, error) {
	shopify := cmd.NewShopifyClient(c)
	options := themeOptions{[]string{"id", "role"}}

	themes, err := shopify.Theme.List(options)
	if err != nil {
		return 0, err
	}

	var id int64
	for _, theme := range(themes) {
		if theme.Role == "main" {
			id = theme.ID
			break
		}
	}

	return id, nil
}

func orderAction(c *cli.Context) error {
	var qs map[string]string

	admin := NewAdminURL(c.String("shop"))

	if c.NArg() == 0 {
		browser.OpenURL(admin.Orders(qs))
		return nil
	}

	id, err := strconv.ParseInt(c.Args().Get(0), 10, 64)
	if err != nil {
		return fmt.Errorf("Order id '%s' invalid: must be an int", c.Args().Get(0))
	}

	browser.OpenURL(admin.Order(id, qs))
	return nil
}


func productAction(c *cli.Context) error {
	var qs map[string]string

	admin := NewAdminURL(c.String("shop"))

	if c.NArg() == 0 {
		browser.OpenURL(admin.Products(qs))
		return nil
	}

	id, err := strconv.ParseInt(c.Args().Get(0), 10, 64)
	if err != nil {
		return fmt.Errorf("Product id '%s' invalid: must be an int", c.Args().Get(0))
	}

	browser.OpenURL(admin.Product(id, qs))
	return nil
}


func themeAction(c *cli.Context) error {
	var id int64
	var err error

	var qs map[string]string

	admin := NewAdminURL(c.String("shop"))

	if c.NArg() == 0 {
		id, err = findPublishedTheme(c)
		if err != nil {
			return fmt.Errorf("Error finding published theme: %s", err)
		}

		if id == 0 {
			return errors.New("No published theme")
		}
	} else {
		id, err = strconv.ParseInt(c.Args().Get(0), 10, 64)
		if err != nil {
			return fmt.Errorf("Theme id '%s' invalid: must be an int", c.Args().Get(0))
		}
	}

	browser.OpenURL(admin.Theme(id, qs))
	return nil
}

func themesAction(c *cli.Context) error {
	var qs map[string]string

	admin := NewAdminURL(c.String("shop"))
	browser.OpenURL(admin.Themes(qs))
	return nil
}

func init() {
	Cmd = cli.Command{
		Name:  "admin",
		Aliases: []string{"a"},
		Usage:   "Open admin pages",
		Subcommands: []*cli.Command{
			{
				Name: "order",
				Aliases: []string{"orders", "o"},
				Usage:   "Open the given order ID for editing; if no ID given open the orders page",
				Flags: cmd.Flags,
				Action: orderAction,
			},
			{
				Name: "product",
				Aliases: []string{"products", "prod", "p"},
				Usage:   "Open the given product ID for editing; if no ID given open the products page",
				Flags: cmd.Flags,
				Action: productAction,
			},
			{
				Name: "theme",
				Usage:   "Open the currently published theme or given theme ID for editing",
				Aliases: []string{"t"},
				Flags: cmd.Flags,
				Action: themeAction,
			},
			{
				Name: "themes",
				Usage:  "Open themes section of the admin (not for editing)",
				Flags: cmd.Flags,
				Action: themesAction,
			},
		},
	}
}
