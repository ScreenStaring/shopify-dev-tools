package inventory

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/cheynewallace/tabby"
	"github.com/urfave/cli/v2"

	"github.com/ScreenStaring/shopify-dev-tools/cmd"
	"github.com/ScreenStaring/shopify-dev-tools/cmd/locations"
	gqlclient "github.com/ScreenStaring/shopify-dev-tools/gql"
)

const locationGIDPrefix = "gid://shopify/Location/"
const transferGIDPrefix = "gid://shopify/InventoryTransfer/"

// toInventoryTransferGID returns the id as an InventoryTransfer GID,
// prepending the gid://shopify/InventoryTransfer/ prefix when given a bare id.
func toInventoryTransferGID(id string) string {
	if strings.HasPrefix(id, "gid://") {
		return id
	}

	return transferGIDPrefix + id
}

// resolveLocationIDs returns the ids of the locations matching the given
// values, which may be names, bare ids, or Location GIDs. An empty value
// resolves to an empty id.
func resolveLocationIDs(client *gqlclient.Client, values ...string) ([]string, error) {
	ids := make([]string, len(values))

	wanted := false
	for _, value := range values {
		if len(strings.TrimSpace(value)) > 0 {
			wanted = true
			break
		}
	}

	if !wanted {
		return ids, nil
	}

	locs, err := locations.ListLocations(client)
	if err != nil {
		return nil, err
	}

	for i, value := range values {
		if len(strings.TrimSpace(value)) == 0 {
			continue
		}

		id, err := matchLocationID(locs, value)
		if err != nil {
			return nil, err
		}

		ids[i] = id
	}

	return ids, nil
}

// matchLocationID returns the id of the location matching value, which may be
// a location name (case-insensitive), a bare id, or a Location GID.
func matchLocationID(locs []locations.Location, value string) (string, error) {
	value = strings.TrimSpace(value)
	bareID := strings.TrimPrefix(value, locationGIDPrefix)

	var matches []locations.Location
	for _, loc := range locs {
		id := strconv.FormatInt(loc.ID, 10)
		if bareID == id {
			return id, nil
		}

		if strings.EqualFold(loc.Name, value) {
			matches = append(matches, loc)
		}
	}

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("No location found with name or id '%s'", value)
	case 1:
		return strconv.FormatInt(matches[0].ID, 10), nil
	default:
		return "", fmt.Errorf("Location '%s' is ambiguous: %d locations have that name", value, len(matches))
	}
}

func transfersAction(c *cli.Context) error {
	client := cmd.NewGraphQLClient(c)

	args := c.Args().Slice()
	if len(args) > 0 {
		return transfersByIDs(c, client, args)
	}

	ids, err := resolveLocationIDs(client, c.String("origin"), c.String("destination"))
	if err != nil {
		return err
	}

	search := inventoryTransferSearchQuery(ids[0], ids[1], c.String("status"))

	transfers, err := listInventoryTransfers(client, search, c.Int("limit"), c.Int("page"))
	if err != nil {
		return err
	}

	printTransfersOrNotice(c, transfers)

	return nil
}

// transfersByIDs prints the transfers with the given ids, which may be bare
// ids or InventoryTransfer GIDs, reporting the ones that don't exist.
func transfersByIDs(c *cli.Context, client *gqlclient.Client, args []string) error {
	ids := make([]string, 0, len(args))
	argForGID := make(map[string]string, len(args))
	for _, arg := range args {
		gid := toInventoryTransferGID(arg)
		ids = append(ids, gid)
		argForGID[gid] = arg
	}

	transfers, missing, err := FetchInventoryTransfersByIDs(client, ids)
	if err != nil {
		return err
	}

	printTransfersOrNotice(c, transfers)
	reportMissing(c, "Inventory Transfer", missing, argForGID)

	if code := exitCodeFor(len(transfers), len(missing)); code != 0 {
		return cli.Exit("", code)
	}

	return nil
}

// exitCodeFor returns the exit code for a lookup of explicitly requested
// transfers: 1 when none of them exist and 2 when only some do, so callers can
// tell a bad id from a partially bad list.
func exitCodeFor(found, missing int) int {
	if missing == 0 {
		return 0
	}

	if found == 0 {
		return 1
	}

	return 2
}

