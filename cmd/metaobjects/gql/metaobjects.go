package gql

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	gqlclient "github.com/ScreenStaring/shopify-dev-tools/gql"
	"github.com/clbanning/mxj"
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

const metaobjectByHandleQuery = `
query($handle: MetaobjectHandleInput!) {
  metaobjectByHandle(handle: $handle) {
    id
    handle
    type
    displayName
  }
}
`

const metaobjectDefinitionByTypeQuery = `
query($type: String!) {
  metaobjectDefinitionByType(type: $type) {
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

// DefinitionGIDPrefix is the GID prefix of metaobject definition ids. Ids are
// displayed without it.
const DefinitionGIDPrefix = "gid://shopify/MetaobjectDefinition/"

// MetaobjectGIDPrefix is the GID prefix of metaobject ids. Ids are displayed
// without it.
const MetaobjectGIDPrefix = "gid://shopify/Metaobject/"

func ToDefinitionGID(id string) string {
	if strings.HasPrefix(id, "gid://") {
		return id
	}
	return DefinitionGIDPrefix + id
}

func ToMetaobjectGID(id string) string {
	if strings.HasPrefix(id, "gid://") {
		return id
	}
	return MetaobjectGIDPrefix + id
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

	d, err := metaobjectDefinition(client, metaobjectDefinitionQuery, "metaobjectDefinition", map[string]interface{}{"id": ToDefinitionGID(id)})
	if err != nil {
		return nil, err
	}

	if d == nil {
		return nil, fmt.Errorf("Metaobject definition not found")
	}

	return d, nil
}

// MetaobjectDefinitionByType returns the metaobject definition of the given
// type, or nil when the shop has no definition of that type.
func MetaobjectDefinitionByType(client *gqlclient.Client, moType string) (*MetaobjectDefinition, error) {
	return metaobjectDefinition(client, metaobjectDefinitionByTypeQuery, "metaobjectDefinitionByType", map[string]interface{}{"type": moType})
}

// metaobjectDefinition runs one of the metaobject definition queries and
// returns the definition found at the response's key, or nil when there was
// none.
func metaobjectDefinition(client *gqlclient.Client, query, key string, variables map[string]interface{}) (*MetaobjectDefinition, error) {
	data, err := client.Execute(query, variables)
	if err != nil {
		return nil, fmt.Errorf("Cannot get metaobject definition: %s", err)
	}

	b, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("Cannot re-encode metaobject definition response: %s", err)
	}

	var response struct {
		Data map[string]*metaobjectDefinitionJSON `json:"data"`
	}

	if err := json.Unmarshal(b, &response); err != nil {
		return nil, fmt.Errorf("Cannot parse metaobject definition response: %s", err)
	}

	found, ok := response.Data[key]
	if !ok || found == nil {
		return nil, nil
	}

	d := jsonToMetaobjectDefinition(*found)
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

const metaobjectDefinitionCreateMutation = `
mutation metaobjectDefinitionCreate($definition: MetaobjectDefinitionCreateInput!) {
  metaobjectDefinitionCreate(definition: $definition) {
    metaobjectDefinition {
      id
      name
      type
    }
    userErrors {
      field
      message
      code
    }
  }
}
`

const metaobjectDefinitionDeleteMutation = `
mutation metaobjectDefinitionDelete($id: ID!) {
  metaobjectDefinitionDelete(id: $id) {
    deletedId
    userErrors {
      field
      message
      code
    }
  }
}
`

// CreateMetaobjectDefinition creates one metaobject definition from the given
// MetaobjectDefinitionCreateInput and returns the created definition's id.
func CreateMetaobjectDefinition(client *gqlclient.Client, definition map[string]interface{}) (string, error) {
	data, err := client.Execute(metaobjectDefinitionCreateMutation, map[string]interface{}{
		"definition": definition,
	})
	if err != nil {
		return "", err
	}

	if messages := userErrorMessages(data, "data.metaobjectDefinitionCreate.userErrors"); len(messages) > 0 {
		return "", errors.New(strings.Join(messages, "; "))
	}

	ids, _ := data.ValuesForPath("data.metaobjectDefinitionCreate.metaobjectDefinition.id")
	if len(ids) == 0 || ids[0] == nil {
		return "", errors.New("no id returned")
	}

	return fmt.Sprint(ids[0]), nil
}

// DeleteMetaobjectDefinition deletes the metaobject definition with the given
// numeric id or GID and returns the deleted definition's id. The definition's
// metafield definitions, metaobjects and metafields are deleted by Shopify
// asynchronously.
func DeleteMetaobjectDefinition(client *gqlclient.Client, id string) (string, error) {
	data, err := client.Execute(metaobjectDefinitionDeleteMutation, map[string]interface{}{
		"id": ToDefinitionGID(id),
	})
	if err != nil {
		return "", err
	}

	if messages := userErrorMessages(data, "data.metaobjectDefinitionDelete.userErrors"); len(messages) > 0 {
		return "", errors.New(strings.Join(messages, "; "))
	}

	ids, _ := data.ValuesForPath("data.metaobjectDefinitionDelete.deletedId")
	if len(ids) == 0 || ids[0] == nil {
		return "", errors.New("not found or access denied")
	}

	return fmt.Sprint(ids[0]), nil
}

const metaobjectCreateMutation = `
mutation metaobjectCreate($metaobject: MetaobjectCreateInput!) {
  metaobjectCreate(metaobject: $metaobject) {
    metaobject {
      id
      handle
      type
    }
    userErrors {
      field
      message
      code
    }
  }
}
`

const metaobjectUpsertMutation = `
mutation metaobjectUpsert($handle: MetaobjectHandleInput!, $metaobject: MetaobjectUpsertInput!) {
  metaobjectUpsert(handle: $handle, metaobject: $metaobject) {
    metaobject {
      id
      handle
      type
    }
    userErrors {
      field
      message
      code
    }
  }
}
`

const metaobjectDeleteMutation = `
mutation metaobjectDelete($id: ID!) {
  metaobjectDelete(id: $id) {
    deletedId
    userErrors {
      field
      message
      code
    }
  }
}
`

// CreateMetaobject creates one metaobject from the given
// MetaobjectCreateInput and returns the created metaobject's id.
func CreateMetaobject(client *gqlclient.Client, metaobject map[string]interface{}) (string, error) {
	data, err := client.Execute(metaobjectCreateMutation, map[string]interface{}{
		"metaobject": metaobject,
	})
	if err != nil {
		return "", err
	}

	if messages := userErrorMessages(data, "data.metaobjectCreate.userErrors"); len(messages) > 0 {
		return "", errors.New(strings.Join(messages, "; "))
	}

	return metaobjectID(data, "data.metaobjectCreate.metaobject.id")
}

// UpsertMetaobject creates or updates the metaobject of the given type and
// handle from the given MetaobjectUpsertInput and returns its id. Only the
// fields in the input are set: the fields it leaves out keep their value.
func UpsertMetaobject(client *gqlclient.Client, moType, handle string, metaobject map[string]interface{}) (string, error) {
	data, err := client.Execute(metaobjectUpsertMutation, map[string]interface{}{
		"handle":     map[string]interface{}{"type": moType, "handle": handle},
		"metaobject": metaobject,
	})
	if err != nil {
		return "", err
	}

	if messages := userErrorMessages(data, "data.metaobjectUpsert.userErrors"); len(messages) > 0 {
		return "", errors.New(strings.Join(messages, "; "))
	}

	return metaobjectID(data, "data.metaobjectUpsert.metaobject.id")
}

// MetaobjectByHandle returns the metaobject of the given type with the given
// handle, or nil when the shop has no metaobject of that type with that handle.
// App owned types are given with their '$app:' prefix.
func MetaobjectByHandle(client *gqlclient.Client, moType, handle string) (*Metaobject, error) {
	return metaobject(client, metaobjectByHandleQuery, "metaobjectByHandle", map[string]interface{}{
		"handle": map[string]interface{}{"type": moType, "handle": handle},
	})
}

// metaobject runs one of the single metaobject queries and returns the
// metaobject found at the response's key, or nil when there was none.
func metaobject(client *gqlclient.Client, query, key string, variables map[string]interface{}) (*Metaobject, error) {
	data, err := client.Execute(query, variables)
	if err != nil {
		return nil, fmt.Errorf("Cannot get metaobject: %s", err)
	}

	b, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("Cannot re-encode metaobject response: %s", err)
	}

	var response struct {
		Data map[string]*metaobjectJSON `json:"data"`
	}

	if err := json.Unmarshal(b, &response); err != nil {
		return nil, fmt.Errorf("Cannot parse metaobject response: %s", err)
	}

	found, ok := response.Data[key]
	if !ok || found == nil {
		return nil, nil
	}

	mo := jsonToMetaobject(*found)
	return &mo, nil
}

// DeleteMetaobject deletes the metaobject with the given numeric id or GID and
// returns the deleted metaobject's id.
func DeleteMetaobject(client *gqlclient.Client, id string) (string, error) {
	data, err := client.Execute(metaobjectDeleteMutation, map[string]interface{}{
		"id": ToMetaobjectGID(id),
	})
	if err != nil {
		return "", err
	}

	if messages := userErrorMessages(data, "data.metaobjectDelete.userErrors"); len(messages) > 0 {
		return "", errors.New(strings.Join(messages, "; "))
	}

	ids, _ := data.ValuesForPath("data.metaobjectDelete.deletedId")
	if len(ids) == 0 || ids[0] == nil {
		return "", errors.New("not found or access denied")
	}

	return fmt.Sprint(ids[0]), nil
}

// metaobjectID returns the id at the given path of a metaobject mutation
// response.
func metaobjectID(data mxj.Map, path string) (string, error) {
	ids, _ := data.ValuesForPath(path)
	if len(ids) == 0 || ids[0] == nil {
		return "", errors.New("no id returned")
	}

	return fmt.Sprint(ids[0]), nil
}

// userErrorMessages returns the messages of the userErrors at the given path,
// prefixed by their field when they have one, sorted so error output is stable.
func userErrorMessages(data mxj.Map, path string) []string {
	userErrors, _ := data.ValuesForPath(path)

	messages := make([]string, 0, len(userErrors))
	for _, ue := range userErrors {
		m := ue.(map[string]interface{})
		field := fmt.Sprint(m["field"])
		if field != "" && field != "<nil>" {
			messages = append(messages, fmt.Sprintf("%s: %s", field, m["message"]))
		} else {
			messages = append(messages, fmt.Sprint(m["message"]))
		}
	}
	sort.Strings(messages)

	return messages
}
