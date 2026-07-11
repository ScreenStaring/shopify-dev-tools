package gql

import (
	"encoding/json"
	"fmt"
	"strings"

	gqlclient "github.com/ScreenStaring/shopify-dev-tools/gql"
)

const metaobjectsQuery = `
query($type: String!, $first: Int!, $after: String, $query: String) {
  metaobjects(type: $type, first: $first, after: $after, query: $query, sortKey: "updated_at") {
    nodes {
      id
      handle
      type
      displayName
      updatedAt
      fields {
        key
        value
      }
    }
    pageInfo {
      hasNextPage
      endCursor
    }
  }
}
`

const metaobjectDefinitionsQuery = `
query($first: Int!, $after: String) {
  metaobjectDefinitions(first: $first, after: $after) {
    nodes {
      id
      name
      type
      displayNameKey
      fieldDefinitions {
        key
        name
        type {
          name
        }
        validations {
          name
          value
        }
      }
    }
    pageInfo {
      hasNextPage
      endCursor
    }
  }
}
`

const metaobjectDefinitionQuery = `
query($id: ID!) {
  metaobjectDefinition(id: $id) {
    id
    name
    type
    displayNameKey
    fieldDefinitions {
      key
      name
      type {
        name
      }
      validations {
        name
        value
      }
    }
  }
}
`

type MetaobjectField struct {
	Key   string
	Value string
}

type Metaobject struct {
	ID          string
	Handle      string
	Type        string
	DisplayName string
	UpdatedAt   string
	Fields      []MetaobjectField
}

type MetaobjectFieldValidation struct {
	Name  string
	Value string
}

type MetaobjectFieldDefinition struct {
	Key         string
	Name        string
	Type        string
	Validations []MetaobjectFieldValidation
}

type MetaobjectDefinition struct {
	ID             string
	Name           string
	Type           string
	DisplayNameKey string
	Fields         []MetaobjectFieldDefinition
}

type metaobjectJSON struct {
	ID          string `json:"id"`
	Handle      string `json:"handle"`
	Type        string `json:"type"`
	DisplayName string `json:"displayName"`
	UpdatedAt   string `json:"updatedAt"`
	Fields      []struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	} `json:"fields"`
}

