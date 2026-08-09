package charges

import (
	"encoding/json"
	"fmt"
	"strconv"
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

func printFormattedRecurringCharges(charges []RecurringCharge) {
	t := tabby.New()

	for _, charge := range charges {
		t.AddLine("Id", charge.ID)
		t.AddLine("Name", charge.Name)
		t.AddLine("Price", charge.Price)
		t.AddLine("Status", charge.Status)
		t.AddLine("Return URL", charge.ReturnURL)
		t.AddLine("Test", charge.Test)
		t.AddLine("Created At", charge.CreatedAt)
		t.Print()

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
	if c.Args().Len() < 2 {
		return fmt.Errorf("You must supply charge name and price")
	}

	price, err := decimal.NewFromString(c.Args().Get(1))
	if err != nil {
		return fmt.Errorf("Cannot create charge: invalid price %s", err)
	}

	charge, err := createOneTimeCharge(cmd.NewGraphQLClient(c), c.Args().Get(0), price.String(), c.Bool("test"), c.String("return-to"))
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

func listOneTimeCharges(c *cli.Context, ids []int64) error {
	var charges []OneTimeCharge

	client := cmd.NewGraphQLClient(c)

	if len(ids) > 0 {
		byID, err := getOneTimeChargesByID(client, ids)
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

func listRecurringCharges(c *cli.Context, ids []int64) error {
	var charges []RecurringCharge

	client := cmd.NewGraphQLClient(c)

	if len(ids) > 0 {
		byID, err := getRecurringChargesByID(client, ids)
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

func listCharges(c *cli.Context) error {
	var ids []int64

	if c.NArg() > 0 {
		for i := 0; i < c.NArg(); i++ {
			id, err := strconv.ParseInt(c.Args().Get(i), 10, 64)
			if err != nil {
				return fmt.Errorf("Charge id '%s' invalid: must be an int", c.Args().Get(i))
			}

			ids = append(ids, id)
		}
	}

	if c.Bool("one-time") {
		return listOneTimeCharges(c, ids)
	}

	return listRecurringCharges(c, ids)
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
			Usage:   "List one-time charges (default is recurring)",
		},
	}

	createFlags := []cli.Flag{
		&cli.StringFlag{
			Name:    "return-to",
			Aliases: []string{"r"},
			Usage:   "URL to redirect user to after charge is accepted",
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
				Usage:     "List the shop's charges or the charges given by the specified IDs",
				ArgsUsage: "[ID [ID ...]]",
				Flags:     append(cmd.Flags, listFlags...),
				Action:    listCharges,
			},
			{
				Name:      "create",
				Aliases:   []string{"c"},
				Usage:     "Create a one-time charge (Application Charge)",
				ArgsUsage: "NAME PRICE",
				Flags:     append(cmd.Flags, createFlags...),
				Action:    createCharge,
			},
		},
	}
}
