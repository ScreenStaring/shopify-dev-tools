package webhooks

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ScreenStaring/shopify-dev-tools/gql"
)

const webhookSubscriptionsQuery = `
query($first: Int!, $topics: [WebhookSubscriptionTopic!]) {
  webhookSubscriptions(first: $first, topics: $topics) {
    edges {
      node {
        id
        legacyResourceId
        topic
        format
        includeFields
        metafieldNamespaces
        apiVersion { handle }
        createdAt
        updatedAt
        endpoint {
          __typename
          ... on WebhookHttpEndpoint {
            callbackUrl
          }
          ... on WebhookEventBridgeEndpoint {
            arn
          }
          ... on WebhookPubSubEndpoint {
            pubSubProject
            pubSubTopic
          }
        }
      }
    }
  }
}
`

const webhookSubscriptionCreateMutation = `
mutation webhookSubscriptionCreate($topic: WebhookSubscriptionTopic!, $webhookSubscription: WebhookSubscriptionInput!) {
  webhookSubscriptionCreate(topic: $topic, webhookSubscription: $webhookSubscription) {
    webhookSubscription {
      id
      legacyResourceId
    }
    userErrors {
      field
      message
    }
  }
}
`

const webhookSubscriptionDeleteMutation = `
mutation webhookSubscriptionDelete($id: ID!) {
  webhookSubscriptionDelete(id: $id) {
    deletedWebhookSubscriptionId
    userErrors {
      field
      message
    }
  }
}
`

const webhookSubscriptionUpdateMutation = `
mutation webhookSubscriptionUpdate($id: ID!, $webhookSubscription: WebhookSubscriptionInput!) {
  webhookSubscriptionUpdate(id: $id, webhookSubscription: $webhookSubscription) {
    webhookSubscription {
      id
      legacyResourceId
    }
    userErrors {
      field
      message
    }
  }
}
`

type Webhook struct {
	ID         int64    `json:"id"`
	GID        string   `json:"-"`
	Topic      string   `json:"topic"`
	Endpoint   string   `json:"endpoint"`
	Format     string   `json:"format"`
	Fields              []string `json:"fields"`
	MetafieldNamespaces []string `json:"metafieldNamespaces"`
	ApiVersion          string   `json:"apiVersion"`
	CreatedAt  string   `json:"createdAt"`
	UpdatedAt  string   `json:"updatedAt"`
}

type endpointJSON struct {
	Typename      string `json:"__typename"`
	CallbackUrl   string `json:"callbackUrl"`
	Arn           string `json:"arn"`
	PubSubProject string `json:"pubSubProject"`
	PubSubTopic   string `json:"pubSubTopic"`
}

type webhookJSON struct {
	ID               string       `json:"id"`
	LegacyResourceId int64        `json:"legacyResourceId,string"`
	Topic            string       `json:"topic"`
	Format           string       `json:"format"`
	IncludeFields       []string     `json:"includeFields"`
	MetafieldNamespaces []string     `json:"metafieldNamespaces"`
	ApiVersion          struct {
		Handle string `json:"handle"`
	} `json:"apiVersion"`
	CreatedAt string       `json:"createdAt"`
	UpdatedAt string       `json:"updatedAt"`
	Endpoint  endpointJSON `json:"endpoint"`
}

type webhooksResponse struct {
	Data struct {
		WebhookSubscriptions struct {
			Edges []struct {
				Node webhookJSON `json:"node"`
			} `json:"edges"`
		} `json:"webhookSubscriptions"`
	} `json:"data"`
}

func endpointAddress(e endpointJSON) string {
	switch e.Typename {
	case "WebhookHttpEndpoint":
		return e.CallbackUrl
	case "WebhookEventBridgeEndpoint":
		return e.Arn
	case "WebhookPubSubEndpoint":
		return e.PubSubProject + ":" + e.PubSubTopic
	default:
		return ""
	}
}

func topicToEnum(topic string) string {
	if strings.Contains(topic, "/") {
		return strings.ToUpper(strings.ReplaceAll(topic, "/", "_"))
	}
	return topic
}

