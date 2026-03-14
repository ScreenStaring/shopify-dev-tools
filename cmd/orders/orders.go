package orders

import (
	"fmt"
	"strings"

	"github.com/urfave/cli/v2"
	"github.com/ScreenStaring/shopify-dev-tools/cmd"
	"github.com/ScreenStaring/shopify-dev-tools/cmd/orders/bulk"
	"github.com/ScreenStaring/shopify-dev-tools/gql"
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
	orders, err := listOrders(shop, cmd.LookupAccessToken(shop, c.String("access-token")), c.String("api-version"), ids, status, c.Int("limit"))
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

func bulkAction(c *cli.Context) error {
	if c.Args().Len() == 0 {
		return fmt.Errorf("you must supply a CSV file path")
	}

	filename := c.Args().Get(0)

	groups, err := parseOrderCSV(filename)
	if err != nil {
		return err
	}

	shop := c.String("shop")
	token := cmd.LookupAccessToken(shop, c.String("access-token"))
	client := gql.NewClient(shop, token, map[string]interface{}{"version": c.String("api-version"), "verbose": c.Bool("verbose")})

	// Batch lookup customers by email
	customerMap := make(map[string]string)
	for _, group := range groups {
		if group.email == "" {
			continue
		}
		if _, ok := customerMap[group.email]; ok {
			continue
		}

		customerGID, err := bulk.FindCustomerByEmail(client, group.email)
		if err != nil {
			return err;
		}
		customerMap[group.email] = customerGID
	}

	fmt.Printf("Processing %d orders from %s\n\n", len(groups), filename)

	for _, group := range groups {
		orderRef := group.orderName
		if orderRef == "" {
			orderRef = group.orderID
		}

		status, errs := processOrder(client, group, customerMap)

		t := tabby.New()
		t.AddLine("Row", group.startRow)
		t.AddLine("Order", orderRef)

		for _, row := range group.rows {
			parts := []string{}
			if row.sku != "" {
				parts = append(parts, "SKU: "+row.sku)
			}
			if row.barcode != "" {
				parts = append(parts, "Barcode: "+row.barcode)
			}
			if row.variantID != "" {
				parts = append(parts, "Variant: "+row.variantID)
			}
			if len(parts) > 0 {
				t.AddLine("Line Item", strings.Join(parts, ", "))
			}
		}

		t.AddLine("Result", status)

		if len(errs) > 0 {
			t.AddLine("Errors", strings.Join(errs, "; "))
		}

		t.Print()
		cmd.PrintSeparator()
	}

	return nil
}

func init() {
	apiVersionFlag := &cli.StringFlag{
		Name:    "api-version",
		Aliases: []string{"a"},
		Usage:   "API version to use; default is a versionless call",
	}

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

	baseFlags := append(cmd.Flags, apiVersionFlag)

	Cmd = cli.Command{
		Name:  "orders",
		Aliases: []string{"o"},
		Usage:   "Information about orders",

		Subcommands: []*cli.Command{
			{
				Name: "useragent",
				Aliases: []string{"ua"},
				Usage:   "Info about the web browser used to place the order",
				Flags: baseFlags,
				Action: userAgentAction,
			},
			{
				Name: "ls",
				Usage:   "List the shop's orders or the orders given by the specified IDs",
				Flags: append(baseFlags, ordersFlags...),
				Action: listAction,
			},
			{
				Name:      "bulk",
				Aliases:   []string{"b"},
				Usage:     "Bulk edit orders from a CSV spreadsheet",
				ArgsUsage: "orders.csv",
				Flags:     baseFlags,
				Action: bulkAction,
			},
		},
	}
}
