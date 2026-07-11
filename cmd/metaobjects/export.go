package metaobjects

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/ScreenStaring/shopify-dev-tools/cmd"
	"github.com/ScreenStaring/shopify-dev-tools/cmd/metaobjects/gql"
	"github.com/urfave/cli/v2"
)

func exportBaseName(shop, moType string) string {
	return strings.SplitN(shop, ".", 2)[0] + "-" + moType
}

func exportAction(c *cli.Context) error {
	if c.NArg() == 0 {
		return errors.New("Metaobject type required")
	}

	shop := c.String("shop")
	token := cmd.LookupAccessToken(shop, c.String("access-token"))
	moType := c.Args().Get(0)
	verbose := c.Bool("verbose")
	query := c.String("query")

	if c.Bool("jsonl") {
		return exportJSONL(shop, token, moType, query, verbose)
	}

	return exportCSV(shop, token, moType, query, verbose)
}

func metaobjectFieldMap(m gql.Metaobject) map[string]string {
	fields := make(map[string]string, len(m.Fields))
	for _, f := range m.Fields {
		fields[f.Key] = f.Value
	}
	return fields
}

func exportJSONL(shop, token, moType, query string, verbose bool) error {
	filename := exportBaseName(shop, moType) + ".jsonl"

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("Cannot create JSONL file: %s", err)
	}
	defer file.Close()

	count := 0
	err = gql.FetchAllMetaobjects(shop, token, moType, query, verbose, func(m gql.Metaobject) error {
		record := map[string]interface{}{
			"id":           strings.TrimPrefix(m.ID, "gid://shopify/Metaobject/"),
			"handle":       m.Handle,
			"type":         m.Type,
			"display_name": m.DisplayName,
			"updated_at":   m.UpdatedAt,
			"fields":       metaobjectFieldMap(m),
		}

		line, err := json.Marshal(record)
		if err != nil {
			return fmt.Errorf("Cannot encode metaobject %s: %s", m.ID, err)
		}

		if _, err := file.Write(append(line, '\n')); err != nil {
			return fmt.Errorf("Cannot write metaobject %s: %s", m.ID, err)
		}

		count++
		fmt.Fprintf(os.Stderr, "\rExported %d", count)

		return nil
	})

	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "\nWrote %s\n", filename)

	return nil
}

func exportCSV(shop, token, moType, query string, verbose bool) error {
	var metaobjects []gql.Metaobject

	err := gql.FetchAllMetaobjects(shop, token, moType, query, verbose, func(m gql.Metaobject) error {
		metaobjects = append(metaobjects, m)
		fmt.Fprintf(os.Stderr, "\rFetched %d", len(metaobjects))
		return nil
	})

	if err != nil {
		return err
	}

	fieldKeySet := make(map[string]bool)
	for _, m := range metaobjects {
		for _, f := range m.Fields {
			fieldKeySet[f.Key] = true
		}
	}

	fieldKeys := make([]string, 0, len(fieldKeySet))
	for k := range fieldKeySet {
		fieldKeys = append(fieldKeys, k)
	}
	sort.Strings(fieldKeys)

	filename := exportBaseName(shop, moType) + ".csv"

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("Cannot create CSV file: %s", err)
	}
	defer file.Close()

	w := csv.NewWriter(file)

	header := append([]string{"ID", "Handle", "Type", "Display Name", "Updated At"}, fieldKeys...)
	if err := w.Write(header); err != nil {
		return fmt.Errorf("Cannot write CSV header: %s", err)
	}

	for _, m := range metaobjects {
		values := metaobjectFieldMap(m)

		row := append([]string{
			strings.TrimPrefix(m.ID, "gid://shopify/Metaobject/"),
			m.Handle,
			m.Type,
			m.DisplayName,
			m.UpdatedAt,
		}, make([]string, len(fieldKeys))...)

		for i, key := range fieldKeys {
			row[5+i] = values[key]
		}

		if err := w.Write(row); err != nil {
			return fmt.Errorf("Cannot write CSV row for %s: %s", m.ID, err)
		}
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "\nWrote %s\n", filename)

	return nil
}
