package orders

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/ScreenStaring/shopify-dev-tools/cmd/orders/bulk"
	"github.com/ScreenStaring/shopify-dev-tools/gql"
)

type csvRow struct {
	orderName         string
	orderID           string
	fulfillmentStatus string
	quantity          int
	sku               string
	barcode           string
	variantID         string
	notes             string
	email             string
}

type orderGroup struct {
	orderName         string
	orderID           string
	fulfillmentStatus string
	notes             string
	hasNotes          bool
	email             string
	startRow          int
	rows              []csvRow
}

func buildBulkColumnIndex(header []string) map[string]int {
	idx := make(map[string]int)
	for i, col := range header {
		idx[strings.ToLower(strings.TrimSpace(col))] = i
	}
	return idx
}

func bulkColVal(row []string, idx int) string {
	if idx < 0 || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}

func parseOrderCSV(filename string) ([]orderGroup, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("cannot open CSV file: %s", err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("cannot read CSV: %s", err)
	}

	if len(records) < 2 {
		return nil, fmt.Errorf("CSV file has no data rows")
	}

	ci := buildBulkColumnIndex(records[0])

	get := func(row []string, name string) string {
		if idx, ok := ci[name]; ok {
			return bulkColVal(row, idx)
		}
		return ""
	}

	_, hasNotes := ci["notes"]

	groupMap := make(map[string]*orderGroup)
	var groupOrder []string

	for i, row := range records[1:] {
		rowNum := i + 2 // 1-indexed, skip header
		orderID := get(row, "id")
		orderName := get(row, "name")

		key := orderID
		if key == "" {
			key = orderName
		}
		if key == "" {
			continue
		}

		qtyStr := get(row, "lineitem quantity")
		qty := 0
		if qtyStr != "" {
			qty, _ = strconv.Atoi(qtyStr)
		}

		r := csvRow{
			orderName:         orderName,
			orderID:           orderID,
			fulfillmentStatus: strings.ToLower(get(row, "fulfillment status")),
			quantity:          qty,
			sku:               get(row, "lineitem sku"),
			barcode:           get(row, "barcode"),
			variantID:         get(row, "variant id"),
			notes:             get(row, "notes"),
			email:             strings.ToLower(get(row, "email")),
		}

		if g, ok := groupMap[key]; ok {
			g.rows = append(g.rows, r)
		} else {
			g = &orderGroup{
				orderName: orderName,
				orderID:   orderID,
				hasNotes:  hasNotes,
				startRow:  rowNum,
				rows:      []csvRow{r},
			}
			groupMap[key] = g
			groupOrder = append(groupOrder, key)
		}
	}

	var groups []orderGroup
	for _, key := range groupOrder {
		g := groupMap[key]
		for _, r := range g.rows {
			if g.fulfillmentStatus == "" && r.fulfillmentStatus != "" {
				g.fulfillmentStatus = r.fulfillmentStatus
			}
			if g.notes == "" && r.notes != "" {
				g.notes = r.notes
			}
			if g.email == "" && r.email != "" {
				g.email = r.email
			}
		}
		groups = append(groups, *g)
	}

	return groups, nil
}

type existingItem struct {
	lineItemID string
	variantID  string
	quantity   int
	sku        string
	barcode    string
}

type quantityChange struct {
	lineItemID string
	quantity   int
}

type newItem struct {
	variantGID string
	quantity   int
}

