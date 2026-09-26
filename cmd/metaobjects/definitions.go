package metaobjects

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ScreenStaring/shopify-dev-tools/cmd"
	"github.com/ScreenStaring/shopify-dev-tools/cmd/metaobjects/gql"
	"github.com/urfave/cli/v2"
)

// CSV column names for metaobject definitions import. Columns are matched by
// name, case-insensitively, so order in the spreadsheet does not matter.
// Unknown columns are rejected.
const (
	colType                    = "type"
	colName                    = "name"
	colDescription             = "description"
	colDisplayNameKey          = "display name key"
	colAccessAdmin             = "access admin"
	colAccessCustomerAccount   = "access customer account"
	colAccessStorefront        = "access storefront"
	colCapPublishable          = "capability publishable"
	colCapOnlineStore          = "capability online store"
	colCapOnlineStoreURL       = "capability online store url handle"
	colCapOnlineStoreRedirects = "capability online store create redirects"
	colCapTranslatable         = "capability translatable"
	colCapRenderable           = "capability renderable"
	colCapRenderableMetaTitle  = "capability renderable meta title key"
	colCapRenderableMetaDesc   = "capability renderable meta description key"
	colFieldKey                = "field key"
	colFieldName               = "field name"
	colFieldType               = "field type"
	colFieldDescription        = "field description"
	colFieldRequired           = "field required"
	colValidationName          = "validation name"
	colValidationValue         = "validation value"
)

// Required columns; importing without them is an error.
var requiredColumns = []string{colType, colName, colFieldKey, colFieldType}

// definitionLevelColumns are set once per definition. Rows of the same type
// may repeat them, but a row that gives a different value than an earlier row
// is an error rather than a silent override.
var definitionLevelColumns = []string{
	colName, colDescription, colDisplayNameKey,
	colAccessAdmin, colAccessCustomerAccount, colAccessStorefront,
	colCapPublishable, colCapOnlineStore, colCapOnlineStoreURL, colCapOnlineStoreRedirects,
	colCapTranslatable, colCapRenderable, colCapRenderableMetaTitle, colCapRenderableMetaDesc,
}

// definitionInput is one metaobject definition to create, assembled from one
// or more spreadsheet rows (each row adds one field).
type definitionInput struct {
	Row            int // spreadsheet row the definition was first seen on
	Type           string
	Name           string
	Desc           string
	DisplayNameKey string
	Access         map[string]string // admin, customerAccount, storefront
	Capability     map[string]bool   // publishable, onlineStore, translatable, renderable

	// Capability subtype data: MetaobjectCapabilityDefinitionDataOnlineStore
	// and MetaobjectCapabilityDefinitionDataRenderable.
	OnlineStoreURL               string
	OnlineStoreCreateRedirects   bool
	RenderableMetaTitleKey       string
	RenderableMetaDescriptionKey string

	Fields []fieldInput
}

type fieldInput struct {
	Key         string
	Name        string
	Type        string
	Desc        string
	Required    bool
	Validations []validationInput
}

type validationInput struct {
	Name  string
	Value string
}

