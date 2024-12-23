package gql

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/ScreenStaring/shopify-dev-tools/cmd"
	"github.com/ScreenStaring/shopify-dev-tools/gql"
	"github.com/urfave/cli/v2"
)

var Cmd cli.Command

func findQuery(c *cli.Context) (string, error) {
	if c.NArg() == 0 {
		query, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("Cannot read from stdin: %s", err)
		}

		// Ensure ^D doesn't show in output when reading from stdin
		fmt.Print("\n")

		return string(query), nil
	}

	file := c.Args().Get(0)

	query, err := ioutil.ReadFile(file)
	if err != nil {
		return "", fmt.Errorf("Cannot read file %s: %s", file, err)
	}

	return string(query), nil
}

func queryAction(c *cli.Context) error {
	shop := c.String("shop")
	client := gql.NewClient(shop, cmd.LookupAccessToken(shop, c.String("access-token")), c.String("api-version"))

	query, err := findQuery(c)
	if err != nil {
		return err
	}

	result, err := client.Query(query)
	if err != nil {
		return err
	}

	err = result.JsonIndentWriter(os.Stdout, "", " ")
	if err != nil {
		return fmt.Errorf("Cannot serialize GraphQL JSON response: %s", err)
	}

	return nil
}

func init() {
	flags := []cli.Flag{
		&cli.StringFlag{
			Name:    "api-version",
			Aliases: []string{"a"},
			Usage:   "API version to use; default is a versionless call",
		},
	}

	Cmd = cli.Command{
		Name:        "graphql",
		Aliases:     []string{"gql"},
		ArgsUsage:   "[query-file.graphql]",
		Usage:       "Run a GraphQL query against the Admin API",
		Description: "If query-file.graphql is not given query is read from stdin",
		Flags:       append(cmd.Flags, flags...),
		Action:      queryAction,
	}
}
