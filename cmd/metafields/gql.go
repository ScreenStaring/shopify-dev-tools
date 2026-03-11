package metafields

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/ScreenStaring/shopify-dev-tools/gql"
)

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

	client := gql.NewClient(shop, token, "")

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
