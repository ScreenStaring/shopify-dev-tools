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

func exportAction(c *cli.Context) error {
	shop := c.String("shop")
	client := cmd.NewGraphQLClient(c)

	ids, err := fetchPurchaseOrderIDs(client)
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

	if err := w.Write([]string{
		"ID",
		"Purchase Order",
		"Supplier",
		"Destination",
		"Status",
		"Transfers",
		"Line Item",
		"SKU",
		"Quantity",
		"Unit Cost",
		"Unit Cost Currency",
		"Total Cost",
		"Total Cost Currency",
	}); err != nil {
		return fmt.Errorf("Cannot write CSV header: %s", err)
	}

	for i, id := range ids {
		order, err := fetchPurchaseOrderExport(client, id)
		if err != nil {
			return err
		}

		if err := writePurchaseOrderRow(w, order); err != nil {
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

// writePurchaseOrderRow writes one row per line item. Purchase orders without
// line items still get a row so they appear in the export.
func writePurchaseOrderRow(w *csv.Writer, order ExportPurchaseOrder) error {
	head := []string{
		purchaseOrderID(order.ID),
		order.Name,
		order.Supplier,
		order.Destination,
		order.Status,
		transferNames(order.Transfers),
	}

	write := func(fields ...string) error {
		row := append(append([]string{}, head...), fields...)
		if err := w.Write(row); err != nil {
			return fmt.Errorf("Cannot write CSV row for purchase order %s: %s", order.Name, err)
		}

		return nil
	}

	if len(order.LineItems) == 0 {
		return write("", "", "", "", "", "", "")
	}

	for _, line := range order.LineItems {
		if err := write(
			line.Title,
			line.SKU,
			strconv.Itoa(line.Quantity),
			line.UnitCost,
			line.UnitCurrency,
			line.TotalCost,
			line.TotalCurrency,
		); err != nil {
			return err
		}
	}

	return nil
}
