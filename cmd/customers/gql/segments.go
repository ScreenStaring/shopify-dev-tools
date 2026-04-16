package gql

import (
	"encoding/json"
	"fmt"
	"strings"

	gqlclient "github.com/ScreenStaring/shopify-dev-tools/gql"
)

const segmentQuery = `
query($id: ID!) {
  segment(id: $id) {
    id
    name
    query
    creationDate
    lastEditDate
  }
}
`

const segmentsQuery = `
query($first: Int!) {
  segments(first: $first, sortKey: CREATION_DATE, reverse: true) {
    edges {
      node {
        id
        name
        query
        creationDate
        lastEditDate
      }
    }
  }
}
`

const segmentDeleteMutation = `
mutation($id: ID!) {
  segmentDelete(id: $id) {
    deletedSegmentId
    userErrors {
      field
      message
    }
  }
}
`

type Segment struct {
	ID           string
	Name         string
	Query        string
	CreationDate string
	LastEditDate string
}

type segmentJSON struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Query        string `json:"query"`
	CreationDate string `json:"creationDate"`
	LastEditDate string `json:"lastEditDate"`
}

type segmentsResponse struct {
	Data struct {
		Segments struct {
			Edges []struct {
				Node segmentJSON `json:"node"`
			} `json:"edges"`
		} `json:"segments"`
	} `json:"data"`
}

func ToGID(id string) string {
	if strings.HasPrefix(id, "gid://") {
		return id
	}
	return "gid://shopify/Segment/" + id
}

func GetSegment(shop, token, id string) (*Segment, error) {
	client := gqlclient.NewClient(shop, token)

	data, err := client.Execute(segmentQuery, map[string]interface{}{"id": ToGID(id)})
	if err != nil {
		return nil, fmt.Errorf("Cannot get segment: %s", err)
	}

	b, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("Cannot re-encode segment response: %s", err)
	}

	var response struct {
		Data struct {
			Segment *segmentJSON `json:"segment"`
		} `json:"data"`
	}

	if err := json.Unmarshal(b, &response); err != nil {
		return nil, fmt.Errorf("Cannot parse segment response: %s", err)
	}

	if response.Data.Segment == nil {
		return nil, fmt.Errorf("Segment not found")
	}

	n := response.Data.Segment
	return &Segment{
		ID:           n.ID,
		Name:         n.Name,
		Query:        n.Query,
		CreationDate: n.CreationDate,
		LastEditDate: n.LastEditDate,
	}, nil
}

func ListSegments(shop, token string, limit int) ([]Segment, error) {
	client := gqlclient.NewClient(shop, token)

	data, err := client.Execute(segmentsQuery, map[string]interface{}{"first": limit})
	if err != nil {
		return nil, fmt.Errorf("Cannot list segments: %s", err)
	}

	b, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("Cannot re-encode segments response: %s", err)
	}

	var response segmentsResponse
	if err := json.Unmarshal(b, &response); err != nil {
		return nil, fmt.Errorf("Cannot parse segments response: %s", err)
	}

	var result []Segment
	for _, edge := range response.Data.Segments.Edges {
		n := edge.Node
		result = append(result, Segment{
			ID:           n.ID,
			Name:         n.Name,
			Query:        n.Query,
			CreationDate: n.CreationDate,
			LastEditDate: n.LastEditDate,
		})
	}

	return result, nil
}

func DeleteSegment(shop, token, id string) (string, error) {
	client := gqlclient.NewClient(shop, token)

	gid := ToGID(id)

	data, err := client.Execute(segmentDeleteMutation, map[string]interface{}{"id": gid})
	if err != nil {
		return "", fmt.Errorf("Cannot delete segment: %s", err)
	}

	b, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("Cannot re-encode segment delete response: %s", err)
	}

	var response struct {
		Data struct {
			SegmentDelete struct {
				DeletedSegmentId *string `json:"deletedSegmentId"`
				UserErrors       []struct {
					Field   []string `json:"field"`
					Message string   `json:"message"`
				} `json:"userErrors"`
			} `json:"segmentDelete"`
		} `json:"data"`
	}

	if err := json.Unmarshal(b, &response); err != nil {
		return "", fmt.Errorf("Cannot parse segment delete response: %s", err)
	}

	if errs := response.Data.SegmentDelete.UserErrors; len(errs) > 0 {
		var messages []string
		for _, e := range errs {
			messages = append(messages, e.Message)
		}
		return "", fmt.Errorf("Cannot delete segment: %s", strings.Join(messages, ", "))
	}

	if response.Data.SegmentDelete.DeletedSegmentId == nil {
		return "", fmt.Errorf("Segment not found")
	}

	return *response.Data.SegmentDelete.DeletedSegmentId, nil
}
