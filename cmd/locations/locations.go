package locations

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cheynewallace/tabby"
	"github.com/urfave/cli/v2"

	"github.com/ScreenStaring/shopify-dev-tools/cmd"
)

var Cmd cli.Command

// toLocationGID returns the id as a Location GID, prepending the
// gid://shopify/Location/ prefix when given a bare id.
func toLocationGID(id string) string {
	if strings.HasPrefix(id, "gid://") {
		return id
	}

	return "gid://shopify/Location/" + id
}

func listLocations(c *cli.Context) error {
	client := cmd.NewGraphQLClient(c)

	var locations []Location
	var err error

	if c.NArg() > 0 {
		ids := make([]string, 0, c.NArg())
		for i := 0; i < c.NArg(); i++ {
			ids = append(ids, toLocationGID(c.Args().Get(i)))
		}

		locations, err = LocationsByID(client, ids)
	} else {
		locations, err = ListLocations(client)
	}

	if err != nil {
		return err
	}

	if len(locations) == 0 {
		if !c.Bool("jsonl") {
			fmt.Println("No locations")
		}
		return nil
	}

	if c.Bool("jsonl") {
		printLocationsJSONL(locations)
	} else {
		printLocations(locations)
	}

	return nil
}

func printLocationsJSONL(locations []Location) {
	for _, loc := range locations {
		line, err := json.Marshal(loc)
		if err != nil {
			panic(err)
		}

		fmt.Println(string(line))
	}
}

func printLocations(locations []Location) {
	t := tabby.New()
	for _, loc := range locations {
		t.AddLine("ID", loc.ID)
		t.AddLine("Name", loc.Name)
		t.AddLine("Active", loc.Active)
		t.AddLine("Fulfillment Service", loc.FulfillmentService)
		t.AddLine("Address", loc.Address)
		t.AddLine("Created At", loc.CreatedAt)
		t.AddLine("Updated At", loc.UpdatedAt)
		t.Print()

		cmd.PrintSeparator()
	}
}

func init() {
	listFlags := []cli.Flag{
		&cli.BoolFlag{
			Name:    "jsonl",
			Aliases: []string{"j"},
			Usage:   "Output the locations in JSONL format",
		},
	}

	Cmd = cli.Command{
		Name:    "locations",
		Aliases: []string{"loc"},
		Usage:   "Do things with locations",
		Subcommands: []*cli.Command{
			{
				Name:      "ls",
				Aliases:   []string{"l"},
				Usage:     "List the shop's locations or the locations given by the specified IDs",
				ArgsUsage: "[ID [ID ...]]",
				Flags:     append(cmd.Flags, listFlags...),
				Action:    listLocations,
			},
		},
	}
}
