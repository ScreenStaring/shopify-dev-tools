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

const endpoint = "https://%s.myshopify.com/admin/api/2021-07/graphql.json"

func NewClient(shop, token string) *Client {
	return &Client{endpoint: fmt.Sprintf(endpoint, shop), token: token}
}

func (c *Client) Query(q string) (mxj.Map, error) {
	return c.request(q, nil)
}

func (c *Client) Mutation(q string, variables map[string]interface{}) (mxj.Map, error) {
	return c.request(q, variables)
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

	return result, nil
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
