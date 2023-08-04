package charges

import (
	"encoding/json"

	"fmt"
	"strconv"
	"strings"

	"github.com/shopspring/decimal"
	shopify "github.com/bold-commerce/go-shopify/v3"
	"github.com/cheynewallace/tabby"
	"github.com/urfave/cli/v2"

	"github.com/ScreenStaring/shopify-dev-tools/cmd"
)

var Cmd cli.Command

func printRecordSeperator() {
	fmt.Printf("%s\n", strings.Repeat("-", 20))
}

func printJSONL(charges []shopify.RecurringApplicationCharge) {
	for _, charge := range charges {
		printChargeJSONL(charge);
	}
}

func printChargeJSONL(charge interface{}) {
	line, err := json.Marshal(charge)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(line))
}

func printFormattedRecurringCharges(charges []shopify.RecurringApplicationCharge) {
	t := tabby.New()

	for _, charge := range charges {
		t.AddLine("Id", charge.ID)
		t.AddLine("Name", charge.Name)
		t.AddLine("Price", charge.Price)
		t.AddLine("Status", charge.Status)
		t.AddLine("Activated On", charge.ActivatedOn)
		t.AddLine("Confirmation URL", charge.ConfirmationURL)
		t.AddLine("Return URL", charge.DecoratedReturnURL)
		t.AddLine("Test", *charge.Test)
		t.AddLine("Created At", charge.CreatedAt)
		t.AddLine("Updated At", charge.UpdatedAt)
		t.Print()

		printRecordSeperator()
	}
}

func printFormattedApplicationCharges(charges []shopify.ApplicationCharge) {
	for _, charge := range charges {
		printFormattedApplicationCharge(&charge)
		printRecordSeperator()
	}
}

func printFormattedApplicationCharge(charge *shopify.ApplicationCharge) {
	t := tabby.New()

	t.AddLine("Id", charge.ID)
	t.AddLine("Name", charge.Name)
	t.AddLine("Price", charge.Price)
	t.AddLine("Status", charge.Status)
	t.AddLine("Confirmation URL", charge.ConfirmationURL)
	t.AddLine("Return URL", charge.DecoratedReturnURL)
	t.AddLine("Test", *charge.Test)
	t.AddLine("Created At", charge.CreatedAt)
	t.AddLine("Updated At", charge.UpdatedAt)
	t.Print()
}


func createCharge(c *cli.Context) error {
	var charge shopify.ApplicationCharge

	if(c.Args().Len() < 2) {
		return fmt.Errorf("You must supply charge name and price")
	}

	price, err := decimal.NewFromString(c.Args().Get(1))
	if err != nil {
		return fmt.Errorf("Cannot create charge: invalid price %s", err)
	}

	charge.Price = &price
	charge.Name = c.Args().Get(0)

	test := c.Bool("test")
	charge.Test = &test

	returnURL := c.String("return-to")
	if len(returnURL) > 0 {
		charge.ReturnURL = returnURL
	}

	result, err := cmd.NewShopifyClient(c).ApplicationCharge.Create(charge)
	if err != nil {
		return fmt.Errorf("Cannot create charge: %s", err)
	}

	if(c.Bool("jsonl")) {
		printChargeJSONL(result)
	} else {
		printFormattedApplicationCharge(result)
	}

	return nil
}

func listOneTimeCharges(c *cli.Context, ids []int64) error {
	var err error
	var charges []shopify.ApplicationCharge

	client := cmd.NewShopifyClient(c)

	if(len(ids) > 0) {
		for _, id := range ids {
			charge, err := client.ApplicationCharge.Get(id, nil)
			if err != nil {
				return fmt.Errorf("Cannot get one-time charge %d: %s", id, err)
			}

			charges = append(charges, *charge)
		}
	} else {
		charges, err = client.ApplicationCharge.List(nil)
		if err != nil {
			return fmt.Errorf("Cannot list one-time charges: %s", err)
		}
	}

	if c.Bool("jsonl") {
		for _, charge := range charges {
			printChargeJSONL(charge);
		}
	} else {
		printFormattedApplicationCharges(charges)
	}

	return nil
}

func listRecurringCharges(c *cli.Context, ids []int64) error {
	var err error
	var charges []shopify.RecurringApplicationCharge

	client := cmd.NewShopifyClient(c)

	if(len(ids) > 0) {
		for _, id := range ids {
			charge, err := client.RecurringApplicationCharge.Get(id, nil)
			if err != nil {
				return fmt.Errorf("Cannot get recurring charge %d: %s", id, err)
			}

			charges = append(charges, *charge)
		}

	} else {
		charges, err = client.RecurringApplicationCharge.List(nil)
		if err != nil {
			return fmt.Errorf("Cannot list recurring charges: %s", err)
		}
	}

	if c.Bool("jsonl") {
		for _, charge := range charges {
			printChargeJSONL(charge);
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
				return fmt.Errorf("Charge id '%s' invalid: must be an int", c.Args().Get(0))
			}

			ids = append(ids, id)
		}

	}

	if (c.Bool("one-time")) {
		return listOneTimeCharges(c, ids)
	} else {
		return listRecurringCharges(c, ids)
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
