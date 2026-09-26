package metaobjects

import (
	"strings"
	"testing"
)

var allColumns = []string{
	colType, colName, colDescription, colDisplayNameKey,
	colAccessAdmin, colAccessCustomerAccount, colAccessStorefront,
	colCapPublishable, colCapOnlineStore, colCapOnlineStoreURL, colCapOnlineStoreRedirects,
	colCapTranslatable, colCapRenderable, colCapRenderableMetaTitle, colCapRenderableMetaDesc,
	colFieldKey, colFieldName, colFieldType, colFieldDescription, colFieldRequired,
	colValidationName, colValidationValue,
}

// csvRaw builds a CSV string from positional rows; rows shorter than the
// header are padded with empty cells, longer rows keep extra cells (for
// unknown-column cases).
func csvRaw(cols []string, rows ...[]string) string {
	var b strings.Builder
	b.WriteString(strings.Join(cols, ","))
	b.WriteString("\n")
	for _, r := range rows {
		n := len(cols)
		if len(r) > n {
			n = len(r)
		}
		out := make([]string, n)
		copy(out, r)
		b.WriteString(strings.Join(out, ","))
		b.WriteString("\n")
	}
	return b.String()
}

func rowFromMap(m map[string]string) []string {
	out := make([]string, len(allColumns))
	for i, c := range allColumns {
		out[i] = m[c]
	}
	return out
}

// csvWith builds a CSV string with one row per map, keyed by column name.
func csvWith(rows ...map[string]string) string {
	out := make([][]string, len(rows))
	for i, r := range rows {
		out[i] = rowFromMap(r)
	}
	return csvRaw(allColumns, out...)
}

func without(cols []string, drop string) []string {
	out := make([]string, 0, len(cols))
	for _, c := range cols {
		if c != drop {
			out = append(out, c)
		}
	}
	return out
}

func TestParseDefinitionCSVHappyPath(t *testing.T) {
	csv := csvWith(
		map[string]string{
			colType: "author", colName: "Author", colDescription: "An author", colDisplayNameKey: "name",
			colAccessAdmin: "MERCHANT_READ_WRITE", colAccessCustomerAccount: "READ", colAccessStorefront: "PUBLIC_READ",
			colCapPublishable: "true", colCapTranslatable: "false", colCapRenderable: "true",
			colFieldKey: "name", colFieldName: "Name", colFieldType: "single_line_text_field",
			colFieldDescription: "Full name", colFieldRequired: "true",
			colValidationName: "min", colValidationValue: "1",
		},
		map[string]string{
			colType: "author", colName: "Author",
			colFieldKey: "bio", colFieldName: "Bio", colFieldType: "multi_line_text_field",
			colValidationName: "max_length", colValidationValue: "500",
		},
		map[string]string{
			colType: "color", colName: "Color", colAccessAdmin: "MERCHANT_READ",
			colCapOnlineStore: "true",
			colFieldKey:       "hex", colFieldName: "Hex", colFieldType: "single_line_text_field",
		},
	)

	defs, err := parseDefinitionCSV(strings.NewReader(csv))
	if err != nil {
		t.Fatalf("parse failed: %s", err)
	}

	if len(defs) != 2 {
		t.Fatalf("want 2 definitions, got %d", len(defs))
	}

	author := defs[0]
	if author.Type != "author" || author.Name != "Author" || author.Row != 2 {
		t.Errorf("author def wrong: %+v", author)
	}
	if author.Desc != "An author" || author.DisplayNameKey != "name" {
		t.Errorf("author description/display name key wrong: %+v", author)
	}
	if !author.Capability["publishable"] || author.Capability["translatable"] || !author.Capability["renderable"] {
		t.Errorf("author capabilities wrong: %v", author.Capability)
	}
	if author.Access["admin"] != "MERCHANT_READ_WRITE" || author.Access["customerAccount"] != "READ" || author.Access["storefront"] != "PUBLIC_READ" {
		t.Errorf("author access wrong: %v", author.Access)
	}
	if len(author.Fields) != 2 {
		t.Fatalf("want 2 author fields, got %d", len(author.Fields))
	}

	name := author.Fields[0]
	if name.Key != "name" || name.Name != "Name" || name.Type != "single_line_text_field" || !name.Required {
		t.Errorf("name field wrong: %+v", name)
	}
	if len(name.Validations) != 1 || name.Validations[0].Name != "min" || name.Validations[0].Value != "1" {
		t.Errorf("name field validations wrong: %+v", name.Validations)
	}

	bio := author.Fields[1]
	if bio.Key != "bio" || bio.Required || len(bio.Validations) != 1 || bio.Validations[0].Value != "500" {
		t.Errorf("bio field wrong: %+v", bio)
	}

	color := defs[1]
	if color.Type != "color" || color.Row != 4 || len(color.Fields) != 1 {
		t.Errorf("color def wrong: %+v", color)
	}
	if !color.Capability["onlineStore"] || color.Capability["publishable"] {
		t.Errorf("color online store capability wrong: %v", color.Capability)
	}
	if color.Access["admin"] != "MERCHANT_READ" || color.Access["storefront"] != "" {
		t.Errorf("color access wrong: %v", color.Access)
	}
}

