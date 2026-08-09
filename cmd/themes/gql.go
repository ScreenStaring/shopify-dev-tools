package themes

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/ScreenStaring/shopify-dev-tools/gql"
)

const themesQuery = `
query($first: Int!, $after: String) {
  themes(first: $first, after: $after) {
    nodes {
      id
      name
      role
      themeStoreId
      createdAt
      updatedAt
    }
    pageInfo {
      hasNextPage
      endCursor
    }
  }
}
`

const themeFilesUpsertMutation = `
mutation($themeId: ID!, $files: [OnlineStoreThemeFilesUpsertFileInput!]!) {
  themeFilesUpsert(themeId: $themeId, files: $files) {
    upsertedThemeFiles {
      filename
    }
    userErrors {
      field
      message
    }
  }
}
`

// Theme is the package's native theme shape as returned by the Admin
// GraphQL API (numeric id parsed from the gid, enum role).
type Theme struct {
	ID           int64
	Gid          string
	Name         string
	Role         string
	ThemeStoreID string
	CreatedAt    string
	UpdatedAt    string
}

type themesResponse struct {
	Data struct {
		Themes struct {
			Nodes []struct {
				ID           string `json:"id"`
				Name         string `json:"name"`
				Role         string `json:"role"`
				ThemeStoreID string `json:"themeStoreId"`
				CreatedAt    string `json:"createdAt"`
				UpdatedAt    string `json:"updatedAt"`
			} `json:"nodes"`
			PageInfo struct {
				HasNextPage bool   `json:"hasNextPage"`
				EndCursor   string `json:"endCursor"`
			} `json:"pageInfo"`
		} `json:"themes"`
	} `json:"data"`
}

func listThemes(client *gql.Client) ([]Theme, error) {
	vars := map[string]interface{}{
		"first": 250,
	}

	var themes []Theme

	for {
		data, err := client.Execute(themesQuery, vars)
		if err != nil {
			return nil, fmt.Errorf("Cannot list themes: %s", err)
		}

		b, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("Cannot list themes: %s", err)
		}

		var response themesResponse
		if err := json.Unmarshal(b, &response); err != nil {
			return nil, fmt.Errorf("Cannot list themes: %s", err)
		}

		for _, n := range response.Data.Themes.Nodes {
			themes = append(themes, Theme{
				ID:           themeIDFromGID(n.ID),
				Gid:          n.ID,
				Name:         n.Name,
				Role:         n.Role,
				ThemeStoreID: n.ThemeStoreID,
				CreatedAt:    n.CreatedAt,
				UpdatedAt:    n.UpdatedAt,
			})
		}

		if !response.Data.Themes.PageInfo.HasNextPage {
			break
		}

		vars["after"] = response.Data.Themes.PageInfo.EndCursor
	}

	return themes, nil
}

func themeIDFromGID(gid string) int64 {
	// gid://shopify/OnlineStoreTheme/123456
	parts := strings.Split(gid, "/")
	if len(parts) == 0 {
		return 0
	}

	id, err := strconv.ParseInt(parts[len(parts)-1], 10, 64)
	if err != nil {
		return 0
	}

	return id
}

// upsertThemeFiles uploads a single file to the theme. The body type is
// either "TEXT" or "BASE64" per OnlineStoreThemeFilesUpsertFileInput.
func upsertThemeFiles(client *gql.Client, themeID int64, filename, bodyType, value string) error {
	data, err := client.Execute(themeFilesUpsertMutation, map[string]interface{}{
		"themeId": fmt.Sprintf("gid://shopify/OnlineStoreTheme/%d", themeID),
		"files": []map[string]interface{}{
			{
				"filename": filename,
				"body":     map[string]interface{}{"type": bodyType, "value": value},
			},
		},
	})
	if err != nil {
		return err
	}

	userErrors, _ := data.ValuesForPath("data.themeFilesUpsert.userErrors")
	if len(userErrors) > 0 {
		ueMap := userErrors[0].(map[string]interface{})
		return fmt.Errorf("%s", ueMap["message"])
	}

	return nil
}
