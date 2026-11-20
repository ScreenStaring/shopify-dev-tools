package webhooks

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/cheynewallace/tabby"
	"github.com/urfave/cli/v2"

	"github.com/ScreenStaring/shopify-dev-tools/cmd"
)

var Cmd cli.Command
var webhookTopic = regexp.MustCompile(`(?i)\A[_a-zA-Z]+/[_a-zA-Z]+\z|\A[A-Z0-9]+_[A-Z0-9_]+\z`)

func format(c *cli.Context) string {
	if c.Bool("xml") {
		return "XML"
	}

	return "JSON"
}

func printFormatted(webhooks []Webhook) {
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

func printJSONL(webhooks []Webhook) {
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
	options := map[string]interface{}{"verbose": c.Bool("verbose")}

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

	topic := c.String("topic")
	address := c.String("address")

	if c.Bool("one") {
		existing, err := listWebhooks(shop, token, []string{topic}, options)
		if err != nil {
			return err
		}
		for _, w := range existing {
			if w.Endpoint == address {
				return fmt.Errorf("Webhook already exists for topic %s at %s", topic, address)
			}
		}
	}

	id, err := createWebhook(shop, token, topic, address, format(c), splitFields(c.StringSlice("fields")), options)
	if err != nil {
		return err
	}

	fmt.Printf("Webhook created: %s\n", id)
	return nil
}

func webhooksFromArgs(shop, token string, args []string, options map[string]interface{}) ([]Webhook, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("You must supply a webhook id or topic")
	}

	var webhooks []Webhook

	for _, arg := range args {
		if webhookTopic.MatchString(arg) {
			found, err := listWebhooks(shop, token, []string{arg}, options)
			if err != nil {
				return nil, fmt.Errorf("Cannot list webhooks for topic %s: %s", arg, err)
			}
			webhooks = append(webhooks, found...)
		} else {
			webhooks = append(webhooks, Webhook{GID: webhookGID(arg)})
		}
	}

	if len(webhooks) == 0 {
		return nil, fmt.Errorf("No webhooks found")
	}

	return webhooks, nil
}

func deleteAction(c *cli.Context) error {
	shop := c.String("shop")
	token := cmd.LookupAccessToken(shop, c.String("access-token"))
	options := map[string]interface{}{"verbose": c.Bool("verbose")}

	var webhooks []Webhook

	if c.Bool("all") {
		var err error
		webhooks, err = listWebhooks(shop, token, nil, options)
		if err != nil {
			return err
		}
		if len(webhooks) == 0 {
			return fmt.Errorf("No webhooks found")
		}
	} else {
		var err error
		webhooks, err = webhooksFromArgs(shop, token, c.Args().Slice(), options)
		if err != nil {
			return err
		}
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
		return fmt.Errorf("You must supply a webhook id or topic")
	}

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

	shop := c.String("shop")
	token := cmd.LookupAccessToken(shop, c.String("access-token"))
	options := map[string]interface{}{"verbose": c.Bool("verbose")}

	webhooks, err := webhooksFromArgs(shop, token, c.Args().Slice(), options)
	if err != nil {
		return err
	}

	for _, w := range webhooks {
		err := updateWebhook(shop, token, w.GID, input, options)
		if err != nil {
			return err
		}
	}

	fmt.Printf("%d webhook(s) updated\n", len(webhooks))
	return nil
}

func parseListArgs(topicFlag string, args []string) (topics []string, ids []string) {
	seenTopics := make(map[string]bool)
	seenIDs := make(map[string]bool)

	addTopic := func(t string) {
		if !seenTopics[t] {
			seenTopics[t] = true
			topics = append(topics, t)
		}
	}

	addID := func(id string) {
		gid := webhookGID(id)
		if !seenIDs[gid] {
			seenIDs[gid] = true
			ids = append(ids, id)
		}
	}

	if topicFlag != "" {
		addTopic(topicFlag)
	}

	for _, arg := range args {
		if webhookTopic.MatchString(arg) {
			addTopic(arg)
		} else {
			addID(arg)
		}
	}

	return topics, ids
}

func dedupeWebhooks(webhooks []Webhook) []Webhook {
	seen := make(map[string]bool, len(webhooks))
	var unique []Webhook
	for _, w := range webhooks {
		key := w.GID
		if key == "" {
			key = fmt.Sprint(w.ID)
		}
		if !seen[key] {
			seen[key] = true
			unique = append(unique, w)
		}
	}
	return unique
}