func processOrder(client *gql.Client, group orderGroup, customerMap map[string]string) (string, []string) {
	var errors []string
	mutated := false

	order, err := bulk.FetchOrder(client, group.orderID, group.orderName)
	if err != nil {
		return "Failed", []string{err.Error()}
	}

	// Build maps of existing line items by SKU and barcode
	skuMap := make(map[string]*existingItem)
	barcodeMap := make(map[string]*existingItem)

	for _, edge := range order.LineItems.Edges {
		li := edge.Node
		item := &existingItem{
			lineItemID: li.ID,
			quantity:   li.Quantity,
			sku:        li.SKU,
		}
		if li.Variant != nil {
			item.variantID = li.Variant.ID
			item.barcode = li.Variant.Barcode
		}
		if li.SKU != "" {
			skuMap[li.SKU] = item
		}
		if item.barcode != "" {
			barcodeMap[item.barcode] = item
		}
	}

	// Determine changes
	var quantityChanges []quantityChange
	var newItems []newItem

	for _, row := range group.rows {
		if row.sku == "" && row.barcode == "" && row.variantID == "" {
			continue
		}

		// Match existing line item by SKU first, then barcode
		var matched *existingItem
		if row.sku != "" {
			matched = skuMap[row.sku]
		}
		if matched == nil && row.barcode != "" {
			matched = barcodeMap[row.barcode]
		}

		if matched != nil {
			if row.quantity != matched.quantity {
				quantityChanges = append(quantityChanges, quantityChange{
					lineItemID: matched.lineItemID,
					quantity:   row.quantity,
				})
			}
		} else {
			// New item - resolve variant GID
			variantGID := ""
			if row.variantID != "" {
				variantGID = fmt.Sprintf("gid://shopify/ProductVariant/%s", row.variantID)
			} else if row.sku != "" {
				variantGID, err = bulk.FindVariantBySKU(client, row.sku)
				if err != nil {
					errors = append(errors, fmt.Sprintf("finding variant for SKU %s: %s", row.sku, err))
					continue
				}
			} else if row.barcode != "" {
				variantGID, err = bulk.FindVariantByBarcode(client, row.barcode)
				if err != nil {
					errors = append(errors, fmt.Sprintf("finding variant for barcode %s: %s", row.barcode, err))
					continue
				}
			}

			if variantGID == "" {
				errors = append(errors, "cannot resolve variant for line item")
				continue
			}

			newItems = append(newItems, newItem{
				variantGID: variantGID,
				quantity:   row.quantity,
			})
		}
	}

	// Apply order edits (quantity changes + new items)
	if len(quantityChanges) > 0 || len(newItems) > 0 {
		mutated = true

		calcOrderID, _, err := bulk.BeginOrderEdit(client, order.ID)
		if err != nil {
			errors = append(errors, err.Error())
			return "Failed", errors
		}

		editFailed := false

		for _, change := range quantityChanges {
			calcLineItemID := strings.Replace(change.lineItemID, "gid://shopify/LineItem/", "gid://shopify/CalculatedLineItem/", 1)
			if err := bulk.SetLineItemQuantity(client, calcOrderID, calcLineItemID, change.quantity); err != nil {
				errors = append(errors, fmt.Sprintf("setting quantity: %s", err))
				editFailed = true
				break
			}
		}

		if !editFailed {
			for _, item := range newItems {
				if err := bulk.AddLineItemVariant(client, calcOrderID, item.variantGID, item.quantity); err != nil {
					errors = append(errors, fmt.Sprintf("adding variant: %s", err))
					editFailed = true
					break
				}
			}
		}

		if !editFailed {
			if err := bulk.CommitOrderEdit(client, calcOrderID); err != nil {
				errors = append(errors, fmt.Sprintf("committing edits: %s", err))
			}
		}
	}

	// Update notes
	if group.hasNotes && group.notes != strings.TrimSpace(order.Note) {
		mutated = true
		if err := bulk.UpdateOrderNote(client, order.ID, group.notes); err != nil {
			errors = append(errors, fmt.Sprintf("updating notes: %s", err))
		}
	}

	// Set customer and/or order email
	if group.email != "" {
		currentCustomerEmail := ""
		if order.Customer != nil {
			currentCustomerEmail = strings.ToLower(order.Customer.Email)
		}

		if currentCustomerEmail != group.email {
			mutated = true
			customerGID, ok := customerMap[group.email]
			if !ok {
				errors = append(errors, fmt.Sprintf("customer not found for email %s", group.email))
			} else if err := bulk.SetOrderCustomer(client, order.ID, customerGID); err != nil {
				errors = append(errors, fmt.Sprintf("setting customer: %s", err))
			}
		}

		if strings.ToLower(order.Email) != group.email {
			mutated = true
			if err := bulk.UpdateOrderEmail(client, order.ID, group.email); err != nil {
				errors = append(errors, fmt.Sprintf("updating order email: %s", err))
			}
		}
	}

	// Handle fulfillment status changes
	currentStatus := strings.ToLower(order.DisplayFulfillmentStatus)
	desiredStatus := group.fulfillmentStatus

	if desiredStatus != "" && desiredStatus != currentStatus {
		mutated = true
		if desiredStatus == "fulfilled" {
			for _, edge := range order.FulfillmentOrders.Edges {
				fo := edge.Node
				if fo.Status == "OPEN" || fo.Status == "IN_PROGRESS" {
					if err := bulk.CreateFulfillment(client, fo.ID); err != nil {
						errors = append(errors, fmt.Sprintf("fulfilling: %s", err))
					}
				}
			}
		} else if desiredStatus == "unfulfilled" {
			for _, f := range order.Fulfillments {
				if f.Status == "SUCCESS" {
					if err := bulk.CancelFulfillment(client, f.ID); err != nil {
						errors = append(errors, fmt.Sprintf("canceling fulfillment: %s", err))
					}
				}
			}
		}
	}

	if !mutated {
		return "No change", errors
	}

	if len(errors) > 0 {
		return "Failed", errors
	}

	return "Updated", errors
}