func importDefinitionAction(c *cli.Context) error {
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

	for _, d := range defs {
		id, err := gql.CreateMetaobjectDefinition(client, definitionInputJSON(d))
		if err != nil {
			failed++
			if jsonOutput {
				jsonResults = append(jsonResults, map[string]interface{}{
					"row":    d.Row,
					"type":   d.Type,
					"id":     "",
					"status": "error",
					"errors": []string{err.Error()},
				})
			} else {
				fmt.Fprintf(os.Stderr, "Row %d %s: Error: %s\n", d.Row, d.Type, err)
			}
			continue
		}
		created++
		id = strings.TrimPrefix(id, gql.DefinitionGIDPrefix)
		if jsonOutput {
			jsonResults = append(jsonResults, map[string]interface{}{
				"row":    d.Row,
				"type":   d.Type,
				"id":     id,
				"status": "ok",
			})
		} else {
			fmt.Printf("Row %3d %s: Created (%s)\n", d.Row, d.Type, id)
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

// rawDefinition accumulates the spreadsheet rows of one metaobject type before
// they're turned into a definitionInput.
type rawDefinition struct {
	row     int // spreadsheet row the type was first seen on
	cells   map[string]string
	fields  []fieldInput
	repeats map[string]int // field key -> spreadsheet row that defined it
}

// parseDefinitionCSV reads metaobject definitions from a CSV spreadsheet. Each
// row is one field of a definition; rows with the same type are merged into a
// single definition. Multiple validations per field come from repeated
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

	var order []string
	byType := map[string]*rawDefinition{}

	for n, row := range records[1:] {
		rowNum := n + 2
		cell := func(col string) string { return strings.TrimSpace(getCell(row, cols, col)) }
		cellAt := func(i int) string {
			if i >= len(row) {
				return ""
			}
			return strings.TrimSpace(row[i])
		}

		moType := cell(colType)
		if moType == "" {
			return nil, fmt.Errorf("Row %d: required column '%s' empty", rowNum, colType)
		}

		raw, ok := byType[moType]
		if !ok {
			raw = &rawDefinition{row: rowNum, cells: map[string]string{}, repeats: map[string]int{}}
			byType[moType] = raw
			order = append(order, moType)
		}

		// A later row may leave definition level columns empty, but must not
		// contradict a value given by an earlier row.
		for _, col := range definitionLevelColumns {
			v := cell(col)
			if v == "" {
				continue
			}
			if have, ok := raw.cells[col]; ok && have != v {
				return nil, fmt.Errorf("Row %d: %s '%s' conflicts with '%s' on row %d", rowNum, col, v, have, raw.row)
			}
			raw.cells[col] = v
		}

		f := fieldInput{
			Key:  cell(colFieldKey),
			Name: cell(colFieldName),
			Type: cell(colFieldType),
			Desc: cell(colFieldDescription),
		}

		for _, col := range requiredColumns {
			if cell(col) == "" {
				return nil, fmt.Errorf("Row %d: required column '%s' empty", rowNum, col)
			}
		}

		// Parse the validation column pairs for this row. The validation value
		// is passed through verbatim (it may itself contain "|", e.g. regex).
		for k := range valNameIdx {
			name, value := cellAt(valNameIdx[k]), cellAt(valValueIdx[k])
			if name != "" || value != "" {
				if name == "" || value == "" {
					return nil, fmt.Errorf("Row %d: %s/%s pair %d must have both name and value", rowNum, colValidationName, colValidationValue, k+1)
				}
				f.Validations = append(f.Validations, validationInput{Name: name, Value: value})
			}
		}

		required, err := parseBool(cell(colFieldRequired))
		if err != nil {
			return nil, fmt.Errorf("Row %d: %s: %s", rowNum, colFieldRequired, err)
		}
		f.Required = required

		if prev, dup := raw.repeats[f.Key]; dup {
			return nil, fmt.Errorf("Row %d: duplicate field key '%s' for type '%s' (first defined on row %d)", rowNum, f.Key, moType, prev)
		}
		raw.repeats[f.Key] = rowNum
		raw.fields = append(raw.fields, f)
	}

	defs := make([]*definitionInput, 0, len(order))
	for _, moType := range order {
		raw := byType[moType]

		d := &definitionInput{
			Type:           moType,
			Name:           raw.cells[colName],
			Desc:           raw.cells[colDescription],
			DisplayNameKey: raw.cells[colDisplayNameKey],
			Access:         map[string]string{},
			Capability:     map[string]bool{},
			Fields:         raw.fields,
			Row:            raw.row,
		}

		// The API's access input has admin, customerAccount and storefront.
		d.Access["admin"] = raw.cells[colAccessAdmin]
		d.Access["customerAccount"] = raw.cells[colAccessCustomerAccount]
		d.Access["storefront"] = raw.cells[colAccessStorefront]

		capabilityCols := []struct{ col, key string }{
			{colCapPublishable, "publishable"},
			{colCapOnlineStore, "onlineStore"},
			{colCapTranslatable, "translatable"},
			{colCapRenderable, "renderable"},
		}
		for _, cc := range capabilityCols {
			b, err := parseBool(raw.cells[cc.col])
			if err != nil {
				return nil, fmt.Errorf("Row %d: %s: %s", raw.row, cc.col, err)
			}
			d.Capability[cc.key] = b
		}

		// Capability subtype data is passed through as-is; the API validates it.
		d.OnlineStoreURL = raw.cells[colCapOnlineStoreURL]
		d.RenderableMetaTitleKey = raw.cells[colCapRenderableMetaTitle]
		d.RenderableMetaDescriptionKey = raw.cells[colCapRenderableMetaDesc]

		redirects, err := parseBool(raw.cells[colCapOnlineStoreRedirects])
		if err != nil {
			return nil, fmt.Errorf("Row %d: %s: %s", raw.row, colCapOnlineStoreRedirects, err)
		}
		d.OnlineStoreCreateRedirects = redirects

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

func getCell(row []string, cols map[string]int, name string) string {
	i, ok := cols[name]
	if !ok || i >= len(row) {
		return ""
	}
	return row[i]
}

func knownColumn(name string) bool {
	switch name {
	case colType, colName, colDescription, colDisplayNameKey,
		colAccessAdmin, colAccessCustomerAccount, colAccessStorefront,
		colCapPublishable, colCapOnlineStore, colCapOnlineStoreURL, colCapOnlineStoreRedirects,
		colCapTranslatable, colCapRenderable, colCapRenderableMetaTitle, colCapRenderableMetaDesc,
		colFieldKey, colFieldName, colFieldType, colFieldDescription, colFieldRequired:
		return true
	}
	return false
}

// definitionInputJSON builds the MetaobjectDefinitionCreateInput for a
// definition.
func definitionInputJSON(d *definitionInput) map[string]interface{} {
	definition := map[string]interface{}{
		"name": d.Name,
		"type": d.Type,
	}

	if d.Desc != "" {
		definition["description"] = d.Desc
	}
	if d.DisplayNameKey != "" {
		definition["displayNameKey"] = d.DisplayNameKey
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
		if !v {
			continue
		}

		capability := map[string]interface{}{"enabled": true}
		if data := d.capabilityData(k); len(data) > 0 {
			capability["data"] = data
		}
		capabilities[k] = capability
	}
	if len(capabilities) > 0 {
		definition["capabilities"] = capabilities
	}

	fields := make([]map[string]interface{}, 0, len(d.Fields))
	for _, f := range d.Fields {
		field := map[string]interface{}{
			"key":  f.Key,
			"type": f.Type,
		}

		if f.Name != "" {
			field["name"] = f.Name
		}
		if f.Desc != "" {
			field["description"] = f.Desc
		}
		if f.Required {
			field["required"] = true
		}

		if len(f.Validations) > 0 {
			validations := make([]map[string]interface{}, 0, len(f.Validations))
			for _, v := range f.Validations {
				validations = append(validations, map[string]interface{}{
					"name":  v.Name,
					"value": v.Value,
				})
			}
			field["validations"] = validations
		}

		fields = append(fields, field)
	}
	definition["fieldDefinitions"] = fields

	return definition
}

// capabilityData returns a capability's subtype data, left out when the
// spreadsheet doesn't set any of its fields so the API applies its defaults.
func (d *definitionInput) capabilityData(name string) map[string]interface{} {
	data := map[string]interface{}{}

	switch name {
	case "onlineStore":
		if d.OnlineStoreURL != "" {
			data["urlHandle"] = d.OnlineStoreURL
		}
		if d.OnlineStoreCreateRedirects {
			data["createRedirects"] = true
		}
	case "renderable":
		if d.RenderableMetaTitleKey != "" {
			data["metaTitleKey"] = d.RenderableMetaTitleKey
		}
		if d.RenderableMetaDescriptionKey != "" {
			data["metaDescriptionKey"] = d.RenderableMetaDescriptionKey
		}
	}

	return data
}

// deleteDefinitionAction deletes each of the given metaobject definitions,
// by numeric id or GID.
func deleteDefinitionAction(c *cli.Context) error {
	if c.NArg() == 0 {
		return errors.New("Metaobject definition id required")
	}

	client := cmd.NewGraphQLClient(c)

	var failures []string
	for _, arg := range c.Args().Slice() {
		id, err := gql.DeleteMetaobjectDefinition(client, arg)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: %s", arg, err))
			continue
		}

		fmt.Printf("Deleted %s\n", strings.TrimPrefix(id, gql.DefinitionGIDPrefix))
	}

	if len(failures) > 0 {
		return fmt.Errorf("Cannot delete metaobject definition(s): %s", strings.Join(failures, ", "))
	}

	return nil
}
