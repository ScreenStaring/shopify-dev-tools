package orders

import (
	"fmt"
	"time"

	shopify "github.com/bold-commerce/go-shopify/v3"
	"github.com/urfave/cli/v2"
	"github.com/ScreenStaring/shopify-dev-tools/cmd"
	"github.com/cheynewallace/tabby"
)

var Cmd cli.Command

// TODO: implement
type listOrdersOptions struct {
	Ids []int64 `url:"ids,comma,omitempty"`
	CreatedAtMin time.Time `url:"created_at_min,omitempty"`
	Status string `url:"status,omitempty"`
}

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
	options := listOrdersOptions{Status: "open"}

	if c.NArg() > 0 {
		for i := 0; i < c.NArg(); i++ {
			id, err := cmd.ParseIntAt(c, i)
			if err != nil {
				return fmt.Errorf("Order id '%s' invalid: must be an int", c.Args().Get(i))
			}

			options.Ids = append(options.Ids, id)
		}

	}

	if len(c.String("status")) > 0 {
		options.Status = c.String("status")
	}

	orders, err := cmd.NewShopifyClient(c).Order.List(options)
	if err != nil {
		return fmt.Errorf("Cannot list orders: %s", err)
	}


	printOrders(orders)

	return nil
}

func printOrders(orders []shopify.Order) {
	t := tabby.New()
	for _, order := range orders {
		t.AddLine("Id", order.ID)
		t.AddLine("Name", order.Name)
		t.AddLine("Created At", order.CreatedAt)
		t.AddLine("Updated At", order.UpdatedAt)
		t.AddLine("Canceled At", order.CancelledAt)
		t.AddLine("Closed At", order.ClosedAt)
		t.AddLine("Order Status URL", order.OrderStatusUrl)

		note := order.Note
		if len(note) > 0 {
			note = "\"" + note + "\""
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

func printLineItems(lines []shopify.LineItem) {
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
			Usage:   "Orders status to filter, defaults to 'open'",
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
