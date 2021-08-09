package shop

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/urfave/cli/v2"
	"github.com/ScreenStaring/shopify-dev-tools/cmd"
	"github.com/cheynewallace/tabby"
)

var Cmd cli.Command

// Some low-budget formatting
func formatField(field string) string {
	re := regexp.MustCompile("([a-z])([A-Z])")
	name := strings.Replace(field, "API", "Api", 1)
	name = re.ReplaceAllString(name, "$1 $2")
	name = strings.Replace(name, " At", "", 1)

	return name
}

func accessAction(c *cli.Context) error {
	// not supported, need to update API client
	return nil
}

func infoAction(c *cli.Context) error {
	shop, err := cmd.NewShopifyClient(c).Shop.Get(nil)
	if err != nil {
		return fmt.Errorf("Cannot get info for shop: %s", err)
	}

	t := tabby.New()
	s := reflect.ValueOf(shop).Elem()

	for i := 0; i < s.NumField(); i++ {
		t.AddLine(formatField(s.Type().Field(i).Name), s.Field(i).Interface())
	}

	t.Print()

	return nil
}

func init() {
	Cmd = cli.Command{
		Name:  "shop",
		Aliases: []string{"s"},
		Usage:   "Information about the given shop",
		Subcommands: []*cli.Command{
			// {
			// 	Name: "access",
			// 	Aliases: []string{"a"},
			// 	Usage:   "Permissions granted to the given token/key",
			// 	Flags: cmd.Flags,
			// 	Action: accessAction,
			// },
			{
				Name: "info",
				Aliases: []string{"i"},
				Usage:   "Information about the shop",
				Flags: cmd.Flags,
				Action: infoAction,
			},
		},
	}
}
