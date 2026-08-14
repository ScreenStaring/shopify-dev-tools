package locations

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/ScreenStaring/shopify-dev-tools/gql"
)

const locationsQuery = `
query($first: Int!, $after: String) {
  locations(first: $first, after: $after) {
    pageInfo {
      hasNextPage
      endCursor
    }
    edges {
      node {
        id
        name
        isActive
        isFulfillmentService
        createdAt
        updatedAt
        address {
          formatted
        }
      }
    }
  }
}
`

const locationsByIDQuery = `
query($ids: [ID!]!) {
  nodes(ids: $ids) {
    ... on Location {
      id
      name
      isActive
      isFulfillmentService
      createdAt
      updatedAt
      address {
        formatted
      }
    }
  }
}
`

type Location struct {
	ID                 int64  `json:"id"`
	Name               string `json:"name"`
	Active             bool   `json:"isActive"`
	FulfillmentService bool   `json:"isFulfillmentService"`
	CreatedAt          string `json:"createdAt"`
	UpdatedAt          string `json:"updatedAt"`
	Address            string `json:"address"`
}

type locationNodeJSON struct {
	ID                   string `json:"id"`
	Name                 string `json:"name"`
	IsActive             bool   `json:"isActive"`
	IsFulfillmentService bool   `json:"isFulfillmentService"`
	CreatedAt            string `json:"createdAt"`
	UpdatedAt            string `json:"updatedAt"`
	Address              struct {
		Formatted []string `json:"formatted"`
	} `json:"address"`
}

type locationsResponse struct {
	Data struct {
		Locations struct {
			PageInfo struct {
				HasNextPage bool   `json:"hasNextPage"`
				EndCursor   string `json:"endCursor"`
			} `json:"pageInfo"`
			Edges []struct {
				Node locationNodeJSON `json:"node"`
			} `json:"edges"`
		} `json:"locations"`
	} `json:"data"`
}

type locationsByIDResponse struct {
	Data struct {
		Nodes []*locationNodeJSON `json:"nodes"`
	} `json:"data"`
}

func ListLocations(client *gql.Client) ([]Location, error) {
	var locations []Location
	vars := map[string]interface{}{"first": 250}

	for {
		data, err := client.Execute(locationsQuery, vars)
		if err != nil {
			return nil, fmt.Errorf("Cannot list locations: %s", err)
		}

		b, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("Cannot re-encode locations response: %s", err)
		}

		var response locationsResponse
		if err := json.Unmarshal(b, &response); err != nil {
			return nil, fmt.Errorf("Cannot parse locations response: %s", err)
		}

		for _, edge := range response.Data.Locations.Edges {
			locations = append(locations, locationFromNode(edge.Node))
		}

		if !response.Data.Locations.PageInfo.HasNextPage {
			break
		}

		vars["after"] = response.Data.Locations.PageInfo.EndCursor
	}

	return locations, nil
}

func LocationsByID(client *gql.Client, ids []string) ([]Location, error) {
	data, err := client.Execute(locationsByIDQuery, map[string]interface{}{"ids": ids})
	if err != nil {
		return nil, fmt.Errorf("Cannot get locations: %s", err)
	}

	b, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("Cannot re-encode locations response: %s", err)
	}

	var response locationsByIDResponse
	if err := json.Unmarshal(b, &response); err != nil {
		return nil, fmt.Errorf("Cannot parse locations response: %s", err)
	}

	var locations []Location
	for i, node := range response.Data.Nodes {
		if node == nil {
			return nil, fmt.Errorf("Cannot get location %s: no location found with that id", ids[i])
		}

		locations = append(locations, locationFromNode(*node))
	}

	return locations, nil
}

func locationFromNode(n locationNodeJSON) Location {
	loc := Location{
		ID:                 idFromGID(n.ID),
		Name:               n.Name,
		Active:             n.IsActive,
		FulfillmentService: n.IsFulfillmentService,
		CreatedAt:          n.CreatedAt,
		UpdatedAt:          n.UpdatedAt,
	}

	if len(n.Address.Formatted) > 0 {
		loc.Address = strings.Join(n.Address.Formatted, ", ")
	}

	return loc
}

// idFromGID extracts the numeric id from a Shopify GID like
// gid://shopify/Location/123456.
func idFromGID(gid string) int64 {
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
