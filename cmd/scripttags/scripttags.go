package scripttags

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/cheynewallace/tabby"
	"github.com/urfave/cli/v2"

	"github.com/ScreenStaring/shopify-dev-tools/cmd"
)

var Cmd cli.Command

// Match https://foo.com or //foo.com
// According to GQL docs this can be *any* URI:
// https://shopify.dev/api/admin-graphql/2022-01/mutations/scriptTagCreate
var scriptTagURL = regexp.MustCompile(`(?i)\A(?:https:)?//[\da-z]`)

func deleteAction(c *cli.Context) error {
	if c.Args().Len() == 0 {
		return fmt.Errorf("You must supply an script tag id or URL")
	}

	var ids []string

	client := cmd.NewGraphQLClient(c)

	if scriptTagURL.MatchString(c.Args().Get(0)) {
		src := c.Args().Get(0)
		tags, err := listScriptTags(client, src)

		if err != nil {
			return fmt.Errorf("Cannot list script tag with URL %s: %s", src, err)
		}

		if len(tags) == 0 {
			return fmt.Errorf("Cannot find script tag with URL %s", src)
		}

		for _, tag := range tags {
			ids = append(ids, tag.ID)
		}
	} else {
		id, err := strconv.ParseInt(c.Args().Get(0), 10, 64)
		if err != nil {
			return fmt.Errorf("Script tag id '%s' is invalid: must be an int", c.Args().Get(0))
		}

		ids = append(ids, scriptTagGID(strconv.FormatInt(id, 10)))
	}

	for _, id := range ids {
		err := deleteScriptTag(client, id)
		if err != nil {
			return err
		}

		fmt.Printf("Script tag %s deleted\n", id)
	}

	return nil
}

func listAction(c *cli.Context) error {
	tags, err := listScriptTags(cmd.NewGraphQLClient(c), "")
	if err != nil {
		return fmt.Errorf("Cannot list ScriptTags: %s", err)
	}

	if c.Bool("jsonl") {
		printJSONL(tags)
	} else {
		printFormatted(tags)
	}

	return nil
}

func printJSONL(tags []ScriptTag) {
	for _, tag := range tags {
		line, err := json.Marshal(tag)
		if err != nil {
			panic(err)
		}

		fmt.Println(string(line))
	}
}

func printFormatted(tags []ScriptTag) {
	t := tabby.New()
	for _, tag := range tags {
		t.AddLine("Id", tag.LegacyResourceID)
		t.AddLine("Gid", tag.ID)
		t.AddLine("Src", tag.Src)
		t.AddLine("Cache", tag.Cache)
		t.AddLine("Display Scope", tag.DisplayScope)
		t.AddLine("Created", tag.CreatedAt)
		t.AddLine("Updated", tag.UpdatedAt)
		t.Print()

		fmt.Printf("%s\n", strings.Repeat("-", 20))
	}
}

func init() {
	apiVersionFlag := &cli.StringFlag{
		Name:    "api-version",
		Aliases: []string{"a"},
		Usage:   "API version to use; default is a versionless call",
	}

	Cmd = cli.Command{
		Name:  "scripttags",
		Usage: "ScriptTag utilities",
		Subcommands: []*cli.Command{
			{
				Name:    "delete",
				Aliases: []string{"del", "rm", "d"},
				Flags:   append(cmd.Flags, apiVersionFlag),
				Action:  deleteAction,
				Usage:   "Delete the given ScriptTag",
			},
			{
				Name:    "list",
				Aliases: []string{"ls"},
				Flags: append(cmd.Flags, &cli.BoolFlag{
					Name:    "jsonl",
					Aliases: []string{"j"},
					Usage:   "Output the script tags in JSONL format",
				}, apiVersionFlag),
				Action: listAction,
				Usage:  "List scripttags for the given shop",
			},
		},
	}
}
