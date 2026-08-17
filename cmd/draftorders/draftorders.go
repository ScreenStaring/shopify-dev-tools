package draftorders

import (
	"encoding/json"
	"fmt"

	"github.com/ScreenStaring/shopify-dev-tools/cmd"
	"github.com/cheynewallace/tabby"
	"github.com/urfave/cli/v2"
)

var Cmd cli.Command

func listAction(c *cli.Context) error {
	ids, skus, err := cmd.ParseIDArgs(c, "a draft order id")
	if err != nil {
		return err
	}

	status := "open"
	if len(c.String("status")) > 0 {
		status = c.String("status")
	}

	sortKey, err := ResolveDraftOrderSortKey(c.String("sort"))
	if err != nil {
		return err
	}

	shop := c.String("shop")
	orders, err := listDraftOrders(shop, cmd.LookupAccessToken(shop, c.String("access-token")), ids, skus, status, c.Int("limit"), sortKey, c.String("api-version"))
	if err != nil {
		return err
	}

	if c.Bool("jsonl") {
		printJSONL(orders)
		return nil
	}

	printDraftOrders(orders)

	return nil
}

// printJSONL outputs the draft orders one JSON object per line.
func printJSONL(orders []DraftOrder) {
	for _, order := range orders {
		line, err := json.Marshal(order)
		if err != nil {
			panic(err)
		}

		fmt.Println(string(line))
	}
}

func printDraftOrders(orders []DraftOrder) {
	t := tabby.New()
	for _, order := range orders {
		t.AddLine("Id", order.ID)
		t.AddLine("Name", order.Name)
		t.AddLine("Status", order.Status)
		t.AddLine("Created At", order.CreatedAt)
		t.AddLine("Updated At", order.UpdatedAt)
		t.AddLine("Completed At", order.CompletedAt)
		t.AddLine("Invoice Sent At", order.InvoiceSentAt)
		t.AddLine("Reserve Inventory Until", order.ReserveInventoryUntil)
		t.AddLine("Order ID", order.OrderID)

		note := order.Note
		if len(order.Note) > 0 {
			note = fmt.Sprintf("%q", order.Note)
		}

		t.AddLine("Note", note)
		t.Print()

		fmt.Println("Line Items")
		printLineItems(order.LineItems)
		fmt.Print("\n")

		cmd.PrintSeparator()
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
	x.AddHeader("ID", "Product ID", "Variant ID", "SKU", "Title", "Quantity")

	for _, line := range lines {
		x.AddLine(
			line.ID,
			line.ProductID,
			line.VariantID,
			line.SKU,
			truncate(line.Name),
			line.Quantity,
		)
	}

	x.Print()
}

func init() {
	draftOrdersFlags := []cli.Flag{
		&cli.StringFlag{
			Name:    "status",
			Aliases: []string{"s"},
			Usage:   "GraphQL Admin API draft order status to filter, defaults to 'open'",
		},
		&cli.IntFlag{
			Name:    "limit",
			Aliases: []string{"l"},
			Usage:   "Maximum number of draft orders to return, must be <= 250",
			Value:   10,
		},
		&cli.StringFlag{
			Name:  "sort",
			Usage: "GQL sort enum value, lowercase accepted",
		},
		&cli.StringFlag{
			Name:    "version",
			Aliases: []string{"api-version"},
			Usage:   "API version to use; default is a versionless call",
		},
		&cli.BoolFlag{
			Name:    "jsonl",
			Aliases: []string{"j"},
			Usage:   "Output the draft orders in JSONL format",
		},
	}

	Cmd = cli.Command{
		Name:    "draftorders",
		Aliases: []string{"do"},
		Usage:   "Information about draft orders",

		Subcommands: []*cli.Command{
			{
				Name:    "ls",
				Aliases: []string{"l"},
				Usage:   "List the shop's draft orders or the draft orders matching the given IDs and/or 'sku:VALUE' arguments",
				Flags:   append(cmd.Flags, draftOrdersFlags...),
				Action:  listAction,
			},
		},
	}
}
