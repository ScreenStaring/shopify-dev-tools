package customers

import (
	"fmt"
	"strings"

	"github.com/urfave/cli/v2"
	"github.com/ScreenStaring/shopify-dev-tools/cmd"
	"github.com/ScreenStaring/shopify-dev-tools/cmd/customers/gql"
	"github.com/cheynewallace/tabby"
)

var Cmd cli.Command

func segmentsListAction(c *cli.Context) error {
	shop := c.String("shop")
	token := cmd.LookupAccessToken(shop, c.String("access-token"))

	if c.Args().Len() > 0 {
		id := c.Args().Get(0)
		segment, err := gql.GetSegment(shop, token, id)
		if err != nil {
			return err
		}

		printSegments([]gql.Segment{*segment})
		return nil
	}

	segments, err := gql.ListSegments(shop, token, c.Int("limit"))
	if err != nil {
		return err
	}

	if len(segments) == 0 {
		fmt.Println("No segments")
		return nil
	}

	printSegments(segments)

	return nil
}

func segmentsDeleteAction(c *cli.Context) error {
	if c.Args().Len() == 0 {
		return fmt.Errorf("You must supply a segment id")
	}

	id := c.Args().Get(0)
	shop := c.String("shop")
	deletedID, err := gql.DeleteSegment(shop, cmd.LookupAccessToken(shop, c.String("access-token")), id)
	if err != nil {
		return err
	}

	fmt.Printf("Segment %s deleted\n", deletedID)

	return nil
}

func printSegments(segments []gql.Segment) {
	t := tabby.New()
	for _, s := range segments {
		t.AddLine("ID", strings.TrimPrefix(s.ID, "gid://shopify/Segment/"))
		t.AddLine("Name", s.Name)
		t.AddLine("Query", s.Query)
		t.AddLine("Created", s.CreationDate)
		t.AddLine("Last Edited", s.LastEditDate)
		t.Print()

		cmd.PrintSeparator()
	}
}

func init() {
	segmentsFlags := []cli.Flag{
		&cli.IntFlag{
			Name:    "limit",
			Aliases: []string{"l"},
			Usage:   "Maximum number of segments to return, must be <= 250",
			Value:   10,
		},
	}

	Cmd = cli.Command{
		Name:    "customers",
		Aliases: []string{"cust"},
		Usage:   "Do things with customers",

		Subcommands: []*cli.Command{
			{
				Name:    "segments",
				Aliases: []string{"seg"},
				Usage:   "Customer segment commands",
				Subcommands: []*cli.Command{
					{
						Name:      "ls",
						ArgsUsage: "[ID]",
						Usage:     "List the shop's segments or a segment given by ID",
						Flags:     append(cmd.Flags, segmentsFlags...),
						Action:    segmentsListAction,
					},
					{
						Name:      "delete",
						Aliases:   []string{"del", "rm", "d"},
						ArgsUsage: "ID",
						Usage:     "Delete the given segment",
						Flags:     cmd.Flags,
						Action:    segmentsDeleteAction,
					},
				},
			},
		},
	}
}
