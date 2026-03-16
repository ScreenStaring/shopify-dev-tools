package metafields

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/ScreenStaring/shopify-dev-tools/gql"
)

const metafieldDefinitionsQuery = `
query($ownerType: MetafieldOwnerType!, $first: Int!, $after: String, $namespace: String) {
  metafieldDefinitions(ownerType: $ownerType, first: $first, after: $after, namespace: $namespace) {
    edges {
      node {
        id
        name
        namespace
        key
        description
        type {
          name
        }
        ownerType
      }
    }
    pageInfo {
      hasNextPage
      endCursor
    }
  }
}
`

type MetafieldDefinition struct {
	ID          string
	Name        string
	Namespace   string
	Key         string
	Description string
	Type        string
	OwnerType   string
}

type metafieldDefinitionsResponse struct {
	Data struct {
		MetafieldDefinitions struct {
			Edges []struct {
				Node struct {
					ID          string `json:"id"`
					Name        string `json:"name"`
					Namespace   string `json:"namespace"`
					Key         string `json:"key"`
					Description string `json:"description"`
					Type        struct {
						Name string `json:"name"`
					} `json:"type"`
					OwnerType string `json:"ownerType"`
				} `json:"node"`
			} `json:"edges"`
			PageInfo struct {
				HasNextPage bool   `json:"hasNextPage"`
				EndCursor   string `json:"endCursor"`
			} `json:"pageInfo"`
		} `json:"metafieldDefinitions"`
	} `json:"data"`
}

func listMetafieldDefinitions(shop, token, ownerType, namespace string, options map[string]interface{}) ([]MetafieldDefinition, error) {
	client := gql.NewClient(shop, token, options)

	vars := map[string]interface{}{
		"ownerType": ownerType,
		"first":     250,
	}

	if namespace != "" {
		vars["namespace"] = namespace
	}

	var definitions []MetafieldDefinition

	for {
		data, err := client.Execute(metafieldDefinitionsQuery, vars)
		if err != nil {
			return nil, fmt.Errorf("Cannot list metafield definitions: %s", err)
		}

		b, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("Cannot list metafield definitions: %s", err)
		}

		var response metafieldDefinitionsResponse
		if err := json.Unmarshal(b, &response); err != nil {
			return nil, fmt.Errorf("Cannot list metafield definitions: %s", err)
		}

		for _, edge := range response.Data.MetafieldDefinitions.Edges {
			n := edge.Node
			definitions = append(definitions, MetafieldDefinition{
				ID:          n.ID,
				Name:        n.Name,
				Namespace:   n.Namespace,
				Key:         n.Key,
				Description: n.Description,
				Type:        n.Type.Name,
				OwnerType:   n.OwnerType,
			})
		}

		if !response.Data.MetafieldDefinitions.PageInfo.HasNextPage {
			break
		}

		vars["after"] = response.Data.MetafieldDefinitions.PageInfo.EndCursor
	}

	return definitions, nil
}

const metafieldsDeleteMutation = `
mutation metafieldsDelete($metafields: [MetafieldIdentifierInput!]!) {
  metafieldsDelete(metafields: $metafields) {
    deletedMetafields {
      key
      namespace
      ownerId
    }
    userErrors {
      field
      message
    }
  }
}
`

var fieldIndexRe = regexp.MustCompile(`\.(\d+)`)

type metafieldInput struct {
	OwnerID   string
	Namespace string
	Key       string
}

type DeletedMetafield struct {
	Key       string
	Namespace string
	OwnerID   string
	Error     string
}

// indexFromField extracts the numeric input index from a userError field path.
// The field value is a []interface{} of path segments (e.g. ["metafields", "0", "key"]);
// joined with "." that becomes "metafields.0.key" and we match the first ".N" segment.
func indexFromField(field interface{}) (int, bool) {
	items, ok := field.([]interface{})
	if !ok {
		return 0, false
	}

	parts := make([]string, len(items))
	for i, p := range items {
		parts[i] = fmt.Sprint(p)
	}

	match := fieldIndexRe.FindStringSubmatch(strings.Join(parts, "."))
	if match == nil {
		return 0, false
	}

	idx, err := strconv.Atoi(match[1])
	if err != nil {
		return 0, false
	}

	return idx, true
}

func deleteMetafields(shop, token string, metafields []metafieldInput) ([]DeletedMetafield, error) {
	inputs := make([]map[string]interface{}, len(metafields))
	for i, mf := range metafields {
		inputs[i] = map[string]interface{}{
			"ownerId":   mf.OwnerID,
			"namespace": mf.Namespace,
			"key":       mf.Key,
		}
	}

	client := gql.NewClient(shop, token)

	data, err := client.Execute(metafieldsDeleteMutation, map[string]interface{}{
		"metafields": inputs,
	})
	if err != nil {
		return nil, fmt.Errorf("Cannot delete metafields: %s", err)
	}

	result := make([]DeletedMetafield, len(metafields))

	// Map user errors back to their input positions via the index embedded in the field path.
	erroredIndices := make(map[int]bool)
	userErrors, _ := data.ValuesForPath("data.metafieldsDelete.userErrors")
	for _, ue := range userErrors {
		ueMap := ue.(map[string]interface{})
		message := fmt.Sprint(ueMap["message"])
		idx, ok := indexFromField(ueMap["field"])
		if ok && idx < len(result) {
			erroredIndices[idx] = true
			result[idx] = DeletedMetafield{Error: message, OwnerID: metafields[idx].OwnerID, Namespace: metafields[idx].Namespace, Key: metafields[idx].Key}
		} else {
			// No index in field path: general error, apply to all non-errored slots.
			for i := range result {
				if !erroredIndices[i] {
					erroredIndices[i] = true
					result[i] = DeletedMetafield{Error: message, OwnerID: metafields[i].OwnerID, Namespace: metafields[i].Namespace, Key: metafields[i].Key}
				}
			}
		}
	}

	// deletedMetafields is ordered to match the non-errored inputs.
	var successIndices []int
	for i := range metafields {
		if !erroredIndices[i] {
			successIndices = append(successIndices, i)
		}
	}

	nodes, _ := data.ValuesForPath("data.metafieldsDelete.deletedMetafields")
	for i, node := range nodes {
		if i >= len(successIndices) {
			break
		}
		n, ok := node.(map[string]interface{})
		if !ok {
			result[successIndices[i]] = DeletedMetafield{Error: "Not found or access denied", OwnerID: metafields[successIndices[i]].OwnerID, Namespace: metafields[successIndices[i]].Namespace, Key: metafields[successIndices[i]].Key}
			continue
		}
		result[successIndices[i]] = DeletedMetafield{
			Key:       fmt.Sprint(n["key"]),
			Namespace: fmt.Sprint(n["namespace"]),
			OwnerID:   fmt.Sprint(n["ownerId"]),
		}
	}

	return result, nil
}
