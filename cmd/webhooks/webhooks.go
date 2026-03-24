package webhooks

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

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

func splitFields(raw []string) []string {
	var fields []string
	for _, f := range raw {
		for _, part := range strings.Split(f, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				fields = append(fields, part)
			}
		}
	}
	return fields
}

func parseMetafields(raw []string) ([]map[string]string, error) {
	var result []map[string]string
	for _, entry := range splitFields(raw) {
		parts := strings.SplitN(entry, ".", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return nil, fmt.Errorf("invalid metafield format %q, expected namespace.key", entry)
		}
		result = append(result, map[string]string{"namespace": parts[0], "key": parts[1]})
	}
	return result, nil
}

func createAction(c *cli.Context) error {
	shop := c.String("shop")
	token := cmd.LookupAccessToken(shop, c.String("access-token"))
	options := map[string]interface{}{"version": c.String("api-version"), "verbose": c.Bool("verbose")}

	metafields, err := parseMetafields(c.StringSlice("metafields"))
	if err != nil {
		return err
	}

	if len(metafields) > 0 {
		options["metafields"] = metafields
	}

	namespaces := splitFields(c.StringSlice("namespaces"))
	if len(namespaces) > 0 {
		options["metafieldNamespaces"] = namespaces
	}

	id, err := createWebhook(shop, token, c.String("topic"), c.String("address"), format(c), splitFields(c.StringSlice("fields")), options)
	if err != nil {
		return err
	}

	fmt.Printf("Webhook created: %s\n", id)
	return nil
}

func deleteAction(c *cli.Context) error {
	shop := c.String("shop")
	token := cmd.LookupAccessToken(shop, c.String("access-token"))
	options := map[string]interface{}{"version": c.String("api-version"), "verbose": c.Bool("verbose")}

	var webhooks []Webhook

	if c.Bool("all") {
		var err error
		webhooks, err = listWebhooks(shop, token, nil, options)
		if err != nil {
			return err
		}
	} else {
		if c.Args().Len() == 0 {
			return fmt.Errorf("You must supply a webhook id or topic")
		}

		for _, arg := range c.Args().Slice() {
			if webhookName.MatchString(arg) {
				found, err := listWebhooks(shop, token, []string{arg}, options)
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
		err := deleteWebhook(shop, token, w.GID, options)
		if err != nil {
			return err
		}
	}

	fmt.Printf("%d webhook(s) deleted\n", len(webhooks))

	return nil
}

func updateAction(c *cli.Context) error {
	if c.Args().Len() == 0 {
		return fmt.Errorf("You must supply a webhook id to update")
	}

	shop := c.String("shop")
	token := cmd.LookupAccessToken(shop, c.String("access-token"))
	options := map[string]interface{}{"version": c.String("api-version"), "verbose": c.Bool("verbose")}
	gid := webhookGID(c.Args().Get(0))

	input := map[string]interface{}{}
	if c.IsSet("address") {
		input["callbackUrl"] = c.String("address")
	}
	if c.IsSet("topic") {
		input["topic"] = topicToEnum(c.String("topic"))
	}
	if c.IsSet("fields") {
		input["includeFields"] = splitFields(c.StringSlice("fields"))
	}
	if c.IsSet("namespaces") {
		input["metafieldNamespaces"] = splitFields(c.StringSlice("namespaces"))
	}
	if c.IsSet("metafields") {
		metafields, err := parseMetafields(c.StringSlice("metafields"))
		if err != nil {
			return err
		}
		input["metafields"] = metafields
	}

	if len(input) == 0 {
		return fmt.Errorf("You must supply at least one option to update")
	}

	err := updateWebhook(shop, token, gid, input, options)
	if err != nil {
		return err
	}

	fmt.Println("Webhook updated")
	return nil
}

func listAction(c *cli.Context) error {
	shop := c.String("shop")
	token := cmd.LookupAccessToken(shop, c.String("access-token"))
	options := map[string]interface{}{"version": c.String("api-version")}

	hooks, err := listWebhooks(shop, token, nil, options)
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
	apiVersionFlag := &cli.StringFlag{
		Name:  "api-version",
		Usage: "API version to use; default is a versionless call",
	}

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
		&cli.StringSliceFlag{
			Name:    "namespaces",
			Aliases: []string{"n"},
			Usage:   "Metafield namespaces to include in the webhook",
		},
		&cli.StringSliceFlag{
			Name:    "metafields",
			Aliases: []string{"m"},
			Usage:   "Metafields to include in the webhook in namespace.key format",
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
		apiVersionFlag,
	}

	updateFlags := []cli.Flag{
		&cli.StringFlag{
			Name: "address",
			Aliases: []string{"a"},
		},
		&cli.StringSliceFlag{
			Name: "fields",
			Aliases: []string{"f"},
		},
		&cli.StringSliceFlag{
			Name:    "namespaces",
			Aliases: []string{"n"},
			Usage:   "Metafield namespaces to include in the webhook",
		},
		&cli.StringSliceFlag{
			Name:    "metafields",
			Aliases: []string{"m"},
			Usage:   "Metafields to include in the webhook in namespace.key format",
		},
		&cli.StringFlag{
			Name: "topic",
			Aliases: []string{"t"},
		},
		apiVersionFlag,
	}

	deleteFlags := []cli.Flag{
		&cli.BoolFlag{
			Name: "all",
			Aliases: []string{"a"},
		},
		apiVersionFlag,
	}

	listFlags := []cli.Flag{
		&cli.BoolFlag{
			Name: "jsonl",
			Aliases: []string{"j"},
		},
		apiVersionFlag,
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
				Name: "update",
				ArgsUsage: "[webhook ID]",
				Aliases: []string{"u"},
				Flags: append(cmd.Flags, updateFlags...),
				Action: updateAction,
				Usage: "Update the given webhook",
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
