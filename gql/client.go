package gql

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	_ "github.com/cheynewallace/tabby"

	"github.com/clbanning/mxj"
)

type Client struct {
	endpoint string
	token    string
}

// We omit the "/" after API for the case where there's no version.
const endpoint = "https://%s.myshopify.com/admin/api%s/graphql.json"

func NewClient(shop, token, version string) *Client {
	if len(version) > 0 {
		version = "/" + version
	}

	// allow for NAME.myshopify.com or just NAME
	shop = strings.SplitN(shop, ".", 2)[0]

	return &Client{endpoint: fmt.Sprintf(endpoint, shop, version), token: token}
}

func (c *Client) Execute(q string, variables ...map[string]interface{}) (mxj.Map, error) {
	merged := map[string]interface{}{}
	for _, v := range variables {
		for k, val := range v {
			merged[k] = val
		}
	}
	return c.request(q, merged)
}

func (c *Client) request(gql string, variables map[string]interface{}) (mxj.Map, error) {
	var result mxj.Map

	body, err := c.createRequestBody(gql, variables)
	if err != nil {
		return result, fmt.Errorf("Failed to marshal GraphQL request body: %s", err)
	}

	client := http.Client{}

	req, err := http.NewRequest("POST", c.endpoint, strings.NewReader(string(body)))
	if err != nil {
		return result, fmt.Errorf("Failed to make GraphQL request: %s", c.endpoint, err)
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-Shopify-Access-Token", c.token)

	resp, err := client.Do(req)
	if err != nil {
		return result, fmt.Errorf("GraphQL request failed: %s", c.endpoint, err)
	}

	defer resp.Body.Close()
	bytes, err := ioutil.ReadAll(resp.Body)

	// results in parse error
	//result, err = mxj.NewMapJsonReader(resp.Body)

	result, err = mxj.NewMapJson(bytes)
	if err != nil {
		return result, fmt.Errorf("Failed to unmarshal GraphQL response body: %s", err)
	}

	if err := responseErrors(result); err != nil {
		return result, err
	}

	return result, nil
}

func responseErrors(result mxj.Map) error {
	errors, _ := result.ValuesForPath("errors")
	if len(errors) == 0 {
		return nil
	}

	var messages []string
	for _, e := range errors {
		eMap, ok := e.(map[string]interface{})
		if !ok {
			messages = append(messages, fmt.Sprint(e))
			continue
		}

		message := fmt.Sprint(eMap["message"])

		if path, ok := eMap["path"]; ok {
			items, ok := path.([]interface{})
			if ok {
				parts := make([]string, len(items))
				for i, p := range items {
					parts[i] = fmt.Sprint(p)
				}
				message += fmt.Sprintf(" at %s", strings.Join(parts, "."))
			}
		}

		messages = append(messages, message)
	}

	return fmt.Errorf("%s", strings.Join(messages, ", "))
}

func (c *Client) createRequestBody(query string, variables map[string]interface{}) (string, error) {
	params := map[string]interface{}{"query": query}

	if len(variables) > 0 {
		params["variables"] = variables
	}

	body, err := json.Marshal(params)
	if err != nil {
		return "", err
	}

	return string(body), nil
}
