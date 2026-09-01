package export

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/urfave/cli/v2"

	"github.com/ScreenStaring/shopify-dev-tools/cmd"
	"github.com/ScreenStaring/shopify-dev-tools/cmd/products/gql"
)

// metafieldJSON is the JSON object emitted per metafield in JSONL output,
// matching the format of `metafields product -j`.
type metafieldJSON struct {
	ID          string `json:"id"`
	Namespace   string `json:"namespace"`
	Key         string `json:"key"`
	Description string `json:"description"`
	Value       string `json:"value"`
	Type        string `json:"type"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

// writeJSONL writes one JSON object per metafield, the same format as
// `metafields product -j`.
func writeJSONL(out io.Writer, pm gql.ProductMetafields) error {
	for _, mf := range pm.Metafields {
		line, err := json.Marshal(metafieldJSON{
			ID:          mf.ID,
			Namespace:   mf.Namespace,
			Key:         mf.Key,
			Description: mf.Description,
			Value:       mf.Value,
			Type:        mf.Type,
			CreatedAt:   mf.CreatedAt,
			UpdatedAt:   mf.UpdatedAt,
		})
		if err != nil {
			return err
		}

		if _, err := fmt.Fprintln(out, string(line)); err != nil {
			return err
		}
	}

	return nil
}

func Metafields(c *cli.Context) error {
	shop := c.String("shop")
	token := cmd.LookupAccessToken(shop, c.String("access-token"))
	namespace := c.String("namespace")
	key := c.String("key")
	baseName := shopBaseName(shop)

	options := map[string]interface{}{}

	total, err := gql.FetchProductCount(shop, token, "", options)
	if err != nil {
		return err
	}

	exportFormat := "CSV"
	if c.Bool("jsonl") {
		exportFormat = "JSONL"
	}

	fmt.Fprintf(os.Stderr, "Exporting metafields for %d products to %s...\n", total, exportFormat)

	var (
		csvOut      *csv.Writer
		csvFile     *os.File
		jsonlBuf    *bufio.Writer
		jsonlFile   *os.File
		header      = []string{"Product ID", "Product Title", "Product Type", "Handle", "Metafield ID", "Metafield Owner", "Metafield Type", "Metafield Namespace", "Metafield Key", "Metafield Value"}
		headerWrite = false
	)

	if c.Bool("jsonl") {
		jsonlFile, err = os.Create(baseName + "-metafields.jsonl")
		if err != nil {
			return fmt.Errorf("Failed to create JSONL file: %s", err)
		}
		jsonlBuf = bufio.NewWriter(jsonlFile)
	} else {
		csvFile, err = os.Create(baseName + "-metafields.csv")
		if err != nil {
			return fmt.Errorf("Failed to create CSV file: %s", err)
		}
		csvOut = csv.NewWriter(csvFile)
	}

	writeRow := func(pm gql.ProductMetafields) error {
		if !headerWrite {
			if err := csvOut.Write(header); err != nil {
				return err
			}
			headerWrite = true
		}

		for _, mf := range pm.Metafields {
			row := []string{
				strconv.FormatInt(pm.ProductID, 10),
				pm.ProductTitle,
				pm.ProductType,
				pm.Handle,
				mf.ID,
				"Product",
				mf.Type,
				mf.Namespace,
				mf.Key,
				mf.Value,
			}

			if err := csvOut.Write(row); err != nil {
				return fmt.Errorf("Cannot write metafield %s: %s", mf.ID, err)
			}
		}

		return nil
	}

	count := 0
	err = gql.FetchAllProductMetafields(shop, token, namespace, key, func(pm gql.ProductMetafields) error {
		var err error

		if jsonlBuf != nil {
			err = writeJSONL(jsonlBuf, pm)
		} else {
			err = writeRow(pm)
		}

		if err != nil {
			return err
		}

		count++
		fmt.Fprintf(os.Stderr, "\rProcessing %d/%d", count, total)

		return nil
	}, options)

	if err != nil {
		return err
	}

	if jsonlBuf != nil {
		jsonlBuf.Flush()
		jsonlFile.Close()
	} else {
		csvOut.Flush()
		if err := csvOut.Error(); err != nil {
			return err
		}
		csvFile.Close()
	}

	fmt.Fprintln(os.Stderr, "\nComplete!")

	return nil
}
