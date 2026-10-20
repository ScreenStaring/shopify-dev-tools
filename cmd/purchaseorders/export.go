package purchaseorders

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/ScreenStaring/shopify-dev-tools/cmd"
)

func exportBaseName(shop string) string {
	return strings.SplitN(shop, ".", 2)[0] + "-purchase-orders"
}

// exportRow is one CSV row: a purchase order and, when it has line items, the
// line item the row is for.
type exportRow struct {
	order ExportPurchaseOrder
	line  *ExportLineItem
}

// lineValue returns fn applied to the row's line item, or "" when the row has
// no line item.
func (r exportRow) lineValue(fn func(ExportLineItem) string) string {
	if r.line == nil {
		return ""
	}

	return fn(*r.line)
}

// exportColumns are the `purchaseorders export` columns, in output order.
var exportColumns = []column[exportRow]{
	{Name: "ID", Value: func(r exportRow) string { return purchaseOrderID(r.order.ID) }},
	{Name: "Purchase Order", Aliases: []string{"Name"}, Value: func(r exportRow) string { return r.order.Name }},
	{Name: "Supplier", Value: func(r exportRow) string { return r.order.Supplier }},
	{Name: "Destination", Value: func(r exportRow) string { return r.order.Destination }},
	{Name: "Status", Value: func(r exportRow) string { return r.order.Status }},
	{Name: "Transfers", Aliases: []string{"Transfer", "Linked Transfer"}, Value: func(r exportRow) string { return transferNames(r.order.Transfers) }},
	{Name: "Line Item", Aliases: []string{"Title"}, Value: func(r exportRow) string { return r.lineValue(func(li ExportLineItem) string { return li.Title }) }},
	{Name: "SKU", Value: func(r exportRow) string { return r.lineValue(func(li ExportLineItem) string { return li.SKU }) }},
	{Name: "Quantity", Value: func(r exportRow) string {
		return r.lineValue(func(li ExportLineItem) string { return strconv.Itoa(li.Quantity) })
	}},
	{Name: "Unit Cost", Value: func(r exportRow) string { return r.lineValue(func(li ExportLineItem) string { return li.UnitCost }) }},
	{Name: "Unit Cost Currency", Value: func(r exportRow) string {
		return r.lineValue(func(li ExportLineItem) string { return li.UnitCurrency })
	}},
	{Name: "Total Cost", Value: func(r exportRow) string { return r.lineValue(func(li ExportLineItem) string { return li.TotalCost }) }},
	{Name: "Total Cost Currency", Value: func(r exportRow) string {
		return r.lineValue(func(li ExportLineItem) string { return li.TotalCurrency })
	}},
}

func exportAction(c *cli.Context) error {
	shop := c.String("shop")
	client := cmd.NewGraphQLClient(c)

	columns := matchingColumns(exportColumns, c.String("fields"))

	ids, err := fetchPurchaseOrderIDs(client, c.Int("limit"), c.Int("page"))
	if err != nil {
		return err
	}

	filename := exportBaseName(shop) + ".csv"

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("Cannot create CSV file: %s", err)
	}
	defer file.Close()

	w := csv.NewWriter(file)

	if err := writeCSVHeader(w, columns); err != nil {
		return err
	}

	for i, id := range ids {
		order, err := fetchPurchaseOrderExport(client, id)
		if err != nil {
			return err
		}

		if err := writePurchaseOrderRow(w, order, columns); err != nil {
			return err
		}

		fmt.Fprintf(os.Stderr, "\rExported %d/%d", i+1, len(ids))
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "\nWrote %s\n", filename)

	return nil
}

// writeCSVHeader writes the given columns' headers.
func writeCSVHeader(w *csv.Writer, columns []column[exportRow]) error {
	if len(columns) == 0 {
		return nil
	}

	if err := w.Write(columnNames(columns)); err != nil {
		return fmt.Errorf("Cannot write CSV header: %s", err)
	}

	return nil
}

// writePurchaseOrderRow writes one row per line item, restricted to the given
// columns. Purchase orders without line items still get a row so they appear
// in the export.
func writePurchaseOrderRow(w *csv.Writer, order ExportPurchaseOrder, columns []column[exportRow]) error {
	if len(columns) == 0 {
		return nil
	}

	write := func(row exportRow) error {
		values := make([]string, 0, len(columns))
		for _, col := range columns {
			values = append(values, col.Value(row))
		}

		if err := w.Write(values); err != nil {
			return fmt.Errorf("Cannot write CSV row for purchase order %s: %s", order.Name, err)
		}

		return nil
	}

	if len(order.LineItems) == 0 {
		return write(exportRow{order: order})
	}

	for i := range order.LineItems {
		if err := write(exportRow{order: order, line: &order.LineItems[i]}); err != nil {
			return err
		}
	}

	return nil
}
