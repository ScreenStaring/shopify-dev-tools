package purchaseorders

import (
	"fmt"
	"strconv"

	"github.com/cheynewallace/tabby"
	"github.com/urfave/cli/v2"

	"github.com/ScreenStaring/shopify-dev-tools/cmd"
)

var Cmd cli.Command

// purchaseOrderIDs parses purchase order IDs as given on the command line.
func purchaseOrderIDs(args []string) ([]int64, error) {
	ids := make([]int64, 0, len(args))

	for _, arg := range args {
		id, err := strconv.ParseInt(arg, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("Argument '%s' invalid: must be a purchase order id", arg)
		}
		ids = append(ids, id)
	}

	return ids, nil
}

func listAction(c *cli.Context) error {
	ids, err := purchaseOrderIDs(c.Args().Slice())
	if err != nil {
		return err
	}

	orders, err := listPurchaseOrders(cmd.NewGraphQLClient(c), ids, c.String("status"), c.Int("limit"), c.Int("page"))
	if err != nil {
		return err
	}

	printPurchaseOrders(orders)

	return nil
}

func printPurchaseOrders(orders []PurchaseOrder) {
	if len(orders) == 0 {
		fmt.Println("No purchase orders")
		return
	}

	t := tabby.New()
	t.AddHeader("ID", "Purchase Order", "Supplier", "Destination", "Status", "Linked Transfer", "Received", "Total", "Expected Arrival")

	for _, order := range orders {
		received := ""
		if quantity, total, ok := receivedQuantity(order.Transfers); ok {
			received = fmt.Sprintf("%d/%d", quantity, total)
		}

		t.AddLine(
			purchaseOrderID(order.ID),
			order.Name,
			order.Supplier,
			order.Destination,
			order.Status,
			transferNames(order.Transfers),
			received,
			order.Total,
			order.ExpectedArrival,
		)
	}

	t.Print()
}

func init() {
	listFlags := []cli.Flag{
		&cli.IntFlag{
			Name:    "limit",
			Aliases: []string{"l"},
			Usage:   "Number of purchase orders per page, must be <= 250",
			Value:   10,
		},
		&cli.IntFlag{
			Name:    "page",
			Aliases: []string{"p"},
			Usage:   "Page of purchase orders to return, 1-based",
			Value:   1,
		},
		&cli.StringFlag{
			Name:    "status",
			Aliases: []string{"s"},
			Usage:   "Purchase order status to filter: DRAFT or ORDERED",
		},
		cmd.APIVersionFlag,
	}

	Cmd = cli.Command{
		Name:    "purchaseorders",
		Aliases: []string{"po"},
		Usage:   "Information about purchase orders",

		Subcommands: []*cli.Command{
			{
				Name:      "ls",
				Aliases:   []string{"l"},
				Usage:     "List the shop's purchase orders or the purchase orders matching the given IDs",
				ArgsUsage: "[PURCHASE ORDER ID [PURCHASE ORDER ID ...]]",
				Flags:     append(cmd.Flags, listFlags...),
				Action:    listAction,
			},
			{
				Name:    "export",
				Aliases: []string{"e", "x"},
				Usage:   "Export the shop's purchase orders and their line items to a CSV file",
				Flags:   append(cmd.Flags, cmd.APIVersionFlag),
				Action:  exportAction,
			},
		},
	}
}
