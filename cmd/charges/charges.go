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

type listChargesOptions struct {
	Ids    []int64   `url:"ids,comma,omitempty"`
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

		fmt.Printf("%s\n", strings.Repeat("-", 20))
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

	fmt.Printf("%s\n", strings.Repeat("-", 20))
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

func listCharges(c *cli.Context) error {
	var options listChargesOptions

	if c.NArg() > 0 {
		for i := 0; i < c.NArg(); i++ {
			id, err := strconv.ParseInt(c.Args().Get(i), 10, 64)
			if err != nil {
				return fmt.Errorf("Charge id '%s' invalid: must be an int", c.Args().Get(0))
			}

			options.Ids = append(options.Ids, id)
		}

	}

	charges, err := cmd.NewShopifyClient(c).RecurringApplicationCharge.List(options)
	if err != nil {
		return fmt.Errorf("Cannot list charges: %s", err)
	}

	if c.Bool("jsonl") {
		printJSONL(charges)
	} else {
		printFormattedRecurringCharges(charges)
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
				Usage:     "List the shop's recurring charges or the recurring charges given by the specified IDs",
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
