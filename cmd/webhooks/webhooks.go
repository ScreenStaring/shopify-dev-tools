package webhooks

import (
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/urfave/cli/v2"
	"github.com/cheynewallace/tabby"

	"github.com/ScreenStaring/shopify-dev-tools/cmd"
)

var Cmd cli.Command
var webhookName = regexp.MustCompile(`(?i)\A[_a-zA-Z]+/[_a-zA-Z]+\z`)

func format(c *cli.Context) string {
	if c.Bool("xml") {
		return "XML"
	}

	return "JSON"
}

func printFormatted(webhooks []Webhook)  {
	t := tabby.New()
	for _, webhook := range webhooks {
		t.AddLine("Id", webhook.ID)
		t.AddLine("Address", webhook.Endpoint)
		t.AddLine("Topic", webhook.Topic)
		t.AddLine("Fields", webhook.Fields)
		t.AddLine("Metafield Namespaces", webhook.MetafieldNamespaces)
		t.AddLine("API Version", webhook.ApiVersion)
		t.AddLine("Created", webhook.CreatedAt)
		t.AddLine("Updated", webhook.UpdatedAt)
		t.Print()

		cmd.PrintSeparator()
	}
}

func printJSONL(webhooks []Webhook)  {
	for _, webhook := range webhooks {
		line, err := json.Marshal(webhook)
		if err != nil {
			panic(err)
		}

		fmt.Println(string(line))
	}
}

func createAction(c *cli.Context) error {
	shop := c.String("shop")
	token := cmd.LookupAccessToken(shop, c.String("access-token"))

	id, err := createWebhook(shop, token, c.String("topic"), c.String("address"), format(c), c.StringSlice("fields"))
	if err != nil {
		return err
	}

	fmt.Printf("Webhook created: %s\n", id)
	return nil
}

func deleteAction(c *cli.Context) error {
	shop := c.String("shop")
	token := cmd.LookupAccessToken(shop, c.String("access-token"))

	var webhooks []Webhook

	if c.Bool("all") {
		var err error
		webhooks, err = listWebhooks(shop, token, nil)
		if err != nil {
			return err
		}
	} else {
		if c.Args().Len() == 0 {
			return fmt.Errorf("You must supply a webhook id or topic")
		}

		for _, arg := range c.Args().Slice() {
			if webhookName.MatchString(arg) {
				found, err := listWebhooks(shop, token, []string{arg})
				if err != nil {
					return fmt.Errorf("Cannot list webhooks for topic %s: %s", arg, err)
				}
				webhooks = append(webhooks, found...)
			} else {
				webhooks = append(webhooks, Webhook{GID: webhookGID(arg)})
			}
		}
	}

	if len(webhooks) == 0 {
		return fmt.Errorf("No webhooks found")
	}

	for _, w := range webhooks {
		err := deleteWebhook(shop, token, w.GID)
		if err != nil {
			return err
		}
	}

	fmt.Printf("%d webhook(s) deleted\n", len(webhooks))

	return nil
}

func listAction(c *cli.Context) error {
	shop := c.String("shop")
	token := cmd.LookupAccessToken(shop, c.String("access-token"))

	hooks, err := listWebhooks(shop, token, nil)
	if err != nil {
		return err
	}

	if c.Bool("jsonl") {
		printJSONL(hooks)
	} else {
		printFormatted(hooks)
	}

	return nil
}

func init() {
	createFlags := []cli.Flag{
		&cli.StringFlag{
			Name: "address",
			Required: true,
			Aliases: []string{"a"},
		},
		&cli.StringSliceFlag{
			Name: "fields",
			Aliases: []string{"f"},
		},
		&cli.BoolFlag{
			Name: "xml",
			Value: false,
		},
		&cli.StringFlag{
			Name: "topic",
			Required: true,
			Aliases: []string{"t"},
		},
	}

	deleteFlags := []cli.Flag{
		&cli.BoolFlag{
			Name: "all",
			Aliases: []string{"a"},
		},
	}

	listFlags := []cli.Flag{
		&cli.BoolFlag{
			Name: "jsonl",
			Aliases: []string{"j"},
		},
	}

	Cmd = cli.Command{
		Name:  "webhook",
		Aliases: []string{"webhooks", "hooks", "w"},
		Usage:   "Webhook utilities",
		Subcommands: []*cli.Command{
			{
				Name: "create",
				Aliases: []string{"c"},
				Flags: append(cmd.Flags, createFlags...),
				Action: createAction,
				Usage: "Create a webhook for the given shop",
			},
			{
				Name: "delete",
				ArgsUsage: "[topic or webhook ID]",
				Aliases: []string{"del", "rm", "d"},
				Flags: append(cmd.Flags, deleteFlags...),
				Action: deleteAction,
				Usage: "Delete the given webhook",
			},
			{
				Name: "ls",
				Flags: append(cmd.Flags, listFlags...),
				Action: listAction,
				Usage: "List the shop's webhooks",
			},
		},
	}
}
