package products

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cheynewallace/tabby"
	"github.com/urfave/cli/v2"

	"github.com/ScreenStaring/shopify-dev-tools/cmd"
	"github.com/ScreenStaring/shopify-dev-tools/gql"
)

var Cmd cli.Command

// func createProducts(c *cli.Context) error {
// 	if c.NArg() == 0 {
// 		return errors.New("CSV required")
// 	}

// 	return nil
// }

type listProductOptions struct {
	Fields       []string  `url:"fields,comma,omitempty"`
	Ids          []int64   `url:"ids,comma,omitempty"`
	Limit        int64     `url:"limit,omitempty"`
	Status       string    `url:"status,omitempty"`
	UpdatedAtMin time.Time `url:"updated_at_min,omitempty"`
}

func buildQuery(options listProductOptions) string {
	var queryType string
	var args []string
	var fields string

	// Determine fields to return
	if len(options.Fields) > 0 {
		fields = strings.Join(options.Fields, " ")
	} else {
		// Default fields that match the original REST API response
		fields = "id title handle status vendor productType createdAt updatedAt"
	}

	if len(options.Ids) > 0 {
		// Query specific products by IDs
		idStrings := make([]string, len(options.Ids))
		for i, id := range options.Ids {
			idStrings[i] = fmt.Sprintf(`"gid://shopify/Product/%d"`, id)
		}
		args = append(args, fmt.Sprintf("ids: [%s]", strings.Join(idStrings, ", ")))
		queryType = "products"
	} else {
		// Query products with filters
		if options.Limit > 0 {
			args = append(args, fmt.Sprintf("first: %d", options.Limit))
		} else {
			args = append(args, "first: 10") // Default limit
		}
		
		if options.Status != "" {
			// Map REST API status values to GraphQL
			switch strings.ToUpper(options.Status) {
			case "ACTIVE":
				args = append(args, `query: "status:active"`)
			case "DRAFT":
				args = append(args, `query: "status:draft"`)
			case "ARCHIVED":
				args = append(args, `query: "status:archived"`)
			}
		}
		queryType = "products"
	}

	argsStr := ""
	if len(args) > 0 {
		argsStr = fmt.Sprintf("(%s)", strings.Join(args, ", "))
	}

	if len(options.Ids) > 0 {
		return fmt.Sprintf(`{
			%s%s {
				nodes {
					%s
				}
			}
		}`, queryType, argsStr, fields)
	} else {
		return fmt.Sprintf(`{
			%s%s {
				edges {
					node {
						%s
					}
				}
			}
		}`, queryType, argsStr, fields)
	}
}

func printJSONL(products []map[string]interface{}) {
	for _, product := range products {
		line, err := json.Marshal(product)
		if err != nil {
			panic(err)
		}

		fmt.Println(string(line))
	}
}

func normalizeField(name string) string {
	return strings.ReplaceAll(strings.ToLower(name), " ", "")
}

func isFieldToPrint(field string, selectedFields []string) bool {
	for _, f := range selectedFields {
		if f == field {
			return true
		}
	}

	return false
}

func printFormatted(products []map[string]interface{}, fieldsToPrint []string) {
	t := tabby.New()
	normalizedFieldsToPrint := []string{}

	for _, field := range fieldsToPrint {
		normalizedFieldsToPrint = append(normalizedFieldsToPrint, normalizeField(field))
	}

	for _, product := range products {
		for key, value := range product {
			normalizedField := normalizeField(key)

			if len(fieldsToPrint) > 0 {
				if isFieldToPrint(normalizedField, normalizedFieldsToPrint) {
					t.AddLine(key, value)
				}
			} else {
				t.AddLine(key, value)
			}
		}

		t.Print()
		fmt.Printf("%s\n", strings.Repeat("-", 20))
	}
}

