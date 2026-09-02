package gql

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/cheynewallace/tabby"
	"github.com/clbanning/mxj"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/parser"
)

type Client struct {
	endpoint  string
	token     string
	costDebug bool
	verbose   bool
}

// We omit the "/" after API for the case where there's no version.
const endpoint = "https://%s.myshopify.com/admin/api%s/graphql.json"

// DefaultAPIVersion is used when NewClient is called without a "version"
// option. Set once per process (the CLI sets it from --api-version).
var DefaultAPIVersion string

const (
	// initialRetryDelay is the wait before the first retry; it doubles per attempt.
	initialRetryDelay = 500 * time.Millisecond
)

// maxAttempts is the total number of request attempts, the original plus
// retries. It defaults to 10 and can be changed with the
// SDT_MAX_RETRY_ATTEMPTS environment variable; an unset, empty, or invalid
// value falls back to the default.
func maxAttempts() int {
	const defaultAttempts = 10

	v := os.Getenv("SDT_MAX_RETRY_ATTEMPTS")
	if v == "" {
		return defaultAttempts
	}

	n, err := strconv.Atoi(v)
	if err != nil || n < 1 {
		return defaultAttempts
	}

	return n
}

func NewClient(shop, token string, options ...map[string]interface{}) *Client {
	opts := map[string]interface{}{}
	if len(options) > 0 {
		opts = options[0]
	}

	version, _ := opts["version"].(string)
	if len(version) == 0 {
		version = DefaultAPIVersion
	}
	if len(version) > 0 {
		version = "/" + version
	}

	// allow for NAME.myshopify.com or just NAME
	shop = strings.SplitN(shop, ".", 2)[0]

	extras, _ := opts["extras"].(bool)
	verbose, _ := opts["verbose"].(bool)

	return &Client{
		endpoint:  fmt.Sprintf(endpoint, shop, version),
		token:     token,
		costDebug: extras,
		verbose:   verbose,
	}
}

func containsMutation(query string) bool {
	doc, err := parser.ParseQuery(&ast.Source{Input: query})
	if err != nil {
		return false
	}
	for _, op := range doc.Operations {
		if op.Operation == ast.Mutation {
			return true
		}
	}
	return false
}

func (c *Client) Execute(q string, variables ...map[string]interface{}) (mxj.Map, error) {
	readonly := os.Getenv("SDT_READONLY")
	if (readonly == "1" || readonly == "true") && containsMutation(q) {
		return nil, fmt.Errorf("Mutation not allowed in read-only mode (SDT_READONLY environment variable is set)")
	}

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

	// Retrying a mutation could apply it twice, so only queries are retried.
	retryable := !containsMutation(gql)

	client := http.Client{}
	attempts := maxAttempts()

	for attempt := 0; ; attempt++ {
		result, retryAfter, err := c.roundTrip(client, body, gql)
		if err == nil {
			return result, nil
		}
		if !retryable || retryAfter < 0 || attempt >= attempts-1 {
			return result, err
		}

		delay := initialRetryDelay << attempt
		if retryAfter > delay {
			delay = retryAfter
		}
		fmt.Fprintf(os.Stderr, "GraphQL request to %s failed (%s); retrying in %s\n", c.endpoint, err, delay)
		time.Sleep(delay)
	}
}

