package scripttags;

import (
	"fmt"
	"strings"

	"github.com/urfave/cli/v2"
	shopify "github.com/bold-commerce/go-shopify/v3"
	"github.com/cheynewallace/tabby"

	"github.com/ScreenStaring/shopify-dev-tools/cmd"
)

var Cmd cli.Command

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
				Name: "ls",
				Flags: append(cmd.Flags),
				Action: listAction,
				Usage: "List scripttags for the given shop",
			},
		},
	}
}