func listProducts(c *cli.Context) error {
	var options listProductOptions

	if c.NArg() > 0 {
		for i := 0; i < c.NArg(); i++ {
			id, err := strconv.ParseInt(c.Args().Get(i), 10, 64)
			if err != nil {
				return fmt.Errorf("Product id '%s' invalid: must be an int", c.Args().Get(0))
			}

			options.Ids = append(options.Ids, id)
		}

	} else {
		if len(c.String("status")) > 0 {
			options.Status = c.String("status")
		}

		if c.Int64("limit") > 0 {
			options.Limit = c.Int64("limit")
		}
	}

	if len(c.String("fields")) > 0 {
		options.Fields = strings.Split(c.String("fields"), ",")
	}

	// Create GraphQL client
	shop := c.String("shop")
	client := gql.NewClient(shop, cmd.LookupAccessToken(shop, c.String("access-token")), c.String("api-version"))

	// Build and execute GraphQL query
	query := buildQuery(options)
	result, err := client.Query(query)
	if err != nil {
		return fmt.Errorf("Cannot list products: %s", err)
	}

	// Parse GraphQL response
	products, err := parseProductsResponse(result, len(options.Ids) > 0)
	if err != nil {
		return fmt.Errorf("Cannot parse products response: %s", err)
	}

	if c.Bool("jsonl") {
		printJSONL(products)
	} else {
		printFormatted(products, options.Fields)
	}

	return nil
}

func parseProductsResponse(result map[string]interface{}, byIds bool) ([]map[string]interface{}, error) {
	var products []map[string]interface{}

	// Check for GraphQL errors
	if errors, ok := result["errors"]; ok {
		return nil, fmt.Errorf("GraphQL errors: %v", errors)
	}

	// Navigate to the products data
	data, ok := result["data"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response structure: missing data")
	}

	productsData, ok := data["products"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response structure: missing products")
	}

	if byIds {
		// For queries by ID, products are returned in nodes array
		nodes, ok := productsData["nodes"].([]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid response structure: missing nodes")
		}

		for _, node := range nodes {
			if product, ok := node.(map[string]interface{}); ok {
				// Convert GraphQL ID to numeric ID
				if id, exists := product["id"]; exists {
					if idStr, ok := id.(string); ok {
						if numericId := extractNumericId(idStr); numericId != "" {
							product["id"] = numericId
						}
					}
				}
				products = append(products, product)
			}
		}
	} else {
		// For paginated queries, products are returned in edges array
		edges, ok := productsData["edges"].([]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid response structure: missing edges")
		}

		for _, edge := range edges {
			if edgeMap, ok := edge.(map[string]interface{}); ok {
				if node, ok := edgeMap["node"].(map[string]interface{}); ok {
					// Convert GraphQL ID to numeric ID
					if id, exists := node["id"]; exists {
						if idStr, ok := id.(string); ok {
							if numericId := extractNumericId(idStr); numericId != "" {
								node["id"] = numericId
							}
						}
					}
					products = append(products, node)
				}
			}
		}
	}

	return products, nil
}

func extractNumericId(gid string) string {
	// Extract numeric ID from GraphQL global ID like "gid://shopify/Product/123"
	parts := strings.Split(gid, "/")
	if len(parts) >= 4 && parts[0] == "gid:" && parts[1] == "" && parts[2] == "shopify" {
		return parts[len(parts)-1]
	}
	return gid
}

func init() {
	productFlags := []cli.Flag{
		// &cli.StringSliceFlag{
		// 	Name:    "order",
		// 	Aliases: []string{"o"},
		// 	Usage:   "Order products by the given properties",
		// },
		&cli.StringFlag{
			Name:    "fields",
			Aliases: []string{"f"},
			Usage:   "Comma separated list of fields to output",
			EnvVars: []string{"SHOPIFY_PRODUCT_FIELDS"},
		},
		&cli.Int64Flag{
			Name:    "limit",
			Aliases: []string{"l"},
			Value: 10,
		},
		&cli.StringFlag{
			Name:    "status",
			Aliases: []string{"s"},
		},
		&cli.BoolFlag{
			Name:    "jsonl",
			Aliases: []string{"j"},
			Usage:   "Output the products in JSONL format",
		},
		&cli.StringFlag{
			Name:    "api-version",
			Aliases: []string{"a"},
			Usage:   "API version to use; default is a versionless call",
		},
	}

	Cmd = cli.Command{
		Name:    "products",
		Aliases: []string{"p"},
		Usage:   "Do things with products",
		Subcommands: []*cli.Command{
			{
				Name:      "ls",
				Aliases:   []string{"l"},
				Usage:     "List some of a shop's products or the products given by the specified IDs",
				ArgsUsage: "[ID [ID ...]]",
				Flags:     append(cmd.Flags, productFlags...),
				Action:    listProducts,
			},
			// {
			// 	Name: "create",
			// 	Aliases: []string{"c"},
			// 	Usage:   "Create products from the give Shopify CSV",
			// 	Flags: cmd.Flags,
			// 	Action: createProducts,
			// },
		},
	}
}
