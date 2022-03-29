package scripttags;

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/urfave/cli/v2"
	shopify "github.com/bold-commerce/go-shopify/v3"
	"github.com/cheynewallace/tabby"

	"github.com/ScreenStaring/shopify-dev-tools/cmd"
)

type listScriptTagOptions struct {
	Src string `url:"src"`
}

var Cmd cli.Command
// Allow for protocol relative URLs
var scriptTagURL = regexp.MustCompile(`(?i)\A(?:https:)?//[\da-z]`)

func deleteAction(c *cli.Context) error {
	if(c.Args().Len() == 0) {
		return fmt.Errorf("You must supply an script tag id or URL")
	}

	var ids[] int64
	var err error

	client := cmd.NewShopifyClient(c)

	if(scriptTagURL.MatchString(c.Args().Get(0))) {
		options := listScriptTagOptions{Src: c.Args().Get(0)}
		tags, err := client.ScriptTag.List(options)

		if err != nil {
			return fmt.Errorf("Cannot list script tag with URL %s: %s", options.Src, err)
		}

		if len(tags) == 0 {
			return fmt.Errorf("Cannot find script tag with URL %s", options.Src)
		}

		// Delete all with givv
		for _, tag := range tags {
			ids = append(ids, tag.ID)
		}
	} else {
		id, err := strconv.ParseInt(c.Args().Get(0), 10, 64)
		if err != nil {
			return fmt.Errorf("Script tag id '%s' is invalid: must be an int", c.Args().Get(0))
		}

		ids = append(ids, id)
	}

	for _, id := range ids {
		err = client.ScriptTag.Delete(id)
		if err != nil {
			return fmt.Errorf("Cannot delete script tag: %s", err)
		}

		fmt.Printf("Script tag %d deleted\n", id)
	}

	return nil
}

func listAction(c *cli.Context) error {
	hooks, err := cmd.NewShopifyClient(c).ScriptTag.List(nil)
	if err != nil {
		return fmt.Errorf("Cannot list ScriptTags: %s", err)
	}

	if c.Bool("jsonl") {
		//printJSONL(hooks)
	} else {
		printFormatted(hooks)
	}

	return nil
}

func printFormatted(webhooks []shopify.ScriptTag)  {
	t := tabby.New()
	for _, webhook := range webhooks {
		t.AddLine("Id", webhook.ID)
		t.AddLine("Src", webhook.Src)
		t.AddLine("Event", webhook.Event)
		t.AddLine("Display Scope", webhook.DisplayScope)
		t.AddLine("Created", webhook.CreatedAt)
		t.AddLine("Updated", webhook.UpdatedAt)
		t.Print()

		fmt.Printf("%s\n", strings.Repeat("-", 20))
	}
}

func init() {
	Cmd = cli.Command{
		Name: "scripttags",
		Usage: "ScriptTag utilities",
		Subcommands: []*cli.Command{
			{
				Name: "delete",
				Aliases: []string{"del", "rm", "d"},
				Flags: append(cmd.Flags),
				Action: deleteAction,
				Usage: "Delete the given ScriptTag",
			},
			{
				Name: "list",
				Aliases: []string{"ls"},
				Flags: append(cmd.Flags),
				Action: listAction,
				Usage: "List scripttags for the given shop",
			},
		},
	}
}
