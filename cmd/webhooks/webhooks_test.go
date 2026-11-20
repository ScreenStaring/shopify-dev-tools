package webhooks

import (
	"testing"

	"github.com/urfave/cli/v2"
)

func newTestApp() *cli.App {
	return &cli.App{
		Commands: []*cli.Command{&Cmd},
	}
}

func TestWebhookTopicRegex(t *testing.T) {
	matching := []string{
		"orders/create",
		"ORDERS_CREATE",
		"products/update",
		"PRODUCTS_UPDATE",
		"CUSTOMERS_DATA_REQUEST",
		"inventory_levels/connect",
	}

	for _, s := range matching {
		if !webhookTopic.MatchString(s) {
			t.Errorf("expected %q to match webhookTopic regex", s)
		}
	}

	notMatching := []string{
		"12345",
		"gid://shopify/WebhookSubscription/12345",
		"invalid",
		"",
	}

	for _, s := range notMatching {
		if webhookTopic.MatchString(s) {
			t.Errorf("expected %q not to match webhookTopic regex", s)
		}
	}
}

func TestWebhooksFromArgs(t *testing.T) {
	t.Run("empty args", func(t *testing.T) {
		_, err := webhooksFromArgs("shop", "token", []string{}, nil)
		if err == nil || err.Error() != "You must supply a webhook id or topic" {
			t.Fatalf("expected 'You must supply a webhook id or topic', got: %v", err)
		}
	})

	t.Run("numeric id", func(t *testing.T) {
		hooks, err := webhooksFromArgs("shop", "token", []string{"123"}, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(hooks) != 1 || hooks[0].GID != "gid://shopify/WebhookSubscription/123" {
			t.Fatalf("unexpected hooks: %+v", hooks)
		}
	})

	t.Run("gid argument", func(t *testing.T) {
		hooks, err := webhooksFromArgs("shop", "token", []string{"gid://shopify/WebhookSubscription/456"}, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(hooks) != 1 || hooks[0].GID != "gid://shopify/WebhookSubscription/456" {
			t.Fatalf("unexpected hooks: %+v", hooks)
		}
	})
}

func TestUpdateActionValidation(t *testing.T) {
	t.Run("error when no arguments provided", func(t *testing.T) {
		app := newTestApp()
		err := app.Run([]string{"sdt", "webhook", "update", "--shop", "test-shop", "-a", "https://example.com/webhook"})
		if err == nil || err.Error() != "You must supply a webhook id or topic" {
			t.Fatalf("expected 'You must supply a webhook id or topic', got: %v", err)
		}
	})

	t.Run("error when no options provided", func(t *testing.T) {
		app := newTestApp()
		err := app.Run([]string{"sdt", "webhook", "update", "--shop", "test-shop", "12345"})
		if err == nil || err.Error() != "You must supply at least one option to update" {
			t.Fatalf("expected 'You must supply at least one option to update', got: %v", err)
		}
	})

	t.Run("error on invalid metafield format", func(t *testing.T) {
		app := newTestApp()
		err := app.Run([]string{"sdt", "webhook", "update", "--shop", "test-shop", "-m", "invalidformat", "12345"})
		if err == nil || err.Error() != `invalid metafield format "invalidformat", expected namespace.key` {
			t.Fatalf("expected invalid metafield error, got: %v", err)
		}
	})
}

func TestDeleteActionValidation(t *testing.T) {
	t.Run("error when no arguments provided and not all", func(t *testing.T) {
		app := newTestApp()
		err := app.Run([]string{"sdt", "webhook", "delete", "--shop", "test-shop"})
		if err == nil || err.Error() != "You must supply a webhook id or topic" {
			t.Fatalf("expected 'You must supply a webhook id or topic', got: %v", err)
		}
	})
}

func TestParseListArgs(t *testing.T) {
	t.Run("no flags and no args", func(t *testing.T) {
		topics, ids := parseListArgs("", nil)
		if len(topics) != 0 || len(ids) != 0 {
			t.Fatalf("expected empty topics and ids, got topics: %v, ids: %v", topics, ids)
		}
	})

	t.Run("topic flag only", func(t *testing.T) {
		topics, ids := parseListArgs("orders/create", nil)
		if len(topics) != 1 || topics[0] != "orders/create" {
			t.Fatalf("expected ['orders/create'], got: %v", topics)
		}
		if len(ids) != 0 {
			t.Fatalf("expected empty ids, got: %v", ids)
		}
	})

	t.Run("positional topics only", func(t *testing.T) {
		topics, ids := parseListArgs("", []string{"orders/create", "PRODUCTS_UPDATE"})
		if len(topics) != 2 || topics[0] != "orders/create" || topics[1] != "PRODUCTS_UPDATE" {
			t.Fatalf("unexpected topics: %v", topics)
		}
		if len(ids) != 0 {
			t.Fatalf("expected empty ids, got: %v", ids)
		}
	})

	t.Run("positional ids only", func(t *testing.T) {
		topics, ids := parseListArgs("", []string{"123", "gid://shopify/WebhookSubscription/456"})
		if len(topics) != 0 {
			t.Fatalf("expected empty topics, got: %v", topics)
		}
		if len(ids) != 2 || ids[0] != "123" || ids[1] != "gid://shopify/WebhookSubscription/456" {
			t.Fatalf("unexpected ids: %v", ids)
		}
	})

	t.Run("topic flag and positional args with topics and ids", func(t *testing.T) {
		topics, ids := parseListArgs("orders/create", []string{"123", "products/update", "456"})
		if len(topics) != 2 || topics[0] != "orders/create" || topics[1] != "products/update" {
			t.Fatalf("unexpected topics: %v", topics)
		}
		if len(ids) != 2 || ids[0] != "123" || ids[1] != "456" {
			t.Fatalf("unexpected ids: %v", ids)
		}
	})

	t.Run("deduplicates topics and ids", func(t *testing.T) {
		topics, ids := parseListArgs("orders/create", []string{"orders/create", "123", "123", "gid://shopify/WebhookSubscription/123"})
		if len(topics) != 1 || topics[0] != "orders/create" {
			t.Fatalf("unexpected topics: %v", topics)
		}
		if len(ids) != 1 || ids[0] != "123" {
			t.Fatalf("unexpected ids: %v", ids)
		}
	})
}

func TestDedupeWebhooks(t *testing.T) {
	hooks := []Webhook{
		{ID: 1, GID: "gid://shopify/WebhookSubscription/1", Topic: "ORDERS_CREATE"},
		{ID: 2, GID: "gid://shopify/WebhookSubscription/2", Topic: "PRODUCTS_UPDATE"},
		{ID: 1, GID: "gid://shopify/WebhookSubscription/1", Topic: "ORDERS_CREATE"},
	}

	deduped := dedupeWebhooks(hooks)
	if len(deduped) != 2 {
		t.Fatalf("expected 2 hooks, got: %d", len(deduped))
	}
	if deduped[0].ID != 1 || deduped[1].ID != 2 {
		t.Fatalf("unexpected order or elements: %+v", deduped)
	}
}

func TestListCommandMetadata(t *testing.T) {
	var lsCmd *cli.Command
	for _, sub := range Cmd.Subcommands {
		if sub.Name == "ls" {
			lsCmd = sub
			break
		}
	}

	if lsCmd == nil {
		t.Fatal("could not find ls subcommand")
	}

	if lsCmd.ArgsUsage != "[topic or webhook ID ...]" {
		t.Fatalf("expected ArgsUsage '[topic or webhook ID ...]', got %q", lsCmd.ArgsUsage)
	}
}
