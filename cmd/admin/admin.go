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

// open prints url to stdout when --print is given; otherwise it opens url
// in the default browser.
func open(c *cli.Context, url string) error {
	if c.Bool("print") {
		fmt.Println(url)
		return nil
	}

	return browser.OpenURL(url)
}

func findPublishedTheme(c *cli.Context) (int64, error) {
	themes, err := listThemes(cmd.NewGraphQLClient(c))
	if err != nil {
		return 0, err
	}

	var id int64
	for _, theme := range themes {
		if theme.Role == "MAIN" {
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
		return open(c, admin.Orders(qs))
	}

	id, err := strconv.ParseInt(c.Args().Get(0), 10, 64)
	if err != nil {
		return fmt.Errorf("Order id '%s' invalid: must be an int", c.Args().Get(0))
	}

	return open(c, admin.Order(id, qs))
}

func productAction(c *cli.Context) error {
	var qs map[string]string

	admin := NewAdminURL(c.String("shop"))

	if c.NArg() == 0 {
		return open(c, admin.Products(qs))
	}

	id, err := strconv.ParseInt(c.Args().Get(0), 10, 64)
	if err != nil {
		return fmt.Errorf("Product id '%s' invalid: must be an int", c.Args().Get(0))
	}

	return open(c, admin.Product(id, qs))
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

	return open(c, admin.Theme(id, qs))
}

func themesAction(c *cli.Context) error {
	var qs map[string]string

	admin := NewAdminURL(c.String("shop"))
	return open(c, admin.Themes(qs))
}

func settingsAction(c *cli.Context) error {
	var qs map[string]string

	admin := NewAdminURL(c.String("shop"))
	return open(c, admin.SettingsGeneral(qs))
}

func settingsAppAction(c *cli.Context) error {
	var qs map[string]string

	admin := NewAdminURL(c.String("shop"))
	return open(c, admin.SettingsApps(qs))
}

func settingsNotificationAction(c *cli.Context) error {
	var qs map[string]string

	admin := NewAdminURL(c.String("shop"))
	return open(c, admin.SettingsNotifications(qs))
}

func settingsUsersAction(c *cli.Context) error {
	var qs map[string]string

	admin := NewAdminURL(c.String("shop"))
	return open(c, admin.SettingsUsers(qs))
}

func init() {
	flags := append(append([]cli.Flag{}, cmd.Flags...), &cli.BoolFlag{
		Name:    "print",
		Aliases: []string{"p"},
		Usage:   "Print the URL instead of opening it",
	})

	Cmd = cli.Command{
		Name:    "admin",
		Aliases: []string{"a"},
		Usage:   "Open admin pages",
		Subcommands: []*cli.Command{
			{
				Name:    "order",
				Aliases: []string{"orders", "o"},
				Usage:   "Open the given order ID for editing; if no ID given open the orders page",
				Flags:   flags,
				Action:  orderAction,
			},
			{
				Name:    "product",
				Aliases: []string{"products", "prod", "p"},
				Usage:   "Open the given product ID for editing; if no ID given open the products page",
				Flags:   flags,
				Action:  productAction,
			},
			{
				Name:    "theme",
				Usage:   "Open the currently published theme or given theme ID for editing",
				Aliases: []string{"t"},
				Flags:   flags,
				Action:  themeAction,
			},
			{
				Name:   "themes",
				Usage:  "Open themes section of the admin (not for editing)",
				Flags:  flags,
				Action: themesAction,
			},
			{
				Name:    "settings",
				Aliases: []string{"s"},
				Usage:   "Open the general settings page or settings sections",
				Flags:   flags,
				Action:  settingsAction,
				Subcommands: []*cli.Command{
					{
						Name:    "app",
						Aliases: []string{"apps"},
						Usage:   "Open the apps settings page",
						Flags:   flags,
						Action:  settingsAppAction,
					},
					{
						Name:    "notification",
						Aliases: []string{"notifications", "notices"},
						Usage:   "Open the notifications settings page",
						Flags:   flags,
						Action:  settingsNotificationAction,
					},
					{
						Name:    "users",
						Aliases: []string{"u"},
						Usage:   "Open the users settings page",
						Flags:   flags,
						Action:  settingsUsersAction,
					},
				},
			},
		},
	}
}
