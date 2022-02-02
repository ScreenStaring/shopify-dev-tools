package orders

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/urfave/cli/v2"
	"github.com/ScreenStaring/shopify-dev-tools/cmd"
	"github.com/cheynewallace/tabby"
)

var Cmd cli.Command

func userAgentAction(c *cli.Context) error {
	if(c.Args().Len() == 0) {
		return fmt.Errorf("You must supply an order id")
	}

	id, err := strconv.ParseInt(c.Args().Get(0), 10, 64)
	if err != nil {
		return fmt.Errorf("Order id '%s' is invalid: must be an int", c.Args().Get(0))
	}

	order, err := cmd.NewShopifyClient(c).Order.Get(id, nil)
	if err != nil {
		return fmt.Errorf("Cannot find order: %s", err)
	}


	t := tabby.New()
	t.AddLine("Id", order.ID)
	t.AddLine("User Agent", order.ClientDetails.UserAgent)
	t.AddLine("Display", fmt.Sprintf("%dx%d", order.ClientDetails.BrowserWidth, order.ClientDetails.BrowserHeight))
	t.AddLine("Accept Language", order.ClientDetails.AcceptLanguage)
	t.AddLine("IP", order.BrowserIp)
	t.AddLine("Session", order.ClientDetails.SessionHash)
	t.Print()

	// TODO: Make a function for this. We use it several places.
	fmt.Printf("%s\n", strings.Repeat("-", 20))

	return nil
}

func init() {
	Cmd = cli.Command{
		Name:  "orders",
		Aliases: []string{"o"},
		Usage:   "Information about orders",
		Subcommands: []*cli.Command{
			{
				Name: "useragent",
				Aliases: []string{"ua"},
				Usage:   "Info about the web browser used to place the order",
				Flags: cmd.Flags,
				Action: userAgentAction,
			},
		},
	}
}
