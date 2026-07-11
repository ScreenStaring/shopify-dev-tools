package metaobjects

import (
	"errors"
	"fmt"
	"strings"

	"github.com/ScreenStaring/shopify-dev-tools/cmd"
	"github.com/ScreenStaring/shopify-dev-tools/cmd/metaobjects/gql"
	"github.com/cheynewallace/tabby"
	"github.com/urfave/cli/v2"
)

var Cmd cli.Command

func listAction(c *cli.Context) error {
	if c.NArg() == 0 {
		return errors.New("Metaobject type required")
	}

	shop := c.String("shop")
	token := cmd.LookupAccessToken(shop, c.String("access-token"))

	metaobjects, err := gql.ListMetaobjects(shop, token, c.Args().Get(0), c.Int("limit"), c.Int("page"), c.String("query"), c.Bool("verbose"))
	if err != nil {
		return err
	}

	if len(metaobjects) == 0 {
		fmt.Println("No metaobjects")
		return nil
	}

	printMetaobjects(metaobjects)
	return nil
}

func printMetaobjects(metaobjects []gql.Metaobject) {
	t := tabby.New()
	for _, m := range metaobjects {
		t.AddLine("ID", strings.TrimPrefix(m.ID, "gid://shopify/Metaobject/"))
		t.AddLine("Handle", m.Handle)
		t.AddLine("Type", m.Type)
		t.AddLine("Display Name", m.DisplayName)
		t.AddLine("Updated", m.UpdatedAt)
		for _, f := range m.Fields {
			t.AddLine(f.Key, f.Value)
		}
		t.Print()

		cmd.PrintSeparator()
	}
}

func defListAction(c *cli.Context) error {
	shop := c.String("shop")
	token := cmd.LookupAccessToken(shop, c.String("access-token"))

	if c.NArg() > 0 {
		var definitions []gql.MetaobjectDefinition
		for i := 0; i < c.NArg(); i++ {
			d, err := gql.GetMetaobjectDefinition(shop, token, c.Args().Get(i), c.Bool("verbose"))
			if err != nil {
				return err
			}
			definitions = append(definitions, *d)
		}

		printMetaobjectDefinitions(definitions)
		return nil
	}

	definitions, err := gql.ListMetaobjectDefinitions(shop, token, c.Int("limit"), c.Int("page"), c.Bool("verbose"))
	if err != nil {
		return err
	}

	if len(definitions) == 0 {
		fmt.Println("No metaobject definitions")
		return nil
	}

	printMetaobjectDefinitions(definitions)
	return nil
}

func printMetaobjectDefinitions(definitions []gql.MetaobjectDefinition) {
	t := tabby.New()
	for _, d := range definitions {
		t.AddLine("ID", strings.TrimPrefix(d.ID, "gid://shopify/MetaobjectDefinition/"))
		t.AddLine("Name", d.Name)
		t.AddLine("Type", d.Type)
		t.AddLine("Display Name Key", d.DisplayNameKey)
		t.Print()

		fmt.Println("Fields")
		printFieldDefinitions(d.Fields)
		fmt.Print("\n")

		cmd.PrintSeparator()
	}
}

func printFieldDefinitions(fields []gql.MetaobjectFieldDefinition) {
	t := tabby.New()
	t.AddHeader("Key", "Name", "Type", "Validations")

	for _, f := range fields {
		t.AddLine(f.Key, f.Name, f.Type, formatValidations(f.Validations))
	}

	t.Print()
}

func formatValidations(validations []gql.MetaobjectFieldValidation) string {
	parts := make([]string, len(validations))
	for i, v := range validations {
		parts[i] = fmt.Sprintf("%s:%s", v.Name, v.Value)
	}

	return strings.Join(parts, ", ")
}

func init() {
	limitFlag := &cli.IntFlag{
		Name:    "limit",
		Aliases: []string{"l"},
		Usage:   "Maximum number of results to return, must be <= 250",
		Value:   10,
	}

	pageFlag := &cli.IntFlag{
		Name:    "page",
		Aliases: []string{"p"},
		Usage:   "Page of results to return, pages are limit sized",
		Value:   1,
	}

	listFlags := []cli.Flag{
		limitFlag,
		pageFlag,
		&cli.StringFlag{
			Name:    "query",
			Aliases: []string{"q"},
			Usage:   "Filter metaobjects using Shopify's search syntax, e.g. fields.key:value",
		},
	}

	defListFlags := []cli.Flag{
		limitFlag,
		pageFlag,
	}

	exportFlags := []cli.Flag{
		&cli.StringFlag{
			Name:    "query",
			Aliases: []string{"q"},
			Usage:   "Filter metaobjects using Shopify's search syntax, e.g. fields.key:value",
		},
		&cli.BoolFlag{
			Name:    "jsonl",
			Aliases: []string{"j"},
			Usage:   "Export as JSONL (one record per line) instead of CSV",
		},
	}

	Cmd = cli.Command{
		Name:    "metaobjects",
		Aliases: []string{"mo"},
		Usage:   "Do things with metaobjects",

		Subcommands: []*cli.Command{
			{
				Name:      "ls",
				Aliases:   []string{"l"},
				ArgsUsage: "TYPE",
				Usage:     "List metaobjects of the given type",
				Flags:     append(cmd.Flags, listFlags...),
				Action:    listAction,
			},
			{
				Name:      "export",
				Aliases:   []string{"x"},
				ArgsUsage: "TYPE",
				Usage:     "Export metaobjects of the given type to CSV or JSONL",
				Flags:     append(cmd.Flags, exportFlags...),
				Action:    exportAction,
			},
			{
				Name:    "def",
				Aliases: []string{"d"},
				Usage:   "Metaobject definition utilities",
				Subcommands: []*cli.Command{
					{
						Name:      "ls",
						Aliases:   []string{"l"},
						ArgsUsage: "[ID ...]",
						Usage:     "List metaobject definitions or the definitions given by ID",
						Flags:     append(cmd.Flags, defListFlags...),
						Action:    defListAction,
					},
				},
			},
		},
	}
}
