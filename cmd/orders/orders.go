package orders

import (
	"fmt"

	"github.com/urfave/cli/v2"
	"github.com/ScreenStaring/shopify-dev-tools/cmd"
	"github.com/cheynewallace/tabby"
)

var Cmd cli.Command


func userAgentAction(c *cli.Context) error {
	if(c.Args().Len() == 0) {
		return fmt.Errorf("You must supply an order id")
	}

	id, err := cmd.ParseIntAt(c, 0)
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

	cmd.PrintSeparator()

	return nil
}

func listAction(c *cli.Context) error {
	var ids []int64

	for i := 0; i < c.NArg(); i++ {
		id, err := cmd.ParseIntAt(c, i)
		if err != nil {
			return fmt.Errorf("Order id '%s' invalid: must be an int", c.Args().Get(i))
		}
		ids = append(ids, id)
	}

	status := "open"
	if len(c.String("status")) > 0 {
		status = c.String("status")
	}

	shop := c.String("shop")
	orders, err := listOrders(shop, cmd.LookupAccessToken(shop, c.String("access-token")), ids, status, c.Int("limit"))
	if err != nil {
		return err
	}

	printOrders(orders)

	return nil
}

func printOrders(orders []Order) {
	t := tabby.New()
	for _, order := range orders {
		t.AddLine("Id", order.ID)
		t.AddLine("Name", order.Name)
		t.AddLine("Created At", order.CreatedAt)
		t.AddLine("Updated At", order.UpdatedAt)
		t.AddLine("Canceled At", order.CancelledAt)
		t.AddLine("Closed At", order.ClosedAt)
		t.AddLine("Financial Status", order.DisplayFinancialStatus)
		t.AddLine("Fulfillment Status", order.DisplayFulfillmentStatus)


		note := order.Note
		if len(order.Note) > 0 {
			note = fmt.Sprintf("%q", order.Note)
		}

		t.AddLine("Note", note)
		t.Print()

		fmt.Println("Line Items")
		printLineItems(order.LineItems)
		fmt.Print("\n")

		cmd.PrintSeparator();

	}

}

func truncate(val string) string {
	max := 25

	if len(val) < max {
		return val
	}

	cut := val[0:max]

	if len(cut) < len(val) {
		cut += "…"
	}

	return cut
}

func printLineItems(lines []LineItem) {
	x := tabby.New()
	x.AddHeader("ID", "Product ID", "Variant ID", "SKU", "Title", "Quantity", "Status")

	for _, line := range lines {
		x.AddLine(
			line.ID,
			line.ProductID,
			line.VariantID,
			line.SKU,
			truncate(line.Name),
			line.Quantity,
			line.FulfillmentStatus,
		)
	}

	x.Print()
}

func init() {
	ordersFlags := []cli.Flag{
		&cli.StringFlag{
			Name:    "status",
			Aliases: []string{"s"},
			Usage:   "GraphQL Admin API orders status to filter, defaults to 'open'",
		},
		&cli.IntFlag{
			Name:    "limit",
			Aliases: []string{"l"},
			Usage:   "Maximum number of orders to return, must be <= 250",
			Value:   10,
		},
	}

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
			{
				Name: "ls",
				Usage:   "List the shop's orders or the orders given by the specified IDs",
				Flags: append(cmd.Flags, ordersFlags...),
				Action: listAction,
			},
		},
	}
}
