package charges

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/cheynewallace/tabby"
	"github.com/shopspring/decimal"
	"github.com/urfave/cli/v2"

	"github.com/ScreenStaring/shopify-dev-tools/cmd"
)

var Cmd cli.Command

func printRecordSeperator() {
	fmt.Printf("%s\n", strings.Repeat("-", 20))
}

func printJSONL(charges []RecurringCharge) {
	for _, charge := range charges {
		printChargeJSONL(charge)
	}
}

func printChargeJSONL(charge interface{}) {
	line, err := json.Marshal(charge)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(line))
}

func printFormattedRecurringCharge(charge *RecurringCharge) {
	t := tabby.New()

	t.AddLine("Id", charge.ID)
	t.AddLine("Type", "Recurring")
	t.AddLine("Name", charge.Name)
	t.AddLine("Price", charge.Price)
	t.AddLine("Status", charge.Status)
	t.AddLine("Return URL", charge.ReturnURL)

	// The GraphQL API only exposes the confirmation URL on the create
	// mutation response, not on existing subscriptions
	if len(charge.ConfirmationURL) > 0 {
		t.AddLine("Confirmation URL", charge.ConfirmationURL)
	}

	t.AddLine("Test", charge.Test)
	t.AddLine("Created At", charge.CreatedAt)
	t.Print()
}

func printFormattedRecurringCharges(charges []RecurringCharge) {
	for _, charge := range charges {
		printFormattedRecurringCharge(&charge)
		printRecordSeperator()
	}
}

func printFormattedApplicationCharges(charges []OneTimeCharge) {
	for _, charge := range charges {
		printFormattedApplicationCharge(&charge)
		printRecordSeperator()
	}
}

func printFormattedApplicationCharge(charge *OneTimeCharge) {
	t := tabby.New()

	t.AddLine("Id", charge.ID)
	t.AddLine("Type", "One-Time")
	t.AddLine("Name", charge.Name)
	t.AddLine("Price", charge.Price)
	t.AddLine("Status", charge.Status)

	// The GraphQL API only exposes the confirmation and return URLs on
	// the create mutation response, not on existing purchases
	if len(charge.ConfirmationURL) > 0 {
		t.AddLine("Confirmation URL", charge.ConfirmationURL)
	}

	if len(charge.ReturnURL) > 0 {
		t.AddLine("Return URL", charge.ReturnURL)
	}

	t.AddLine("Test", charge.Test)
	t.AddLine("Created At", charge.CreatedAt)
	t.Print()
}

func createCharge(c *cli.Context) error {
	if c.Args().Len() < 3 {
		return fmt.Errorf("You must supply charge name, price, and return URL")
	}

	price, err := decimal.NewFromString(c.Args().Get(1))
	if err != nil {
		return fmt.Errorf("Cannot create charge: invalid price %s", err)
	}

	returnURL := c.Args().Get(2)

	client := cmd.NewGraphQLClient(c)

	if c.IsSet("interval") {
		charge, err := createRecurringCharge(client, c.Args().Get(0), price.String(), c.Bool("test"), returnURL, c.String("interval"))
		if err != nil {
			return fmt.Errorf("Cannot create charge: %s", err)
		}

		if c.Bool("jsonl") {
			printChargeJSONL(charge)
		} else {
			printFormattedRecurringCharge(charge)
		}

		return nil
	}

	charge, err := createOneTimeCharge(client, c.Args().Get(0), price.String(), c.Bool("test"), returnURL)
	if err != nil {
		return fmt.Errorf("Cannot create charge: %s", err)
	}

	if c.Bool("jsonl") {
		printChargeJSONL(charge)
	} else {
		printFormattedApplicationCharge(charge)
	}

	return nil
}

// toAppSubscriptionGID returns the id as an AppSubscription GID,
// prepending the gid://shopify/AppSubscription/ prefix when given a
// bare id.
func toAppSubscriptionGID(id string) string {
	if strings.HasPrefix(id, "gid://") {
		return id
	}

	return "gid://shopify/AppSubscription/" + id
}

func cancelCharge(c *cli.Context) error {
	ids := c.Args().Slice()

	if len(ids) == 0 {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("Cannot read from stdin: %s", err)
		}

		for _, line := range strings.Split(strings.TrimRight(string(data), "\n"), "\n") {
			if line != "" {
				ids = append(ids, line)
			}
		}
	}

	if len(ids) == 0 {
		return fmt.Errorf("You must supply at least one charge id")
	}

	client := cmd.NewGraphQLClient(c)
	prorate := c.Bool("prorate")

	for _, id := range ids {
		chargeID, status, err := CancelRecurringCharge(client, toAppSubscriptionGID(id), prorate)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error cancelling charge %s: %s\n", id, err)
			continue
		}

		fmt.Printf("Cancelled charge %d, status %s\n", chargeID, status)
	}

	return nil
}

func listOneTimeCharges(c *cli.Context, gids []string) error {
	var charges []OneTimeCharge

	client := cmd.NewGraphQLClient(c)

	if len(gids) > 0 {
		byID, err := getOneTimeChargesByID(client, gids)
		if err != nil {
			return err
		}

		charges = byID
	} else {
		var err error
		charges, err = listOneTimeChargesGQL(client)
		if err != nil {
			return fmt.Errorf("Cannot list one-time charges: %s", err)
		}
	}

	if c.Bool("jsonl") {
		for _, charge := range charges {
			printChargeJSONL(charge)
		}
	} else {
		printFormattedApplicationCharges(charges)
	}

	return nil
}

