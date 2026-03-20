package events

import (
	"encoding/json"
	"fmt"

	gqlclient "github.com/ScreenStaring/shopify-dev-tools/gql"
)

const productEventsQuery = `
query($id: ID!, $first: Int!, $after: String) {
  product(id: $id) {
    events(first: $first, after: $after, sortKey: CREATED_AT, reverse: true) {
      pageInfo {
        hasNextPage
        endCursor
      }
      edges {
        node {
          id
          appTitle
          action
          message
          createdAt
        }
      }
    }
  }
}
`

const productVariantEventsQuery = `
query($id: ID!, $first: Int!, $after: String) {
  productVariant(id: $id) {
    events(first: $first, after: $after, sortKey: CREATED_AT, reverse: true) {
      pageInfo {
        hasNextPage
        endCursor
      }
      edges {
        node {
          id
          appTitle
          action
          message
          createdAt
        }
      }
    }
  }
}
`

type Event struct {
	ID        string `json:"id"`
	AppTitle  string `json:"appTitle"`
	Action    string `json:"action"`
	Message   string `json:"message"`
	CreatedAt string `json:"createdAt"`
}

type eventsConnection struct {
	PageInfo struct {
		HasNextPage bool   `json:"hasNextPage"`
		EndCursor   string `json:"endCursor"`
	} `json:"pageInfo"`
	Edges []struct {
		Node Event `json:"node"`
	} `json:"edges"`
}

type productEventsResponse struct {
	Data struct {
		Product struct {
			Events eventsConnection `json:"events"`
		} `json:"product"`
	} `json:"data"`
}

type productVariantEventsResponse struct {
	Data struct {
		ProductVariant struct {
			Events eventsConnection `json:"events"`
		} `json:"productVariant"`
	} `json:"data"`
}

func fetchEvents(shop, token, gid, resourceType string, options map[string]interface{}) ([]Event, error) {
	client := gqlclient.NewClient(shop, token, options)

	vars := map[string]interface{}{
		"id":    gid,
		"first": 250,
	}

	var query string
	switch resourceType {
	case "Product":
		query = productEventsQuery
	case "ProductVariant":
		query = productVariantEventsQuery
	}

	var events []Event

	for {
		data, err := client.Execute(query, vars)
		if err != nil {
			return nil, fmt.Errorf("Cannot fetch events: %s", err)
		}

		b, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("Cannot re-encode events response: %s", err)
		}

		var conn eventsConnection

		switch resourceType {
		case "Product":
			var response productEventsResponse
			if err := json.Unmarshal(b, &response); err != nil {
				return nil, fmt.Errorf("Cannot parse events response: %s", err)
			}
			conn = response.Data.Product.Events
		case "ProductVariant":
			var response productVariantEventsResponse
			if err := json.Unmarshal(b, &response); err != nil {
				return nil, fmt.Errorf("Cannot parse events response: %s", err)
			}

			conn = response.Data.ProductVariant.Events
		}

		for _, edge := range conn.Edges {
			events = append(events, edge.Node)
		}

		if !conn.PageInfo.HasNextPage {
			break
		}

		vars["after"] = conn.PageInfo.EndCursor
	}

	return events, nil
}
