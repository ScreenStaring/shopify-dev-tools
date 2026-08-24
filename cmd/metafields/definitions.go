package metafields

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/ScreenStaring/shopify-dev-tools/cmd"
	"github.com/ScreenStaring/shopify-dev-tools/gql"
	"github.com/urfave/cli/v2"
)

// CSV column names for metafield definitions import. Columns are matched by
// name, case-insensitively, so order in the spreadsheet does not matter.
// Unknown columns are rejected.
const (
	colOwnerType             = "owner type"
	colName                  = "name"
	colKey                   = "key"
	colType                  = "type"
	colNamespace             = "namespace"
	colDescription           = "description"
	colPin                   = "pin"
	colAccessAdmin           = "access admin"
	colAccessCustomerAccount = "access customer account"
	colAccessStorefront      = "access storefront"
	colCapAdminFilterable    = "capability admin filterable"
	colCapSmartCollection    = "capability smart collection condition"
	colCapUniqueValues       = "capability unique values"
	colConstraintsKey        = "constraint key"
	colConstraintsValues     = "constraint value"
	colValidationName        = "validation name"
	colValidationValue       = "validation value"
)

// Required columns; importing without them is an error.
var requiredColumns = []string{colOwnerType, colName, colKey, colType}

// definitionInput is one metafield definition to create, assembled from one
// or more spreadsheet rows (validation rows merge onto their definition).
type definitionInput struct {
	OwnerType         string
	Name              string
	Key               string
	Type              string
	Namespace         string
	Desc              string
	Pin               bool
	Access            map[string]string // admin, customerAccount, storefront
	Capability        map[string]bool   // adminFilterable, smartCollectionCondition, uniqueValues
	ConstraintsKey    string
	ConstraintsValues []string
	Validations       []validationInput
}

type validationInput struct {
	Name  string
	Value string
}

func importDefinitionsAction(c *cli.Context) error {
	jsonOutput := c.Bool("json")

	var reader io.Reader
	if c.NArg() == 0 {
		reader = os.Stdin
	} else {
		f, err := os.Open(c.Args().Get(0))
		if err != nil {
			return err
		}
		defer f.Close()
		reader = f
	}

	defs, err := parseDefinitionCSV(reader)
	if err != nil {
		return err
	}

	if len(defs) == 0 {
		return errors.New("No definitions found in input")
	}

	client := cmd.NewGraphQLClient(c)

	created, failed := 0, 0
	jsonResults := make([]map[string]interface{}, 0, len(defs))

	for n, d := range defs {
		id, err := createDefinition(client, d)
		if err != nil {
			failed++
			if jsonOutput {
				jsonResults = append(jsonResults, map[string]interface{}{
					"row":       n + 2,
					"namespace": d.Namespace,
					"key":       d.Key,
					"id":        "",
					"status":    "error",
					"errors":    []string{err.Error()},
				})
			} else {
				fmt.Fprintf(os.Stderr, "Row %d %s.%s: Error: %s\n", n+2, d.Namespace, d.Key, err)
			}
			continue
		}
		created++
		id = strings.TrimPrefix(id, "gid://shopify/MetafieldDefinition/")
		if jsonOutput {
			jsonResults = append(jsonResults, map[string]interface{}{
				"row":       n + 2,
				"namespace": d.Namespace,
				"key":       d.Key,
				"id":        id,
				"status":    "ok",
			})
		} else {
			fmt.Printf("Row %3d %s.%s: Created (%s)\n", n+2, d.Namespace, d.Key, id)
		}
	}

	if jsonOutput {
		b, err := json.MarshalIndent(jsonResults, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(b))
		if failed > 0 {
			return cli.Exit("", 1)
		}
		return nil
	}

	if failed > 0 {
		return fmt.Errorf("%d of %d definitions created, %d failed", created, len(defs), failed)
	}

	fmt.Printf("%d definition(s) created\n", created)
	return nil
}