func listAction(c *cli.Context) error {
	shop := c.String("shop")
	token := cmd.LookupAccessToken(shop, c.String("access-token"))
	options := map[string]interface{}{"verbose": c.Bool("verbose")}

	if c.IsSet("address") {
		options["address"] = c.String("address")
	}

	var topicFlag string
	if c.IsSet("topic") {
		topicFlag = c.String("topic")
	}

	topics, ids := parseListArgs(topicFlag, c.Args().Slice())

	var hooks []Webhook

	if len(topics) == 0 && len(ids) == 0 {
		var err error
		hooks, err = listWebhooks(shop, token, nil, options)
		if err != nil {
			return err
		}
	} else {
		if len(topics) > 0 {
			topicHooks, err := listWebhooks(shop, token, topics, options)
			if err != nil {
				return err
			}
			hooks = append(hooks, topicHooks...)
		}

		if len(ids) > 0 {
			idHooks, err := webhooksByID(shop, token, ids, options)
			if err != nil {
				return err
			}
			hooks = append(hooks, idHooks...)
		}

		hooks = dedupeWebhooks(hooks)
	}

	if c.Bool("jsonl") {
		printJSONL(hooks)
	} else {
		printFormatted(hooks)
	}

	return nil
}

func init() {
	apiVersionFlag := cmd.APIVersionFlag

	createFlags := []cli.Flag{
		&cli.StringFlag{
			Name:     "address",
			Required: true,
			Aliases:  []string{"a"},
		},
		&cli.StringSliceFlag{
			Name:    "fields",
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
			Name:    "one",
			Aliases: []string{"1"},
			Usage:   "Error if a webhook already exists for this topic and address",
		},
		&cli.BoolFlag{
			Name:  "xml",
			Value: false,
		},
		&cli.StringFlag{
			Name:     "topic",
			Required: true,
			Aliases:  []string{"t"},
		},
		apiVersionFlag,
	}

	updateFlags := []cli.Flag{
		&cli.StringFlag{
			Name:    "address",
			Aliases: []string{"a"},
		},
		&cli.StringSliceFlag{
			Name:    "fields",
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
			Name:    "topic",
			Aliases: []string{"t"},
		},
		apiVersionFlag,
	}

	deleteFlags := []cli.Flag{
		&cli.BoolFlag{
			Name:    "all",
			Aliases: []string{"a"},
		},
		apiVersionFlag,
	}

	listFlags := []cli.Flag{
		&cli.StringFlag{
			Name:    "address",
			Aliases: []string{"a"},
			Usage:   "Only show webhooks with the given address",
		},
		&cli.BoolFlag{
			Name:    "jsonl",
			Aliases: []string{"j"},
			Usage:   "Output the webhooks in JSONL format",
		},
		&cli.StringFlag{
			Name:    "topic",
			Aliases: []string{"t"},
			Usage:   "Only show webhooks with the given topic",
		},
		apiVersionFlag,
	}

	Cmd = cli.Command{
		Name:    "webhook",
		Aliases: []string{"webhooks", "hooks", "w"},
		Usage:   "Webhook utilities",
		Subcommands: []*cli.Command{
			{
				Name:    "create",
				Aliases: []string{"c"},
				Flags:   append(cmd.Flags, createFlags...),
				Action:  createAction,
				Usage:   "Create a webhook for the given shop",
			},
			{
				Name:      "delete",
				ArgsUsage: "[topic or webhook ID ...]",
				Aliases:   []string{"del", "rm", "d"},
				Flags:     append(cmd.Flags, deleteFlags...),
				Action:    deleteAction,
				Usage:     "Delete the given webhook",
			},
			{
				Name:      "update",
				ArgsUsage: "[topic or webhook ID ...]",
				Aliases:   []string{"u"},
				Flags:     append(cmd.Flags, updateFlags...),
				Action:    updateAction,
				Usage:     "Update the given webhook",
			},
			{
				Name:      "ls",
				ArgsUsage: "[topic or webhook ID ...]",
				Flags:     append(cmd.Flags, listFlags...),
				Action:    listAction,
				Usage:     "List the shop's webhooks",
			},
		},
	}
}