func printTransfersOrNotice(c *cli.Context, transfers []Transfer) {
	if len(transfers) == 0 {
		if !c.Bool("jsonl") {
			fmt.Println("No inventory transfers")
		}
		return
	}

	if c.Bool("jsonl") {
		printTransfersJSONL(transfers)
	} else {
		printTransfers(transfers)
	}
}

func printTransfersJSONL(transfers []Transfer) {
	for _, transfer := range transfers {
		line, err := json.Marshal(transfer)
		if err != nil {
			panic(err)
		}

		fmt.Println(string(line))
	}
}

func printTransfers(transfers []Transfer) {
	for _, transfer := range transfers {
		t := tabby.New()
		t.AddLine("ID", idFromGID(transfer.ID))
		t.AddLine("Name", transfer.Name)
		t.AddLine("Created At", transfer.DateCreated)
		t.AddLine("Status", transfer.Status)
		t.AddLine("Origin", transferOrigin(transfer))
		t.AddLine("Destination", transfer.Destination)
		t.Print()

		fmt.Println("Line Items")
		printTransferLineItems(transfer.LineItems)
		fmt.Print("\n")

		cmd.PrintSeparator()
	}
}

// transferOrigin returns the transfer's origin location name, noting when the
// origin no longer links to a live location. Shopify keeps the name snapshot
// but leaves the location nil once the location is deleted or isn't
// accessible, which is indistinguishable from one that never was a location.
func transferOrigin(transfer Transfer) string {
	if len(transfer.Origin) == 0 {
		return transfer.Origin
	}

	if len(transfer.OriginLocationID) == 0 {
		return transfer.Origin + " (No Location)"
	}

	return transfer.Origin
}

func printTransferLineItems(lineItems []TransferLineItem) {
	x := tabby.New()
	x.AddHeader("ID", "Product", "Total Quantity", "Shipped Quantity")

	for _, line := range lineItems {
		x.AddLine(
			idFromGID(line.ID),
			lineItemTitle(line),
			line.TotalQuantity,
			line.ShippedQuantity,
		)
	}

	x.Print()
}

// defaultVariantTitle is the title Shopify gives a product's only variant.
const defaultVariantTitle = "Default Title"

// lineItemTitle returns the line item's product title, with the variant title
// appended when the item isn't the product's default variant.
func lineItemTitle(line TransferLineItem) string {
	if len(line.VariantTitle) == 0 || line.VariantTitle == defaultVariantTitle {
		return line.Title
	}

	return line.Title + " - " + line.VariantTitle
}

// transfersCommand builds the `inventory transfers` subcommand.
func transfersCommand() *cli.Command {
	apiVersionFlag := cmd.APIVersionFlag

	transferFlags := []cli.Flag{
		&cli.StringFlag{
			Name:    "origin",
			Aliases: []string{"o"},
			Usage:   "Only list transfers originating from the location with the given name or id",
		},
		&cli.StringFlag{
			Name:    "destination",
			Aliases: []string{"d"},
			Usage:   "Only list transfers destined for the location with the given name or id",
		},
		&cli.StringFlag{
			Name:    "status",
			Aliases: []string{"s"},
			Usage:   "Only list transfers with the given GraphQL transfer status",
		},
		&cli.IntFlag{
			Name:    "limit",
			Aliases: []string{"l"},
			Usage:   "Number of inventory transfers per page, must be <= 250",
			Value:   10,
		},
		&cli.IntFlag{
			Name:    "page",
			Aliases: []string{"p"},
			Usage:   "Page of inventory transfers to return, 1-based",
			Value:   1,
		},
		&cli.BoolFlag{
			Name:    "jsonl",
			Aliases: []string{"j"},
			Usage:   "Output the inventory transfers in JSONL format",
		},
	}

	return &cli.Command{
		Name:        "transfers",
		Aliases:     []string{"tx"},
		Usage:       "List the shop's inventory transfers, or those with the given IDs",
		ArgsUsage:   "[ID [ID ...]]",
		Description: "Without IDs every transfer is listed",
		Flags:       append(cmd.Flags, append(transferFlags, apiVersionFlag)...),
		Action:      transfersAction,
	}
}
