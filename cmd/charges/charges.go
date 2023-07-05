package charges

import (
	"encoding/json"

	"fmt"
	"strconv"
	"strings"

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
		line, err := json.Marshal(charge)
		if err != nil {
			panic(err)
		}

		fmt.Println(string(line))
	}
}

func printFormatted(charges []shopify.RecurringApplicationCharge) {
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
		printFormatted(charges)
	}

	return nil
}

func init() {
	chargeFlags := []cli.Flag{
		&cli.BoolFlag{
			Name:    "jsonl",
			Aliases: []string{"j"},
			Usage:   "Output the charges in JSONL format",
		},

	}

	Cmd = cli.Command{
		Name:    "charges",
		Usage:   "Do things with charges (only recurring for now)",
		Subcommands: []*cli.Command{
			{
				Name:      "ls",
				Aliases:   []string{"l"},
				Usage:     "List the shop's recurring charges or the recurring charges given by the specified IDs",
				ArgsUsage: "[ID [ID ...]]",
				Flags:     append(cmd.Flags, chargeFlags...),
				Action:    listCharges,
			},
		},
	}
}
