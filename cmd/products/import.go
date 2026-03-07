package products

import (
	"encoding/json"
	"fmt"
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
	parallel := c.Int("parallel")

	fmt.Printf("Parsing %s...\n", csvFile)

	products, err := parseCSV(csvFile)
	if err != nil {
		return err
	}

	if len(products) == 0 {
		return fmt.Errorf("No products found in CSV")
	}

	setProductIdentifiers(products, c.String("identify-by"))

	fmt.Printf("Importing %d products...\n", len(products))

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

			result, err := gql.ProductSet(shop, token, variables)
			if err != nil {
				results[idx].Err = err
				return
			}

			results[idx].ID = result.ProductID
			results[idx].Errors = result.UserErrors
		}(i, p)
	}

	wg.Wait()

	fmt.Println("Done!\n")

	t := tabby.New()
	t.AddHeader("Row", "Product", "Status")

	var failures int
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

	if failures > 0 {
		return cli.Exit("", 1)
	}

	return nil
}