func TestParseDefinitionCSVRepeatedPairColumns(t *testing.T) {
	// validation name/validation value repeated in the header (no number
	// suffix) is treated as pair 2 by position.
	cols := append(append([]string{}, allColumns...), colValidationName, colValidationValue)
	row := rowFromMap(map[string]string{
		colType: "author", colName: "Author",
		colFieldKey: "bio", colFieldName: "Bio", colFieldType: "multi_line_text_field",
		colValidationName: "min", colValidationValue: "1",
	})
	row = append(row, "regex", "^(foo|bar)$")

	defs, err := parseDefinitionCSV(strings.NewReader(csvRaw(cols, row)))
	if err != nil {
		t.Fatalf("parse failed: %s", err)
	}
	if len(defs) != 1 {
		t.Fatalf("want 1 definition, got %d", len(defs))
	}

	got := defs[0].Fields[0].Validations
	if len(got) != 2 {
		t.Fatalf("want 2 validations, got %d", len(got))
	}
	if got[1].Name != "regex" || got[1].Value != "^(foo|bar)$" {
		t.Errorf("validations wrong: %+v", got)
	}
}

func TestParseDefinitionCSVCaseInsensitiveHeaders(t *testing.T) {
	cols := make([]string, len(allColumns))
	for i, c := range allColumns {
		cols[i] = strings.ToUpper(c)
	}

	defs, err := parseDefinitionCSV(strings.NewReader(csvRaw(cols, rowFromMap(map[string]string{
		colType: "AUTHOR", colName: "Author", colDisplayNameKey: "name", colCapPublishable: "true",
		colFieldKey: "name", colFieldName: "Name", colFieldType: "single_line_text_field", colFieldRequired: "true",
	}))))
	if err != nil {
		t.Fatalf("parse failed: %s", err)
	}
	if len(defs) != 1 {
		t.Fatalf("want 1 definition, got %d", len(defs))
	}

	got := defs[0]
	if got.Type != "AUTHOR" || !got.Capability["publishable"] || !got.Fields[0].Required {
		t.Errorf("def wrong: %+v", got)
	}
}

func TestParseDefinitionCSVFieldOrderFollowsRows(t *testing.T) {
	csv := csvWith(
		map[string]string{
			colType: "author", colName: "Author",
			colFieldKey: "second", colFieldName: "Second", colFieldType: "single_line_text_field",
		},
		map[string]string{
			colType: "author", colName: "Author",
			colFieldKey: "first", colFieldName: "First", colFieldType: "single_line_text_field",
		},
	)

	defs, err := parseDefinitionCSV(strings.NewReader(csv))
	if err != nil {
		t.Fatalf("parse failed: %s", err)
	}
	if len(defs) != 1 {
		t.Fatalf("want 1 definition, got %d", len(defs))
	}
	if len(defs[0].Fields) != 2 || defs[0].Fields[0].Key != "second" || defs[0].Fields[1].Key != "first" {
		t.Errorf("field order wrong: %+v", defs[0].Fields)
	}
}

