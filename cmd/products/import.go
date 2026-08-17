package products

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/cheynewallace/tabby"
	"github.com/urfave/cli/v2"

	"github.com/ScreenStaring/shopify-dev-tools/cmd"
	"github.com/ScreenStaring/shopify-dev-tools/cmd/products/gql"
)

func syncImportProducts(c *cli.Context) error {
	if c.NArg() < 1 {
		return fmt.Errorf("CSV file path required")
	}

	csvFile := c.Args().First()
	shop := c.String("shop")
	token := cmd.LookupAccessToken(shop, c.String("access-token"))
	options := map[string]interface{}{"version": c.String("api-version")}
	parallel := c.Int("parallel")
	jsonOutput := c.Bool("json")

	out := os.Stdout
	if jsonOutput {
		out = os.Stderr
	}

	locations, err := gql.FetchLocations(shop, token, options)
	if err != nil {
		return err
	}

	fmt.Fprintf(out, "Parsing %s...\n", csvFile)

	products, err := parseCSV(csvFile, locations)
	if err != nil {
		return err
	}

	if len(products) == 0 {
		return fmt.Errorf("No products found in CSV. Does the identifier column exist?")
	}

	setProductIdentifiers(products, c.String("identify-by"))

	fmt.Fprintf(out, "Importing %d products...\n", len(products))

	type importResult struct {
		Row    int
		ID     string
		Errors []string
		Err    error
	}

	results := make([]importResult, len(products))
	sem := make(chan struct{}, parallel)
	var wg sync.WaitGroup

	for i, p := range products {
		wg.Add(1)
		go func(idx int, product importProductInput) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			results[idx] = importResult{Row: idx + 1}

			b, err := json.Marshal(product)
			if err != nil {
				results[idx].Err = err
				return
			}

			var variables map[string]interface{}
			if err := json.Unmarshal(b, &variables); err != nil {
				results[idx].Err = err
				return
			}

			result, err := gql.ProductSet(shop, token, variables, options)
			if err != nil {
				results[idx].Err = err
				return
			}

			results[idx].ID = strings.TrimPrefix(result.ProductID, "gid://shopify/Product/")
			results[idx].Errors = result.UserErrors
		}(i, p)
	}

	wg.Wait()

	var failures int

	if jsonOutput {
		jsonResults := make([]map[string]interface{}, 0, len(results))

		for _, r := range results {
			item := map[string]interface{}{"row": r.Row, "id": r.ID, "status": "ok"}

			if r.Err != nil {
				failures++
				item["status"] = "error"
				item["errors"] = []string{r.Err.Error()}
			} else if len(r.Errors) > 0 {
				failures++
				item["status"] = "error"
				item["errors"] = r.Errors
			}

			jsonResults = append(jsonResults, item)
		}

		b, err := json.MarshalIndent(jsonResults, "", "  ")
		if err != nil {
			return err
		}

		fmt.Println(string(b))
	} else {
		fmt.Fprintln(out, "Done!\n")

		t := tabby.New()
		t.AddHeader("Row", "Product", "Status")

		for _, r := range results {
			if r.Err != nil {
				failures++
				t.AddLine(r.Row, "", "Error: "+r.Err.Error())
			} else if len(r.Errors) > 0 {
				failures++
				t.AddLine(r.Row, r.ID, "Error: "+strings.Join(r.Errors, "; "))
			} else {
				t.AddLine(r.Row, r.ID, "OK")
			}
		}
		t.Print()
	}

	if failures > 0 {
		return cli.Exit("", 1)
	}

	return nil
}