type metaobjectDefinitionJSON struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Type             string `json:"type"`
	DisplayNameKey   string `json:"displayNameKey"`
	FieldDefinitions []struct {
		Key  string `json:"key"`
		Name string `json:"name"`
		Type struct {
			Name string `json:"name"`
		} `json:"type"`
		Validations []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"validations"`
	} `json:"fieldDefinitions"`
}

func ToDefinitionGID(id string) string {
	if strings.HasPrefix(id, "gid://") {
		return id
	}
	return "gid://shopify/MetaobjectDefinition/" + id
}

func jsonToMetaobject(n metaobjectJSON) Metaobject {
	fields := make([]MetaobjectField, len(n.Fields))
	for i, f := range n.Fields {
		fields[i] = MetaobjectField{Key: f.Key, Value: f.Value}
	}

	return Metaobject{
		ID:          n.ID,
		Handle:      n.Handle,
		Type:        n.Type,
		DisplayName: n.DisplayName,
		UpdatedAt:   n.UpdatedAt,
		Fields:      fields,
	}
}

func jsonToMetaobjectDefinition(n metaobjectDefinitionJSON) MetaobjectDefinition {
	fields := make([]MetaobjectFieldDefinition, len(n.FieldDefinitions))
	for i, f := range n.FieldDefinitions {
		validations := make([]MetaobjectFieldValidation, len(f.Validations))
		for j, v := range f.Validations {
			validations[j] = MetaobjectFieldValidation{Name: v.Name, Value: v.Value}
		}

		fields[i] = MetaobjectFieldDefinition{Key: f.Key, Name: f.Name, Type: f.Type.Name, Validations: validations}
	}

	return MetaobjectDefinition{
		ID:             n.ID,
		Name:           n.Name,
		Type:           n.Type,
		DisplayNameKey: n.DisplayNameKey,
		Fields:         fields,
	}
}

func ListMetaobjects(shop, token, moType string, limit, page int, query string, verbose bool) ([]Metaobject, error) {
	client := gqlclient.NewClient(shop, token, map[string]interface{}{"verbose": verbose})

	if page < 1 {
		page = 1
	}

	var response struct {
		Data struct {
			Metaobjects struct {
				Nodes    []metaobjectJSON `json:"nodes"`
				PageInfo struct {
					HasNextPage bool   `json:"hasNextPage"`
					EndCursor   string `json:"endCursor"`
				} `json:"pageInfo"`
			} `json:"metaobjects"`
		} `json:"data"`
	}

	var after string
	for i := 0; i < page; i++ {
		vars := map[string]interface{}{"type": moType, "first": limit, "query": query}
		if after != "" {
			vars["after"] = after
		}

		data, err := client.Execute(metaobjectsQuery, vars)
		if err != nil {
			return nil, fmt.Errorf("Cannot list metaobjects: %s", err)
		}

		b, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("Cannot re-encode metaobjects response: %s", err)
		}

		response.Data.Metaobjects.Nodes = nil
		if err := json.Unmarshal(b, &response); err != nil {
			return nil, fmt.Errorf("Cannot parse metaobjects response: %s", err)
		}

		if !response.Data.Metaobjects.PageInfo.HasNextPage && i < page-1 {
			break
		}

		after = response.Data.Metaobjects.PageInfo.EndCursor
	}

	var result []Metaobject
	for _, n := range response.Data.Metaobjects.Nodes {
		result = append(result, jsonToMetaobject(n))
	}

	return result, nil
}

func FetchAllMetaobjects(shop, token, moType, query string, verbose bool, fn func(Metaobject) error) error {
	client := gqlclient.NewClient(shop, token, map[string]interface{}{"verbose": verbose})

	var response struct {
		Data struct {
			Metaobjects struct {
				Nodes    []metaobjectJSON `json:"nodes"`
				PageInfo struct {
					HasNextPage bool   `json:"hasNextPage"`
					EndCursor   string `json:"endCursor"`
				} `json:"pageInfo"`
			} `json:"metaobjects"`
		} `json:"data"`
	}

	var after string
	for {
		vars := map[string]interface{}{"type": moType, "first": 250, "query": query}
		if after != "" {
			vars["after"] = after
		}

		data, err := client.Execute(metaobjectsQuery, vars)
		if err != nil {
			return fmt.Errorf("Cannot list metaobjects: %s", err)
		}

		b, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("Cannot re-encode metaobjects response: %s", err)
		}

		response.Data.Metaobjects.Nodes = nil
		if err := json.Unmarshal(b, &response); err != nil {
			return fmt.Errorf("Cannot parse metaobjects response: %s", err)
		}

		for _, n := range response.Data.Metaobjects.Nodes {
			if err := fn(jsonToMetaobject(n)); err != nil {
				return err
			}
		}

		if !response.Data.Metaobjects.PageInfo.HasNextPage {
			break
		}

		after = response.Data.Metaobjects.PageInfo.EndCursor
	}

	return nil
}

func GetMetaobjectDefinition(shop, token, id string, verbose bool) (*MetaobjectDefinition, error) {
	client := gqlclient.NewClient(shop, token, map[string]interface{}{"verbose": verbose})

	data, err := client.Execute(metaobjectDefinitionQuery, map[string]interface{}{"id": ToDefinitionGID(id)})
	if err != nil {
		return nil, fmt.Errorf("Cannot get metaobject definition: %s", err)
	}

	b, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("Cannot re-encode metaobject definition response: %s", err)
	}

	var response struct {
		Data struct {
			MetaobjectDefinition *metaobjectDefinitionJSON `json:"metaobjectDefinition"`
		} `json:"data"`
	}

	if err := json.Unmarshal(b, &response); err != nil {
		return nil, fmt.Errorf("Cannot parse metaobject definition response: %s", err)
	}

	if response.Data.MetaobjectDefinition == nil {
		return nil, fmt.Errorf("Metaobject definition not found")
	}

	d := jsonToMetaobjectDefinition(*response.Data.MetaobjectDefinition)
	return &d, nil
}

func ListMetaobjectDefinitions(shop, token string, limit, page int, verbose bool) ([]MetaobjectDefinition, error) {
	client := gqlclient.NewClient(shop, token, map[string]interface{}{"verbose": verbose})

	if page < 1 {
		page = 1
	}

	var response struct {
		Data struct {
			MetaobjectDefinitions struct {
				Nodes    []metaobjectDefinitionJSON `json:"nodes"`
				PageInfo struct {
					HasNextPage bool   `json:"hasNextPage"`
					EndCursor   string `json:"endCursor"`
				} `json:"pageInfo"`
			} `json:"metaobjectDefinitions"`
		} `json:"data"`
	}

	var after string
	for i := 0; i < page; i++ {
		vars := map[string]interface{}{"first": limit}
		if after != "" {
			vars["after"] = after
		}

		data, err := client.Execute(metaobjectDefinitionsQuery, vars)
		if err != nil {
			return nil, fmt.Errorf("Cannot list metaobject definitions: %s", err)
		}

		b, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("Cannot re-encode metaobject definitions response: %s", err)
		}

		response.Data.MetaobjectDefinitions.Nodes = nil
		if err := json.Unmarshal(b, &response); err != nil {
			return nil, fmt.Errorf("Cannot parse metaobject definitions response: %s", err)
		}

		if !response.Data.MetaobjectDefinitions.PageInfo.HasNextPage && i < page-1 {
			break
		}

		after = response.Data.MetaobjectDefinitions.PageInfo.EndCursor
	}

	var result []MetaobjectDefinition
	for _, n := range response.Data.MetaobjectDefinitions.Nodes {
		result = append(result, jsonToMetaobjectDefinition(n))
	}

	return result, nil
}