func TestParseDefinitionCSVErrors(t *testing.T) {
	field := map[string]string{
		colType: "author", colName: "Author",
		colFieldKey: "name", colFieldName: "Name", colFieldType: "single_line_text_field",
	}
	withField := func(overrides map[string]string) map[string]string {
		row := map[string]string{}
		for k, v := range field {
			row[k] = v
		}
		for k, v := range overrides {
			row[k] = v
		}
		return row
	}

	tests := []struct {
		name string
		csv  string
		want string
	}{
		{
			"missing required column",
			csvRaw(without(allColumns, colFieldKey), rowFromMap(field)),
			"Missing required column 'field key'",
		},
		{
			"unknown column",
			csvRaw(append(append([]string{}, allColumns...), "typo"), rowFromMap(field)),
			"Unknown column 'typo'",
		},
		{
			"invalid field required",
			csvWith(withField(map[string]string{colFieldRequired: "maybe"})),
			"'maybe' is not a valid boolean",
		},
		{
			"invalid capability",
			csvWith(withField(map[string]string{colCapPublishable: "yes"})),
			"capability publishable",
		},
		{
			"empty required value",
			csvWith(withField(map[string]string{colFieldKey: ""})),
			"required column 'field key' empty",
		},
		{
			"empty type",
			csvWith(withField(map[string]string{colType: ""})),
			"required column 'type' empty",
		},
		{
			"conflicting definition level value",
			csvWith(
				withField(nil),
				withField(map[string]string{colName: "Writer", colFieldKey: "bio", colFieldName: "Bio"}),
			),
			"name 'Writer' conflicts with 'Author' on row 2",
		},
		{
			"duplicate field key",
			csvWith(
				withField(nil),
				withField(map[string]string{colFieldName: "Other"}),
			),
			"duplicate field key 'name' for type 'author' (first defined on row 2)",
		},
		{
			"validation name without value",
			csvWith(withField(map[string]string{colValidationName: "min"})),
			"pair 1 must have both name and value",
		},
		{
			"mismatched validation pair columns",
			csvRaw(append(append([]string{}, allColumns...), colValidationName), rowFromMap(field)),
			"must match",
		},
		{
			"duplicate non-validation column",
			csvRaw(append(append([]string{}, allColumns...), colName), rowFromMap(field)),
			"Duplicate column 'name' in header",
		},
		{
			"header only",
			csvRaw(allColumns),
			"at least one data row",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseDefinitionCSV(strings.NewReader(tt.csv))
			if err == nil {
				t.Fatal("want error, got nil")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error %q does not contain %q", err, tt.want)
			}
		})
	}
}

func TestParseDefinitionCSVReadmeExample(t *testing.T) {
	csv := "Type,Name,Display Name Key,Access Storefront,Capability Publishable,Field Key,Field Name,Field Type,Field Required,Validation Name,Validation Value\n" +
		"author,Author,name,PUBLIC_READ,true,name,Name,single_line_text_field,true,max_length,255\n" +
		"author,Author,,,,bio,Biography,multi_line_text_field,false,,\n"

	defs, err := parseDefinitionCSV(strings.NewReader(csv))
	if err != nil {
		t.Fatalf("parse failed: %s", err)
	}
	if len(defs) != 1 {
		t.Fatalf("want 1 definition, got %d", len(defs))
	}

	got := defs[0]
	if got.Type != "author" || got.DisplayNameKey != "name" || got.Access["storefront"] != "PUBLIC_READ" || !got.Capability["publishable"] {
		t.Errorf("def wrong: %+v", got)
	}
	if len(got.Fields) != 2 {
		t.Fatalf("want 2 fields, got %d", len(got.Fields))
	}
	if got.Fields[0].Key != "name" || !got.Fields[0].Required || got.Fields[0].Validations[0].Name != "max_length" {
		t.Errorf("name field wrong: %+v", got.Fields[0])
	}
	if got.Fields[1].Key != "bio" || got.Fields[1].Required || len(got.Fields[1].Validations) != 0 {
		t.Errorf("bio field wrong: %+v", got.Fields[1])
	}
}

