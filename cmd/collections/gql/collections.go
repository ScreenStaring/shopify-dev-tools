package gql

import (
	"encoding/json"
	"fmt"
	"strings"

	gqlclient "github.com/ScreenStaring/shopify-dev-tools/gql"
)

const collectionQuery = `
query($id: ID!) {
  collection(id: $id) {
    id
    title
    handle
    productsCount {
      count
    }
    updatedAt
    ruleSet {
      appliedDisjunctively
    }
  }
}
`

const collectionsQuery = `
query($first: Int!, $after: String, $query: String) {
  collections(first: $first, after: $after, query: $query, sortKey: TITLE) {
    nodes {
      id
      title
      handle
      productsCount {
        count
      }
      updatedAt
      ruleSet {
        appliedDisjunctively
      }
    }
    pageInfo {
      hasNextPage
      endCursor
    }
  }
}
`

type Collection struct {
	ID            string
	Title         string
	Handle        string
	Type          string
	ProductsCount int
	UpdatedAt     string
}

type collectionJSON struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	Handle        string `json:"handle"`
	ProductsCount struct {
		Count int `json:"count"`
	} `json:"productsCount"`
	UpdatedAt string       `json:"updatedAt"`
	RuleSet   *ruleSetJSON `json:"ruleSet"`
}

type ruleSetJSON struct {
	AppliedDisjunctively bool `json:"appliedDisjunctively"`
}

func ToGID(id string) string {
	if strings.HasPrefix(id, "gid://") {
		return id
	}
	return "gid://shopify/Collection/" + id
}

func jsonToCollection(n collectionJSON) Collection {
	collectionType := "Custom"
	if n.RuleSet != nil {
		collectionType = "Smart"
	}

	return Collection{
		ID:            n.ID,
		Title:         n.Title,
		Handle:        n.Handle,
		Type:          collectionType,
		ProductsCount: n.ProductsCount.Count,
		UpdatedAt:     n.UpdatedAt,
	}
}

func GetCollection(shop, token, id string) (*Collection, error) {
	client := gqlclient.NewClient(shop, token)

	data, err := client.Execute(collectionQuery, map[string]interface{}{"id": ToGID(id)})
	if err != nil {
		return nil, fmt.Errorf("Cannot get collection: %s", err)
	}

	b, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("Cannot re-encode collection response: %s", err)
	}

	var response struct {
		Data struct {
			Collection *collectionJSON `json:"collection"`
		} `json:"data"`
	}

	if err := json.Unmarshal(b, &response); err != nil {
		return nil, fmt.Errorf("Cannot parse collection response: %s", err)
	}

	if response.Data.Collection == nil {
		return nil, fmt.Errorf("Collection not found")
	}

	c := jsonToCollection(*response.Data.Collection)
	return &c, nil
}

func ListCollections(shop, token string, limit, page int, title string, custom, smart bool) ([]Collection, error) {
	client := gqlclient.NewClient(shop, token)

	var parts []string
	if title != "" {
		parts = append(parts, "title:"+title)
	}
	if custom {
		parts = append(parts, "collection_type:custom")
	}
	if smart {
		parts = append(parts, "collection_type:smart")
	}
	query := strings.Join(parts, " ")

	if page < 1 {
		page = 1
	}

	var response struct {
		Data struct {
			Collections struct {
				Nodes    []collectionJSON `json:"nodes"`
				PageInfo struct {
					HasNextPage bool   `json:"hasNextPage"`
					EndCursor   string `json:"endCursor"`
				} `json:"pageInfo"`
			} `json:"collections"`
		} `json:"data"`
	}

	var after string
	for i := 0; i < page; i++ {
		vars := map[string]interface{}{"first": limit, "query": query}
		if after != "" {
			vars["after"] = after
		}

		data, err := client.Execute(collectionsQuery, vars)
		if err != nil {
			return nil, fmt.Errorf("Cannot list collections: %s", err)
		}

		b, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("Cannot re-encode collections response: %s", err)
		}

		response.Data.Collections.Nodes = nil
		if err := json.Unmarshal(b, &response); err != nil {
			return nil, fmt.Errorf("Cannot parse collections response: %s", err)
		}

		if !response.Data.Collections.PageInfo.HasNextPage && i < page-1 {
			break
		}

		after = response.Data.Collections.PageInfo.EndCursor
	}

	var result []Collection
	for _, n := range response.Data.Collections.Nodes {
		result = append(result, jsonToCollection(n))
	}

	return result, nil
}
