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