func TestDefinitionInputJSONCapabilities(t *testing.T) {
	d := &definitionInput{
		Type:           "size_chart",
		Name:           "Size Chart",
		Desc:           "Apparel measurements",
		DisplayNameKey: "title",
		Access:         map[string]string{"admin": "", "customerAccount": "", "storefront": "PUBLIC_READ"},
		Capability:     map[string]bool{"publishable": true, "onlineStore": true, "translatable": false, "renderable": true},

		OnlineStoreURL:               "size-charts",
		OnlineStoreCreateRedirects:   true,
		RenderableMetaTitleKey:       "title",
		RenderableMetaDescriptionKey: "blurb",

		Fields: []fieldInput{
			{Key: "title", Name: "Title", Type: "single_line_text_field", Required: true},
		},
	}

	got := definitionInputJSON(d)

	if got["name"] != "Size Chart" || got["type"] != "size_chart" || got["displayNameKey"] != "title" {
		t.Errorf("definition wrong: %v", got)
	}

	capabilities := got["capabilities"].(map[string]interface{})
	if len(capabilities) != 3 {
		t.Errorf("capabilities: want publishable, onlineStore and renderable, got %v", capabilities)
	}

	publishable := capabilities["publishable"].(map[string]interface{})
	if publishable["enabled"] != true || publishable["data"] != nil {
		t.Errorf("publishable wrong: %v", publishable)
	}

	onlineStore := capabilities["onlineStore"].(map[string]interface{})
	if onlineStore["enabled"] != true {
		t.Errorf("onlineStore enabled: want true, got %v", onlineStore["enabled"])
	}
	onlineStoreData := onlineStore["data"].(map[string]interface{})
	if onlineStoreData["urlHandle"] != "size-charts" || onlineStoreData["createRedirects"] != true {
		t.Errorf("onlineStore data wrong: %v", onlineStoreData)
	}

	renderable := capabilities["renderable"].(map[string]interface{})
	renderableData := renderable["data"].(map[string]interface{})
	if renderableData["metaTitleKey"] != "title" || renderableData["metaDescriptionKey"] != "blurb" {
		t.Errorf("renderable data wrong: %v", renderableData)
	}

	access := got["access"].(map[string]interface{})
	if len(access) != 1 || access["storefront"] != "PUBLIC_READ" {
		t.Errorf("access wrong: %v", access)
	}

	fields := got["fieldDefinitions"].([]map[string]interface{})
	if len(fields) != 1 || fields[0]["key"] != "title" || fields[0]["required"] != true {
		t.Errorf("fields wrong: %v", fields)
	}
}

func TestDefinitionInputJSONOmitsCapabilityData(t *testing.T) {
	d := &definitionInput{
		Type:       "color",
		Name:       "Color",
		Access:     map[string]string{"admin": "", "customerAccount": "", "storefront": ""},
		Capability: map[string]bool{"publishable": false, "onlineStore": true, "translatable": false, "renderable": false},
		Fields:     []fieldInput{{Key: "hex", Type: "single_line_text_field"}},
	}

	got := definitionInputJSON(d)

	onlineStore := got["capabilities"].(map[string]interface{})["onlineStore"].(map[string]interface{})
	if onlineStore["enabled"] != true {
		t.Errorf("onlineStore enabled: want true, got %v", onlineStore["enabled"])
	}
	if _, ok := onlineStore["data"]; ok {
		t.Errorf("onlineStore data should be omitted when the spreadsheet sets none: %v", onlineStore)
	}
}

func TestDefinitionInputJSONOmitsEmpty(t *testing.T) {
	d := &definitionInput{
		Type:       "faq",
		Name:       "FAQ",
		Access:     map[string]string{"admin": "", "customerAccount": "", "storefront": ""},
		Capability: map[string]bool{"publishable": false, "onlineStore": false, "translatable": false, "renderable": false},
		Fields:     []fieldInput{{Key: "question", Type: "single_line_text_field"}},
	}

	got := definitionInputJSON(d)

	if len(got) != 3 {
		t.Errorf("want name, type and fieldDefinitions only, got %v", got)
	}
	if _, ok := got["capabilities"]; ok {
		t.Errorf("capabilities should be omitted: %v", got)
	}
}

func TestParseBool(t *testing.T) {
	for _, s := range []string{"", "false", "FALSE", "False"} {
		if b, err := parseBool(s); err != nil || b {
			t.Errorf("parseBool(%q) = %v, %v; want false, nil", s, b, err)
		}
	}
	for _, s := range []string{"true", "TRUE", "True"} {
		if b, err := parseBool(s); err != nil || !b {
			t.Errorf("parseBool(%q) = %v, %v; want true, nil", s, b, err)
		}
	}
	if _, err := parseBool("yes"); err == nil {
		t.Error("parseBool(yes) want error")
	}
}