func listRecurringCharges(c *cli.Context, gids []string) error {
	var charges []RecurringCharge

	client := cmd.NewGraphQLClient(c)

	if len(gids) > 0 {
		byID, err := getRecurringChargesByID(client, gids)
		if err != nil {
			return err
		}

		charges = byID
	} else {
		var err error
		charges, err = listRecurringChargesGQL(client)
		if err != nil {
			return fmt.Errorf("Cannot list recurring charges: %s", err)
		}
	}

	if c.Bool("jsonl") {
		for _, charge := range charges {
			printChargeJSONL(charge)
		}
	} else {
		printFormattedRecurringCharges(charges)
	}

	return nil
}

// classifyChargeIDs returns the one-time and recurring charge GIDs for
// the given ids. GIDs determine their own type; bare ids are treated as
// one-time charges unless recurring is set.
func classifyChargeIDs(ids []string, recurring bool) ([]string, []string, error) {
	var oneTimeGIDs, recurringGIDs []string

	for _, id := range ids {
		switch {
		case strings.HasPrefix(id, "gid://shopify/AppPurchaseOneTime/"):
			oneTimeGIDs = append(oneTimeGIDs, id)
		case strings.HasPrefix(id, "gid://shopify/AppSubscription/"):
			recurringGIDs = append(recurringGIDs, id)
		case strings.HasPrefix(id, "gid://"):
			return nil, nil, fmt.Errorf("Charge id '%s' invalid: unknown charge type", id)
		case recurring:
			recurringGIDs = append(recurringGIDs, toAppSubscriptionGID(id))
		default:
			oneTimeGIDs = append(oneTimeGIDs, toAppPurchaseOneTimeGID(id))
		}
	}

	return oneTimeGIDs, recurringGIDs, nil
}

// toAppPurchaseOneTimeGID returns the id as an AppPurchaseOneTime GID,
// prepending the gid://shopify/AppPurchaseOneTime/ prefix when given a
// bare id.
func toAppPurchaseOneTimeGID(id string) string {
	if strings.HasPrefix(id, "gid://") {
		return id
	}

	return "gid://shopify/AppPurchaseOneTime/" + id
}

func listCharges(c *cli.Context) error {
	if c.Bool("one-time") && c.Bool("recurring") {
		return fmt.Errorf("--one-time and --recurring cannot be used together")
	}

	var ids []string
	for i := 0; i < c.NArg(); i++ {
		ids = append(ids, c.Args().Get(i))
	}

	oneTimeGIDs, recurringGIDs, err := classifyChargeIDs(ids, c.Bool("recurring"))
	if err != nil {
		return err
	}

	if len(oneTimeGIDs) == 0 && len(recurringGIDs) == 0 {
		if c.Bool("recurring") {
			return listRecurringCharges(c, nil)
		}

		return listOneTimeCharges(c, nil)
	}

	if len(oneTimeGIDs) > 0 {
		if err := listOneTimeCharges(c, oneTimeGIDs); err != nil {
			return err
		}
	}

	if len(recurringGIDs) > 0 {
		if err := listRecurringCharges(c, recurringGIDs); err != nil {
			return err
		}
	}

	return nil
}

func init() {
	listFlags := []cli.Flag{
		&cli.BoolFlag{
			Name:    "jsonl",
			Aliases: []string{"j"},
			Usage:   "Output the charges in JSONL format",
		},
		&cli.BoolFlag{
			Name:    "one-time",
			Aliases: []string{"1"},
			Usage:   "List one-time charges (the default)",
		},
		&cli.BoolFlag{
			Name:    "recurring",
			Aliases: []string{"r"},
			Usage:   "List recurring charges",
		},
	}

	createFlags := []cli.Flag{
		&cli.StringFlag{
			Name:    "interval",
			Aliases: []string{"i"},
			Value:   "30d",
			Usage:   "Billing interval for a recurring charge (passing this creates a recurring charge): 30d, 1y, or 365d (default: 30d)",
		},
		&cli.BoolFlag{
			Name:    "test",
			Aliases: []string{"t"},
			Usage:   "Make the charge a test charge",
		},
		// lib does not support
		// &cli.StringFlag{
		// 	Name:    "currency",
		// 	Aliases: []string{"c"},
		// 	Usage:   "Currency to use",
		// },
	}

	Cmd = cli.Command{
		Name:    "charges",
		Aliases: []string{"c", "ch"},
		Usage:   "Do things with charges",
		Subcommands: []*cli.Command{
			{
				Name:      "ls",
				Aliases:   []string{"l"},
				Usage:     "List the shop's charges or the charges given by the specified IDs (bare ids are one time charges unless -r given)",
				ArgsUsage: "[ID [ID ...]]",
				Flags:     append(cmd.Flags, listFlags...),
				Action:    listCharges,
			},
			{
				Name:      "create",
				Aliases:   []string{"c"},
				Usage:     "Create a charge (one-time by default; use -i to create a recurring charge)",
				ArgsUsage: "NAME PRICE RETURN-URL",
				Flags:     append(cmd.Flags, createFlags...),
				Action:    createCharge,
			},
			{
				Name:        "cancel",
				Usage:       "Cancel recurring charges (app subscriptions) by ID",
				ArgsUsage:   "[ID [ID ...]]",
				Description: "If IDs are not given they're read from stdin one per line. One-time charges cannot be cancelled.",
				Flags: append(cmd.Flags,
					&cli.BoolFlag{
						Name:  "prorate",
						Usage: "Issue prorated credits for the unused portion of the subscription",
					},
				),
				Action: cancelCharge,
			},
		},
	}
}