// roundTrip performs one HTTP request and parses the response. On failure it
// returns the suggested wait before retrying; a negative retryAfter means the
// failure is not retryable and zero means fall back to the default backoff.
func (c *Client) roundTrip(client http.Client, body, gql string) (mxj.Map, time.Duration, error) {
	var result mxj.Map

	req, err := http.NewRequest("POST", c.endpoint, strings.NewReader(body))
	if err != nil {
		return result, -1, fmt.Errorf("Failed to make GraphQL request to %s: %s", c.endpoint, err)
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-Shopify-Access-Token", c.token)
	if c.costDebug {
		req.Header.Add("Shopify-GraphQL-Cost-Debug", "1")
	}
	// Ask Shopify to include cost data so throttled responses can compute a
	// precise retry delay (see throttledRetryDelay).
	req.Header.Add("X-GraphQL-Cost-Include-Fields", "true")

	if c.verbose {
		fmt.Fprintf(os.Stderr, "> %s %s\n", req.Method, req.URL)
		for name, values := range req.Header {
			for _, v := range values {
				fmt.Fprintf(os.Stderr, "> %s: %s\n", name, v)
			}
		}
		fmt.Fprintf(os.Stderr, ">\n%s\n\n", body)
	}

	resp, err := client.Do(req)
	if err != nil {
		// Transport-level failure (connection refused, timeout): retryable.
		return result, 0, fmt.Errorf("GraphQL request to %s failed: %s", c.endpoint, err)
	}

	defer resp.Body.Close()
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		// The response body failed mid-stream; treat it like a transport error.
		return result, 0, fmt.Errorf("Failed to read GraphQL response from %s: %s", c.endpoint, err)
	}

	if resp.StatusCode != http.StatusOK {
		var msg error
		if len(bytes) > 0 {
			msg = fmt.Errorf("query failed with HTTP response code %d: %s", resp.StatusCode, string(bytes))
		} else {
			msg = fmt.Errorf("query failed with HTTP response code %d", resp.StatusCode)
		}

		// 429 and 5xx are transient server conditions; other 4xx errors are
		// client mistakes that retrying will not fix.
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			return result, retryDelay(resp), msg
		}
		return result, -1, msg
	}

	if c.verbose {
		fmt.Fprintf(os.Stderr, "< %s\n", resp.Status)
		for name, values := range resp.Header {
			for _, v := range values {
				fmt.Fprintf(os.Stderr, "< %s: %s\n", name, v)
			}
		}
		fmt.Fprintf(os.Stderr, "<\n%s\n\n", string(bytes))
	}

	// results in parse error
	//result, err = mxj.NewMapJsonReader(resp.Body)

	result, err = mxj.NewMapJson(bytes)
	if err != nil {
		return result, -1, fmt.Errorf("Failed to unmarshal GraphQL response body: %s", err)
	}

	if err := responseErrors(result); err != nil {
		return result, graphQLErrorRetryDelay(result), err
	}

	return result, 0, nil
}

// graphQLErrorRetryDelay returns the wait suggested by GraphQL-level errors:
// THROTTLED responses carry the time needed for the query cost to become
// available, and TIMEOUT / INTERNAL_SERVER_ERROR fall back to the default
// backoff. Any other GraphQL error is not retryable and yields -1.
func graphQLErrorRetryDelay(result mxj.Map) time.Duration {
	errors, _ := result["errors"].([]interface{})
	for _, e := range errors {
		eMap, ok := e.(map[string]interface{})
		if !ok {
			continue
		}

		extensions, ok := eMap["extensions"].(map[string]interface{})
		if !ok {
			continue
		}

		switch extensions["code"] {
		case "THROTTLED":
			return throttledRetryDelay(extensions)
		case "TIMEOUT", "INTERNAL_SERVER_ERROR":
			return 0
		}
	}

	return -1
}

// throttledRetryDelay computes the sleep needed for a throttled query to
// become available from the error's extensions.cost block, mirroring
// Shopify's documented calculation. It returns zero (use the default backoff)
// when the cost data is absent or unusable.
func throttledRetryDelay(extensions map[string]interface{}) time.Duration {
	cost, ok := extensions["cost"].(map[string]interface{})
	if !ok {
		return 0
	}

	// A cost block with actualQueryCost describes a normal response, not a
	// throttled one.
	if _, ok := numVal(cost["actualQueryCost"]); ok {
		return 0
	}

	status, ok := cost["throttleStatus"].(map[string]interface{})
	if !ok {
		return 0
	}

	requested, ok := numVal(cost["requestedQueryCost"])
	if !ok {
		return 0
	}
	available, ok := numVal(status["currentlyAvailable"])
	if !ok {
		return 0
	}
	restoreRate, ok := numVal(status["restoreRate"])
	if !ok || restoreRate <= 0 {
		return 0
	}

	wait := (requested - available) / restoreRate
	if wait <= 0 {
		return 0
	}
	return time.Duration(wait * float64(time.Second))
}

// numVal converts a JSON number decoded by mxj (int, int64, or float64) to a
// float64.
func numVal(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	}
	return 0, false
}

// retryDelay parses the Retry-After header (seconds or an HTTP date) and
// returns zero when it is absent or unparseable, letting the caller use its
// own backoff.
func retryDelay(resp *http.Response) time.Duration {
	if v := resp.Header.Get("Retry-After"); v != "" {
		if seconds, err := strconv.Atoi(v); err == nil {
			return time.Duration(seconds) * time.Second
		}
	}
	return 0
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
