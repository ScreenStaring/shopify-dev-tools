package purchaseorders

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/cheynewallace/tabby"
	"github.com/urfave/cli/v2"

	"github.com/ScreenStaring/shopify-dev-tools/cmd"
)

var Cmd cli.Command

// column is one selectable output column of a purchase orders command. Value
// extracts the column's value from the command's row type.
type column[T any] struct {
	Name    string
	Aliases []string
	Value   func(T) string
}

// listColumns are the `purchaseorders ls` columns, in output order.
var listColumns = []column[PurchaseOrder]{
	{Name: "ID", Value: func(po PurchaseOrder) string { return purchaseOrderID(po.ID) }},
	{Name: "Purchase Order", Aliases: []string{"Name"}, Value: func(po PurchaseOrder) string { return po.Name }},
	{Name: "Supplier", Value: func(po PurchaseOrder) string { return po.Supplier }},
	{Name: "Destination", Value: func(po PurchaseOrder) string { return po.Destination }},
	{Name: "Status", Value: func(po PurchaseOrder) string { return po.Status }},
	{Name: "Linked Transfer", Aliases: []string{"Transfers", "Transfer"}, Value: func(po PurchaseOrder) string { return transferNames(po.Transfers) }},
	{Name: "Received", Value: func(po PurchaseOrder) string {
		received, total, ok := receivedQuantity(po.Transfers)
		if !ok {
			return ""
		}

		return fmt.Sprintf("%d/%d", received, total)
	}},
	{Name: "Total", Value: func(po PurchaseOrder) string { return po.Total }},
	{Name: "Expected Arrival", Value: func(po PurchaseOrder) string { return po.ExpectedArrival }},
}

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

	columns := matchingColumns(listColumns, c.String("fields"))

	orders, err := listPurchaseOrders(cmd.NewGraphQLClient(c), ids, c.String("status"), c.Int("limit"), c.Int("page"))
	if err != nil {
		return err
	}

	printPurchaseOrders(orders, columns)

	return nil
}

func printPurchaseOrders(orders []PurchaseOrder, columns []column[PurchaseOrder]) {
	if len(orders) == 0 {
		fmt.Println("No purchase orders")
		return
	}

	t := tabby.New()
	t.AddHeader(columnHeaders(columns)...)

	for _, order := range orders {
		t.AddLine(columnValues(columns, order)...)
	}

	t.Print()
}

// parseFields splits a --fields value into its field names.
func parseFields(value string) []string {
	var fields []string

	for _, field := range strings.Split(value, ",") {
		field = strings.TrimSpace(field)
		if len(field) == 0 {
			continue
		}
		fields = append(fields, field)
	}

	return fields
}

// matches reports whether the column is named by any of the given fields.
func (c column[T]) matches(fields []string) bool {
	names := append([]string{c.Name}, c.Aliases...)

	for _, name := range names {
		normalized := cmd.NormalizeField(name)
		for _, field := range fields {
			if cmd.NormalizeField(field) == normalized {
				return true
			}
		}
	}

	return false
}

// matchingColumns returns the columns named by a --fields value, in output
// order. Every column is returned when value is empty, and no column is
// returned when no field matches.
func matchingColumns[T any](columns []column[T], value string) []column[T] {
	fields := parseFields(value)
	if len(fields) == 0 {
		return columns
	}

	var selected []column[T]
	for _, col := range columns {
		if col.matches(fields) {
			selected = append(selected, col)
		}
	}

	return selected
}

// columnNames returns the columns' output headers.
func columnNames[T any](columns []column[T]) []string {
	names := make([]string, 0, len(columns))
	for _, col := range columns {
		names = append(names, col.Name)
	}

	return names
}

// columnHeaders returns the columns' headers in the form tabby takes.
func columnHeaders[T any](columns []column[T]) []interface{} {
	headers := make([]interface{}, 0, len(columns))
	for _, col := range columns {
		headers = append(headers, col.Name)
	}

	return headers
}

// columnValues returns the columns' values for the given row.
func columnValues[T any](columns []column[T], row T) []interface{} {
	values := make([]interface{}, 0, len(columns))
	for _, col := range columns {
		values = append(values, col.Value(row))
	}

	return values
}

func init() {
	limitFlag := &cli.IntFlag{
		Name:    "limit",
		Aliases: []string{"l"},
		Usage:   "Number of purchase orders per page, must be <= 250",
		Value:   10,
	}

	pageFlag := &cli.IntFlag{
		Name:    "page",
		Aliases: []string{"p"},
		Usage:   "Page of purchase orders to return, 1-based",
		Value:   1,
	}

	fieldsFlag := &cli.StringFlag{
		Name:    "fields",
		Aliases: []string{"f"},
		Usage:   "Comma separated list of fields to output",
	}

	listFlags := []cli.Flag{
		limitFlag,
		pageFlag,
		fieldsFlag,
		&cli.StringFlag{
			Name:    "status",
			Aliases: []string{"s"},
			Usage:   "Purchase order status to filter: DRAFT or ORDERED",
		},
		cmd.APIVersionFlag,
	}

	// Export honours the same paging flags as ls, but exports only the
	// purchase orders on the given page.
	exportFlags := []cli.Flag{
		limitFlag,
		pageFlag,
		fieldsFlag,
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
				Flags:   append(cmd.Flags, exportFlags...),
				Action:  exportAction,
			},
		},
	}
}
