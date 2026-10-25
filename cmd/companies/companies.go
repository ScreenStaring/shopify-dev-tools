package companies

import (
	"fmt"
	"strings"

	"github.com/cheynewallace/tabby"
	"github.com/urfave/cli/v2"

	"github.com/ScreenStaring/shopify-dev-tools/cmd"
	"github.com/ScreenStaring/shopify-dev-tools/cmd/draftorders"
	"github.com/ScreenStaring/shopify-dev-tools/cmd/orders"
	gqlclient "github.com/ScreenStaring/shopify-dev-tools/gql"
)

var Cmd cli.Command

func listAction(c *cli.Context) error {
	client := cmd.NewGraphQLClient(c)
	includeDrafts := c.Bool("include-drafts")

	if c.NArg() > 0 {
		return listByIDs(c, client, includeDrafts)
	}

	companies, err := FetchCompanies(client, c.Int("limit"), includeDrafts)
	if err != nil {
		return err
	}

	if len(companies) == 0 {
		fmt.Println("No companies")
		return nil
	}

	printCompanies(companies, includeDrafts)

	return nil
}

// listByIDs prints the companies with the given ids, which may be bare ids or
// Company GIDs, reporting the ones that don't exist.
func listByIDs(c *cli.Context, client *gqlclient.Client, includeDrafts bool) error {
	args := c.Args().Slice()

	ids := make([]string, 0, len(args))
	argForGID := make(map[string]string, len(args))
	for _, arg := range args {
		gid := ToCompanyGID(arg)
		ids = append(ids, gid)
		argForGID[gid] = arg
	}

	companies, missing, err := FetchCompaniesByIDs(client, ids, includeDrafts)
	if err != nil {
		return err
	}

	if len(companies) == 0 {
		fmt.Println("No companies")
	} else {
		printCompanies(companies, includeDrafts)
	}

	if len(missing) > 0 {
		notFound := make([]string, 0, len(missing))
		for _, gid := range missing {
			notFound = append(notFound, argForGID[gid])
		}

		fmt.Printf("Company(s) not found: %s\n", strings.Join(notFound, ", "))
	}

	return nil
}

func printCompanies(companies []Company, includeDrafts bool) {
	for _, company := range companies {
		t := tabby.New()
		t.AddLine("ID", strings.TrimPrefix(company.ID, companyGIDPrefix))
		t.AddLine("Name", company.Name)
		t.AddLine("External ID", company.ExternalID)
		t.AddLine("Customer Since", company.CustomerSince)
		t.AddLine("Created", company.CreatedAt)
		t.AddLine("Updated", company.UpdatedAt)

		if len(company.Note) > 0 {
			t.AddLine("Note", fmt.Sprintf("%q", company.Note))
		}

		t.Print()

		if includeDrafts {
			fmt.Println("Recent Draft Orders")
			printRecentDraftOrders(company.DraftOrders)
			fmt.Print("\n")
		}

		fmt.Println("Recent Orders")
		printRecentOrders(company.Orders)
		fmt.Print("\n")

		cmd.PrintSeparator()
	}
}

// printRecentDraftOrders shows a company's most recent draft orders, one per
// row, along with the SKUs and quantities of each draft order's line items.
func printRecentDraftOrders(recent []draftorders.DraftOrder) {
	if len(recent) == 0 {
		fmt.Println("None")
		return
	}

	t := tabby.New()
	t.AddHeader("ID", "Name", "Status", "Created At", "Line Items")

	for _, draftOrder := range recent {
		t.AddLine(
			draftOrder.ID,
			draftOrder.Name,
			draftOrder.Status,
			draftOrder.CreatedAt,
			draftOrderLineItemsSummary(draftOrder.LineItems),
		)
	}

	t.Print()
}

// printRecentOrders shows a company's most recent orders, one per row, along
// with the SKUs and quantities of each order's line items.
func printRecentOrders(recent []orders.Order) {
	if len(recent) == 0 {
		fmt.Println("None")
		return
	}

	t := tabby.New()
	t.AddHeader("ID", "Name", "Created At", "Financial Status", "Fulfillment Status", "Line Items")

	for _, order := range recent {
		t.AddLine(
			order.ID,
			order.Name,
			order.CreatedAt,
			order.DisplayFinancialStatus,
			order.DisplayFulfillmentStatus,
			lineItemsSummary(order.LineItems),
		)
	}

	t.Print()
}

// lineItemsSummary formats line items as "SKU x QUANTITY" pairs.
func lineItemsSummary(lines []orders.LineItem) string {
	parts := make([]string, 0, len(lines))

	for _, line := range lines {
		parts = append(parts, lineItemSummary(line.SKU, line.Name, line.Quantity))
	}

	return strings.Join(parts, ", ")
}

// draftOrderLineItemsSummary formats draft order line items as
// "SKU x QUANTITY" pairs.
func draftOrderLineItemsSummary(lines []draftorders.LineItem) string {
	parts := make([]string, 0, len(lines))

	for _, line := range lines {
		parts = append(parts, lineItemSummary(line.SKU, line.Name, line.Quantity))
	}

	return strings.Join(parts, ", ")
}

// lineItemSummary formats a single line item, falling back to its name when it
// has no SKU.
func lineItemSummary(sku, name string, quantity int) string {
	if len(sku) == 0 {
		sku = name
	}

	return fmt.Sprintf("%s x %d", shorten(sku), quantity)
}

func shorten(name string) string {
	const limit = 25

	runes := []rune(name)
	if len(runes) <= limit {
		return name
	}

	return strings.TrimRight(string(runes[:limit]), " ") + "…"
}

func init() {
	apiVersionFlag := cmd.APIVersionFlag

	listFlags := []cli.Flag{
		&cli.IntFlag{
			Name:    "limit",
			Aliases: []string{"l"},
			Usage:   "Maximum number of companies to return, must be <= 250",
			Value:   10,
		},
		&cli.BoolFlag{
			Name:    "include-drafts",
			Aliases: []string{"d"},
			Usage:   "Also show each company's 5 most recent draft orders",
		},
	}

	Cmd = cli.Command{
		Name:    "companies",
		Aliases: []string{"com"},
		Usage:   "Do things with companies",

		Subcommands: []*cli.Command{
			{
				Name:      "ls",
				Aliases:   []string{"l"},
				ArgsUsage: "[ID [ID ...]]",
				Usage:     "List the shop's companies or the companies given by the specified IDs",
				Flags:     append(cmd.Flags, append(listFlags, apiVersionFlag)...),
				Action:    listAction,
			},
		},
	}
}
