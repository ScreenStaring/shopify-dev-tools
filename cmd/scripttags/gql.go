package scripttags

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ScreenStaring/shopify-dev-tools/gql"
)

const scriptTagsQuery = `
query($first: Int!, $after: String, $src: String) {
  scriptTags(first: $first, after: $after, query: $src) {
    nodes {
      id
      legacyResourceId
      cache
      createdAt
      displayScope
      src
      updatedAt
    }
    pageInfo {
      hasNextPage
      endCursor
    }
  }
}
`

const scriptTagDeleteMutation = `
mutation($id: ID!) {
  scriptTagDelete(id: $id) {
    deletedScriptTagId
    userErrors {
      field
      message
    }
  }
}
`

// ScriptTag is the package's native script tag shape as returned by the
// Admin GraphQL API (gid string, numeric legacy id).
type ScriptTag struct {
	ID               string `json:"id"`
	LegacyResourceID int64  `json:"legacyResourceId,string"`
	Src              string `json:"src"`
	Cache            bool   `json:"cache"`
	DisplayScope     string `json:"displayScope"`
	CreatedAt        string `json:"createdAt"`
	UpdatedAt        string `json:"updatedAt"`
}

type scriptTagsResponse struct {
	Data struct {
		ScriptTags struct {
			Nodes    []ScriptTag `json:"nodes"`
			PageInfo struct {
				HasNextPage bool   `json:"hasNextPage"`
				EndCursor   string `json:"endCursor"`
			} `json:"pageInfo"`
		} `json:"scriptTags"`
	} `json:"data"`
}

func listScriptTags(client *gql.Client, src string) ([]ScriptTag, error) {
	vars := map[string]interface{}{
		"first": 250,
	}

	if src != "" {
		vars["src"] = "src:" + src
	}

	var tags []ScriptTag

	for {
		data, err := client.Execute(scriptTagsQuery, vars)
		if err != nil {
			return nil, fmt.Errorf("Cannot list script tags: %s", err)
		}

		b, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("Cannot list script tags: %s", err)
		}

		var response scriptTagsResponse
		if err := json.Unmarshal(b, &response); err != nil {
			return nil, fmt.Errorf("Cannot list script tags: %s", err)
		}

		tags = append(tags, response.Data.ScriptTags.Nodes...)

		if !response.Data.ScriptTags.PageInfo.HasNextPage {
			break
		}

		vars["after"] = response.Data.ScriptTags.PageInfo.EndCursor
	}

	return tags, nil
}

func deleteScriptTag(client *gql.Client, id string) error {
	data, err := client.Execute(scriptTagDeleteMutation, map[string]interface{}{
		"id": id,
	})
	if err != nil {
		return fmt.Errorf("Cannot delete script tag: %s", err)
	}

	userErrors, _ := data.ValuesForPath("data.scriptTagDelete.userErrors")
	if len(userErrors) > 0 {
		ueMap := userErrors[0].(map[string]interface{})
		return fmt.Errorf("Cannot delete script tag: %s", ueMap["message"])
	}

	return nil
}

func scriptTagGID(id string) string {
	if strings.HasPrefix(id, "gid://") {
		return id
	}
	return "gid://shopify/ScriptTag/" + id
}