func webhookGID(id string) string {
	if strings.HasPrefix(id, "gid://") {
		return id
	}
	return "gid://shopify/WebhookSubscription/" + id
}

func listWebhooks(shop, token string, topics []string) ([]Webhook, error) {
	client := gql.NewClient(shop, token, "")

	variables := map[string]interface{}{"first": 250}
	if len(topics) > 0 {
		enumTopics := make([]string, len(topics))
		for i, t := range topics {
			enumTopics[i] = topicToEnum(t)
		}
		variables["topics"] = enumTopics
	}

	data, err := client.Query(webhookSubscriptionsQuery, variables)
	if err != nil {
		return nil, fmt.Errorf("Cannot list webhooks: %s", err)
	}

	b, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("Cannot re-encode webhooks response: %s", err)
	}

	var response webhooksResponse
	if err := json.Unmarshal(b, &response); err != nil {
		return nil, fmt.Errorf("Cannot parse webhooks response: %s", err)
	}

	var result []Webhook
	for _, edge := range response.Data.WebhookSubscriptions.Edges {
		n := edge.Node
		result = append(result, Webhook{
			ID:         n.LegacyResourceId,
			GID:        n.ID,
			Topic:      n.Topic,
			Endpoint:   endpointAddress(n.Endpoint),
			Format:     n.Format,
			Fields:              n.IncludeFields,
			MetafieldNamespaces: n.MetafieldNamespaces,
			ApiVersion:          n.ApiVersion.Handle,
			CreatedAt:  n.CreatedAt,
			UpdatedAt:  n.UpdatedAt,
		})
	}

	return result, nil
}

func createWebhook(shop, token, topic, address, format string, fields []string) (string, error) {
	client := gql.NewClient(shop, token, "")

	input := map[string]interface{}{
		"callbackUrl": address,
		"format":      format,
	}
	if len(fields) > 0 {
		input["includeFields"] = fields
	}

	data, err := client.Mutation(webhookSubscriptionCreateMutation, map[string]interface{}{
		"topic":               topicToEnum(topic),
		"webhookSubscription": input,
	})
	if err != nil {
		return "", fmt.Errorf("Cannot create webhook: %s", err)
	}

	userErrors, _ := data.ValuesForPath("data.webhookSubscriptionCreate.userErrors")
	if len(userErrors) > 0 {
		ueMap := userErrors[0].(map[string]interface{})
		return "", fmt.Errorf("Cannot create webhook: %s", ueMap["message"])
	}

	id, err := data.ValueForPath("data.webhookSubscriptionCreate.webhookSubscription.legacyResourceId")
	if err != nil {
		return "", fmt.Errorf("Cannot read created webhook ID: %s", err)
	}

	return fmt.Sprint(id), nil
}

func updateWebhook(shop, token, gid string, input map[string]interface{}) error {
	client := gql.NewClient(shop, token, "")

	data, err := client.Mutation(webhookSubscriptionUpdateMutation, map[string]interface{}{
		"id":                  gid,
		"webhookSubscription": input,
	})
	if err != nil {
		return fmt.Errorf("Cannot update webhook: %s", err)
	}

	userErrors, _ := data.ValuesForPath("data.webhookSubscriptionUpdate.userErrors")
	if len(userErrors) > 0 {
		ueMap := userErrors[0].(map[string]interface{})
		return fmt.Errorf("Cannot update webhook: %s", ueMap["message"])
	}

	return nil
}

func deleteWebhook(shop, token, gid string) error {
	client := gql.NewClient(shop, token, "")

	data, err := client.Mutation(webhookSubscriptionDeleteMutation, map[string]interface{}{
		"id": gid,
	})
	if err != nil {
		return fmt.Errorf("Cannot delete webhook: %s", err)
	}

	userErrors, _ := data.ValuesForPath("data.webhookSubscriptionDelete.userErrors")
	if len(userErrors) > 0 {
		ueMap := userErrors[0].(map[string]interface{})
		return fmt.Errorf("Cannot delete webhook: %s", ueMap["message"])
	}

	return nil
}