// parseDefinitionCSV reads definitions from a CSV spreadsheet, one row per
// definition. Multiple validations per definition come from repeated
// validation name/validation value column pairs.
func parseDefinitionCSV(reader io.Reader) ([]*definitionInput, error) {
	r := csv.NewReader(reader)
	r.FieldsPerRecord = -1
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("Invalid CSV: %s", err)
	}
	if len(records) < 2 {
		return nil, errors.New("CSV must have a header row and at least one data row")
	}

	// Map header names to column indexes. Multiple validations are given as
	// repeated validation name/validation value column pairs; the k-th
	// validation name column pairs with the k-th validation value column.
	// All other duplicate columns are rejected.
	cols := make(map[string]int, len(records[0]))
	var valNameIdx, valValueIdx []int
	for i, h := range records[0] {
		h = strings.ToLower(strings.TrimSpace(h))
		if h == "" {
			continue
		}
		switch h {
		case colValidationName:
			valNameIdx = append(valNameIdx, i)
			continue
		case colValidationValue:
			valValueIdx = append(valValueIdx, i)
			continue
		}
		if _, dup := cols[h]; dup {
			return nil, fmt.Errorf("Duplicate column '%s' in header", h)
		}
		cols[h] = i
	}

	if len(valNameIdx) != len(valValueIdx) {
		return nil, fmt.Errorf("Number of %s and %s columns must match", colValidationName, colValidationValue)
	}

	for _, col := range requiredColumns {
		if _, ok := cols[col]; !ok {
			return nil, fmt.Errorf("Missing required column '%s'", col)
		}
	}

	// Reject unknown columns so typos don't silently drop data.
	for h := range cols {
		if !knownColumn(h) {
			return nil, fmt.Errorf("Unknown column '%s'", h)
		}
	}

	var defs []*definitionInput

	for n, row := range records[1:] {
		cell := func(col string) string { return strings.TrimSpace(getCell(row, cols, col)) }
		cellAt := func(i int) string {
			if i >= len(row) {
				return ""
			}
			return strings.TrimSpace(row[i])
		}

		d := &definitionInput{
			// Owner type is case-insensitive; the API requires the uppercase enum.
			OwnerType:  strings.ToUpper(cell(colOwnerType)),
			Name:       cell(colName),
			Key:        cell(colKey),
			Type:       cell(colType),
			Namespace:  cell(colNamespace),
			Desc:       cell(colDescription),
			Access:     map[string]string{},
			Capability: map[string]bool{},
		}

		// Parse the validation column pairs for this row. The validation value
		// is passed through verbatim (it may itself contain "|", e.g. regex).
		for k := range valNameIdx {
			name, value := cellAt(valNameIdx[k]), cellAt(valValueIdx[k])
			if name != "" || value != "" {
				if name == "" || value == "" {
					return nil, fmt.Errorf("Row %d: %s/%s pair %d must have both name and value", n+2, colValidationName, colValidationValue, k+1)
				}
				d.Validations = append(d.Validations, validationInput{Name: name, Value: value})
			}
		}

		for _, col := range requiredColumns {
			if cell(col) == "" {
				return nil, fmt.Errorf("Row %d: required column '%s' empty", n+2, col)
			}
		}

		pin, err := parseBool(cell(colPin))
		if err != nil {
			return nil, fmt.Errorf("Row %d: pin: %s", n+2, err)
		}
		d.Pin = pin

		capabilityCols := []struct{ col, key string }{
			{colCapAdminFilterable, "adminFilterable"},
			{colCapSmartCollection, "smartCollectionCondition"},
			{colCapUniqueValues, "uniqueValues"},
		}
		for _, cc := range capabilityCols {
			b, err := parseBool(cell(cc.col))
			if err != nil {
				return nil, fmt.Errorf("Row %d: %s: %s", n+2, cc.col, err)
			}
			d.Capability[cc.key] = b
		}

		// The API's access input has admin, customerAccount and storefront.
		d.Access["admin"] = cell(colAccessAdmin)
		d.Access["customerAccount"] = cell(colAccessCustomerAccount)
		d.Access["storefront"] = cell(colAccessStorefront)

		d.ConstraintsKey = cell(colConstraintsKey)
		if values := cell(colConstraintsValues); values != "" {
			d.ConstraintsValues = splitPipe(values)
		}

		if d.ConstraintsKey == "" && len(d.ConstraintsValues) > 0 {
			return nil, fmt.Errorf("Row %d: %s given without %s", n+2, colConstraintsValues, colConstraintsKey)
		}
		if d.ConstraintsKey != "" && len(d.ConstraintsValues) == 0 {
			return nil, fmt.Errorf("Row %d: %s given without %s", n+2, colConstraintsKey, colConstraintsValues)
		}

		defs = append(defs, d)
	}

	return defs, nil
}

func parseBool(s string) (bool, error) {
	switch strings.ToLower(s) {
	case "", "false":
		return false, nil
	case "true":
		return true, nil
	}
	return false, fmt.Errorf("'%s' is not a valid boolean (use true/false)", s)
}

