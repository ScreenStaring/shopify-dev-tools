package webhooks

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/urfave/cli/v2"
	shopify "github.com/bold-commerce/go-shopify/v3"
	"github.com/cheynewallace/tabby"

	"github.com/ScreenStaring/shopify-dev-tools/cmd"
)

type webhookOptions struct {
	// sort..
	JSONL bool
}

type listWebhookOptions struct {
	Topic string `url:"topic"`
}

var Cmd cli.Command
var webhookName = regexp.MustCompile(`(?i)\A[_a-zA-Z]+/[_a-zA-Z]+\z`)

func format(c *cli.Context) string {
	if c.Bool("xml") {
		return "xml"
	}

	return "json"
}

func printFormatted(webhooks []shopify.Webhook)  {
	t := tabby.New()
	for _, webhook := range webhooks {
		t.AddLine("Id", webhook.ID)
		t.AddLine("Address", webhook.Address)
		t.AddLine("Topic", webhook.Topic)
		t.AddLine("Fields", webhook.Fields)
		// Not in shopify-go:
		//t.AddLine("api version", webhook.APIVersion)
		// ---
		// webhook.MetafieldNamespaces
		t.AddLine("Created", webhook.CreatedAt)
		t.AddLine("Updated", webhook.UpdatedAt)
		t.Print()

		fmt.Printf("%s\n", strings.Repeat("-", 20))
	}
}

func printJSONL(webhooks []shopify.Webhook)  {
	for _, webhook := range webhooks {
		line, err := json.Marshal(webhook)
		if err != nil {
			panic(err)
		}

		fmt.Println(string(line))
	}
}

func findAllWebhooks(client *shopify.Client) ([]int64, error) {
	var hookIDs []int64

	// FIXME: pagination
	webhooks, err := client.Webhook.List(nil)
	if err != nil {
		return []int64{}, fmt.Errorf("Cannot list webhooks: %s", err)
	}

	for _, webhook := range webhooks {
		hookIDs = append(hookIDs, webhook.ID)
	}

	return hookIDs, nil
}

func findGivenWebhooks(client *shopify.Client, wanted []string) ([]int64, error) {
	var hookIDs []int64

	for _, arg := range wanted {
		if webhookName.MatchString(arg) {
			options := listWebhookOptions{Topic: arg}
			webhooks, err := client.Webhook.List(options)
			if err != nil {
				return []int64{}, fmt.Errorf("Cannot list webhooks for topic %s: %s", options.Topic, err)
			}

			for _, webhook := range webhooks {
				hookIDs = append(hookIDs, webhook.ID)
			}
		} else {
			id, err := strconv.ParseInt(arg, 10, 64)
			if err != nil {
				return []int64{}, fmt.Errorf("Webhook id '%s' is invalid: must be an int", arg)
			}

			hookIDs = append(hookIDs, id)
		}
	}

	return hookIDs, nil
}

func createAction(c *cli.Context) error {
	options := shopify.Webhook{
		Address: c.String("address"),
		Topic: c.String("topic"),
		Fields: c.StringSlice("fields"),
		Format: format(c),
	}

	hook, err := cmd.NewShopifyClient(c).Webhook.Create(options)
	if err != nil {
		return fmt.Errorf("Cannot create webhook: %s", err)
	}

	fmt.Printf("Webhook created: %d\n", hook.ID)
	return nil
}

func deleteAction(c *cli.Context) error {
	var err error
	var hookIDs []int64

	client := cmd.NewShopifyClient(c)

	if(c.Bool("all")) {
		hookIDs, err = findAllWebhooks(client)
	} else {
		if(c.Args().Len() == 0) {
			return fmt.Errorf("You must supply a webhook id or topic")
		}

		hookIDs, err = findGivenWebhooks(client, c.Args().Slice())
	}

	if err != nil {
		return err
	}

	if len(hookIDs) == 0 {
		return fmt.Errorf("No webhooks found")
	}

	for _, id := range hookIDs {
		err = client.Webhook.Delete(id)
		if err != nil {
			return fmt.Errorf("Cannot delete webhook %d: %s", id, err)
		}
	}

	fmt.Printf("%d webhook(s) deleted\n", len(hookIDs))

	return nil
}

func listAction(c *cli.Context) error {
	hooks, err := cmd.NewShopifyClient(c).Webhook.List(nil)
	if err != nil {
		return fmt.Errorf("Cannot list webhooks: %s", err)
	}

	if c.Bool("jsonl") {
		printJSONL(hooks)
	} else {
		printFormatted(hooks)
	}

	return nil
}

func updateAction(c *cli.Context) error {
	if(c.Args().Len() == 0) {
		return fmt.Errorf("You must supply a webhook id to update")
	}

	id, err := strconv.ParseInt(c.Args().Get(0), 10, 64)
	if err != nil {
		return fmt.Errorf("Webhook id '%s' is invalid: must be an int", c.Args().Get(0))
	}

	options := shopify.Webhook{
		ID: id,
		Address: c.String("address"),
		Topic: c.String("topic"),
		Fields: c.StringSlice("fields"),
		Format: format(c),
	}

	_, err = cmd.NewShopifyClient(c).Webhook.Update(options)
	if err != nil {
		return fmt.Errorf("Cannot update webhook: %s", err)
	}

	fmt.Println("Webhook updated")
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
			// {
			// 	Name: "update",
			// 	Aliases: []string{"u"},
			// 	// No! FLags here are optional!
			// 	Flags: append(cmd.Flags, createFlags...),
			// 	Action: updateAction,
			// 	Usage: "Update the given webhook",
			// },
		},
	}
}