func splitPipe(s string) []string {
	parts := strings.Split(s, "|")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func getCell(row []string, cols map[string]int, name string) string {
	i, ok := cols[name]
	if !ok || i >= len(row) {
		return ""
	}
	return row[i]
}

func knownColumn(name string) bool {
	switch name {
	case colOwnerType, colName, colKey, colType, colNamespace, colDescription,
		colPin, colAccessAdmin, colAccessCustomerAccount, colAccessStorefront,
		colCapAdminFilterable, colCapSmartCollection, colCapUniqueValues,
		colConstraintsKey, colConstraintsValues:
		return true
	}
	return false
}

const metafieldDefinitionCreateMutation = `
mutation metafieldDefinitionCreate($definition: MetafieldDefinitionInput!) {
  metafieldDefinitionCreate(definition: $definition) {
    createdDefinition {
      id
      name
      namespace
      key
      ownerType
    }
    userErrors {
      field
      message
    }
  }
}
`

// definitionGID accepts either a numeric id or a full GID and returns the
// GID form the API requires.
func definitionGID(id string) string {
	if _, err := strconv.ParseInt(id, 10, 64); err == nil {
		return "gid://shopify/MetafieldDefinition/" + id
	}
	return id
}

const metafieldDefinitionDeleteMutation = `
mutation metafieldDefinitionDelete($id: ID!, $deleteAllAssociatedMetafields: Boolean!) {
  metafieldDefinitionDelete(id: $id, deleteAllAssociatedMetafields: $deleteAllAssociatedMetafields) {
    deletedDefinitionId
    userErrors {
      field
      message
    }
  }
}
`

func deleteDefinitionAction(c *cli.Context) error {
	if c.NArg() == 0 {
		return errors.New("Metafield definition id required")
	}

	client := cmd.NewGraphQLClient(c)
	deleteAll := !c.Bool("no-metafields")

	var failures []string
	for _, arg := range c.Args().Slice() {
		id := definitionGID(arg)
		data, err := client.Execute(metafieldDefinitionDeleteMutation, map[string]interface{}{
			"id":                            id,
			"deleteAllAssociatedMetafields": deleteAll,
		})
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: %s", arg, err))
			continue
		}

		userErrors, _ := data.ValuesForPath("data.metafieldDefinitionDelete.userErrors")
		if len(userErrors) > 0 {
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
			failures = append(failures, fmt.Sprintf("%s: %s", arg, strings.Join(messages, "; ")))
			continue
		}

		deletedID, _ := data.ValuesForPath("data.metafieldDefinitionDelete.deletedDefinitionId")
		if len(deletedID) == 0 || deletedID[0] == nil {
			failures = append(failures, fmt.Sprintf("%s: not found or access denied", arg))
			continue
		}

		fmt.Printf("Deleted %s\n", fmt.Sprint(deletedID[0]))
	}

	if len(failures) > 0 {
		return fmt.Errorf("Cannot delete metafield definition(s): %s", strings.Join(failures, ", "))
	}

	return nil
}

// createDefinition creates one metafield definition and returns the created
// definition's id.
func createDefinition(client *gql.Client, d *definitionInput) (string, error) {
	definition := map[string]interface{}{
		"ownerType": d.OwnerType,
		"name":      d.Name,
		"key":       d.Key,
		"type":      d.Type,
	}

	if d.Namespace != "" {
		definition["namespace"] = d.Namespace
	}
	if d.Desc != "" {
		definition["description"] = d.Desc
	}
	if d.Pin {
		definition["pin"] = true
	}

	access := map[string]interface{}{}
	accessCols := []struct{ key, col string }{
		{"admin", colAccessAdmin},
		{"customerAccount", colAccessCustomerAccount},
		{"storefront", colAccessStorefront},
	}
	for _, ac := range accessCols {
		if v := d.Access[ac.key]; v != "" {
			access[ac.key] = v
		}
	}
	if len(access) > 0 {
		definition["access"] = access
	}

	capabilities := map[string]interface{}{}
	for k, v := range d.Capability {
		if v {
			capabilities[k] = map[string]interface{}{"enabled": true}
		}
	}
	if len(capabilities) > 0 {
		definition["capabilities"] = capabilities
	}

	// Both key and values are required by MetafieldDefinitionConstraintsInput;
	// parseDefinitionCSV already rejects a key without values.
	if d.ConstraintsKey != "" && len(d.ConstraintsValues) > 0 {
		definition["constraints"] = map[string]interface{}{
			"key":    d.ConstraintsKey,
			"values": d.ConstraintsValues,
		}
	}

	if len(d.Validations) > 0 {
		validations := make([]map[string]interface{}, 0, len(d.Validations))
		for _, v := range d.Validations {
			validations = append(validations, map[string]interface{}{
				"name":  v.Name,
				"value": v.Value,
			})
		}
		definition["validations"] = validations
	}

	data, err := client.Execute(metafieldDefinitionCreateMutation, map[string]interface{}{
		"definition": definition,
	})
	if err != nil {
		return "", err
	}

	userErrors, _ := data.ValuesForPath("data.metafieldDefinitionCreate.userErrors")
	if len(userErrors) > 0 {
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
		return "", errors.New(strings.Join(messages, "; "))
	}

	ids, _ := data.ValuesForPath("data.metafieldDefinitionCreate.createdDefinition.id")
	if len(ids) == 0 || ids[0] == nil {
		return "", errors.New("no id returned")
	}

	return fmt.Sprint(ids[0]), nil
}
