package metafields

import (
	"strings"
	"testing"
)

var allColumns = []string{
	colOwnerType, colName, colKey, colType, colNamespace, colDescription,
	colPin, colAccessAdmin, colAccessCustomerAccount,
	colCapAdminFilterable, colCapSmartCollection, colCapUniqueValues,
	colConstraintsKey, colConstraintsValues, colValidationName, colValidationValue,
}

func indexOfColumn(col string) int {
	for i, c := range allColumns {
		if c == col {
			return i
		}
	}
	return -1
}

// csvWith builds a CSV string; rows shorter than the header are padded with
// empty cells, longer rows keep extra cells (for unknown-column cases).
func csvWith(cols []string, rows ...[]string) string {
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

func TestParseDefinitionCSVHappyPath(t *testing.T) {
	cols := append(append([]string{}, allColumns...), "validation name", "validation value")
	csv := csvWith(cols,
		[]string{"PRODUCT", "Material", "material", "single_line_text_field", "custom", "Fabric", "false", "MERCHANT_READ_WRITE", "NONE", "true", "false", "false"},
		[]string{"PRODUCT", "Weight", "weight", "weight", "custom", "", "false", "MERCHANT_READ", "NONE", "false", "false", "false", "", "", "min", "0.1", "max", "25"},
	)

	defs, err := parseDefinitionCSV(strings.NewReader(csv))
	if err != nil {
		t.Fatalf("parse failed: %s", err)
	}

	if len(defs) != 2 {
		t.Fatalf("want 2 definitions, got %d", len(defs))
	}

	material := defs[0]
	if material.Key != "material" || !material.Capability["adminFilterable"] || material.Capability["uniqueValues"] {
		t.Errorf("material def wrong: %+v", material)
	}
	if material.Access["admin"] != "MERCHANT_READ_WRITE" || material.Access["customerAccount"] != "NONE" {
		t.Errorf("material access wrong: %v", material.Access)
	}
	if len(material.Validations) != 0 {
		t.Errorf("material should have no validations, got %v", material.Validations)
	}

	weight := defs[1]
	if weight.Key != "weight" || weight.Pin {
		t.Errorf("weight def wrong: %+v", weight)
	}
	if len(weight.Validations) != 2 {
		t.Fatalf("want 2 validations, got %d", len(weight.Validations))
	}
	if weight.Validations[0].Name != "min" || weight.Validations[0].Value != "0.1" {
		t.Errorf("min validation wrong: %+v", weight.Validations[0])
	}
	if weight.Validations[1].Name != "max" || weight.Validations[1].Value != "25" {
		t.Errorf("max validation wrong: %+v", weight.Validations[1])
	}
}

func TestParseDefinitionCSVRepeatedPairColumns(t *testing.T) {
	// validation name/validation value repeated in the header (no number
	// suffix) is treated as pair 2 by position.
	cols := append(append([]string{}, allColumns...), "validation name", "validation value")
	row := make([]string, len(cols))
	copy(row, []string{"PRODUCT", "Weight", "weight", "weight", "custom", "", "false"})
	row[indexOfColumn(colValidationName)] = "min"
	row[indexOfColumn(colValidationValue)] = "0.1"
	row[len(cols)-2] = "regex"
	row[len(cols)-1] = "^(foo|bar)$"

	defs, err := parseDefinitionCSV(strings.NewReader(csvWith(cols, row)))
	if err != nil {
		t.Fatalf("parse failed: %s", err)
	}
	if len(defs) != 1 {
		t.Fatalf("want 1 definition, got %d", len(defs))
	}
	got := defs[0].Validations
	if len(got) != 2 || got[1].Name != "regex" || got[1].Value != "^(foo|bar)$" {
		t.Errorf("validations wrong: %+v", got)
	}
}

func TestParseDefinitionCSVOwnerTypeCaseInsensitive(t *testing.T) {
	csv := csvWith(allColumns,
		[]string{"product", "Material", "material", "single_line_text_field", "custom", "Fabric", "false", "", "", "", "true"},
	)

	defs, err := parseDefinitionCSV(strings.NewReader(csv))
	if err != nil {
		t.Fatalf("parse failed: %s", err)
	}
	if len(defs) != 1 || defs[0].OwnerType != "PRODUCT" {
		t.Errorf("owner type not uppercased: %+v", defs)
	}
}

func TestParseDefinitionCSVConstraints(t *testing.T) {
	csv := csvWith(allColumns,
		[]string{"PRODUCT", "Mat", "m", "single_line_text_field", "custom", "", "false", "", "", "", "", "", "category", "gid://shopify/TaxonomyCategory/aa-8|gid://shopify/TaxonomyCategory/aa-8-1"},
	)

	defs, err := parseDefinitionCSV(strings.NewReader(csv))
	if err != nil {
		t.Fatalf("parse failed: %s", err)
	}
	if len(defs) != 1 {
		t.Fatalf("want 1 definition, got %d", len(defs))
	}
	got := defs[0]
	if got.ConstraintsKey != "category" {
		t.Errorf("constraints key: want category, got %s", got.ConstraintsKey)
	}
	if len(got.ConstraintsValues) != 2 || got.ConstraintsValues[1] != "gid://shopify/TaxonomyCategory/aa-8-1" {
		t.Errorf("constraints values wrong: %v", got.ConstraintsValues)
	}
}

func TestParseDefinitionCSVCaseInsensitiveHeaders(t *testing.T) {
	cols := make([]string, len(allColumns))
	for i, c := range allColumns {
		cols[i] = strings.ToUpper(c)
	}

	defs, err := parseDefinitionCSV(strings.NewReader(csvWith(cols,
		[]string{"PRODUCT", "Material", "material", "single_line_text_field", "custom", "", "true", "MERCHANT_READ_WRITE", "", "", "", "", "", "", "min", "0.1"})))
	if err != nil {
		t.Fatalf("parse failed: %s", err)
	}
	if len(defs) != 1 {
		t.Fatalf("want 1 definition, got %d", len(defs))
	}
	got := defs[0]
	if got.Key != "material" || !got.Pin || got.Access["admin"] != "MERCHANT_READ_WRITE" {
		t.Errorf("def wrong: %+v", got)
	}
	if len(got.Validations) != 1 || got.Validations[0].Name != "min" || got.Validations[0].Value != "0.1" {
		t.Errorf("validations wrong: %+v", got.Validations)
	}
}

func TestParseDefinitionCSVErrors(t *testing.T) {
	without := func(cols []string, drop string) []string {
		out := make([]string, 0, len(cols))
		for _, c := range cols {
			if c != drop {
				out = append(out, c)
			}
		}
		return out
	}

	tests := []struct {
		name string
		csv  string
		want string
	}{
		{
			"missing required column",
			csvWith(without(allColumns, colKey),
				[]string{"PRODUCT", "Mat", "single_line_text_field"}),
			"Missing required column 'key'",
		},
		{
			"unknown column",
			csvWith(append(append([]string{}, allColumns...), "typo"),
				[]string{"PRODUCT", "Mat", "m", "single_line_text_field"}),
			"Unknown column",
		},
		{
			"invalid pin",
			csvWith(allColumns,
				[]string{"PRODUCT", "Mat", "m", "single_line_text_field", "custom", "", "maybe"}),
			"pin",
		},
		{
			"constraints values without key",
			csvWith(allColumns,
				[]string{"PRODUCT", "Mat", "m", "single_line_text_field", "custom", "", "false", "", "", "", "", "", "", "gid://shopify/X"}),
			"constraint value",
		},
		{
			"constraints key without values",
			csvWith(allColumns,
				[]string{"PRODUCT", "Mat", "m", "single_line_text_field", "custom", "", "false", "", "", "", "", "", "category"}),
			"constraint key given without constraint value",
		},
		{
			"validation name without value",
			csvWith(allColumns,
				[]string{"PRODUCT", "Mat", "m", "single_line_text_field", "custom", "", "false", "", "", "", "", "", "", "", "min"}),
			"pair 1 must have both name and value",
		},
		{
			"empty required value",
			csvWith(allColumns,
				[]string{"PRODUCT", "Mat", "", "single_line_text_field"}),
			"required column 'key' empty",
		},
		{
			"numbered validation column rejected",
			csvWith(append(append([]string{}, allColumns...), "validation name2", "validation value2"),
				[]string{"PRODUCT", "Mat", "m", "single_line_text_field"}),
			"Unknown column",
		},
		{
			"mismatched validation pair columns",
			csvWith(append(append([]string{}, allColumns...), "validation name", "validation value", "validation name"),
				[]string{"PRODUCT", "Mat", "m", "single_line_text_field"}),
			"must match",
		},
		{
			"second pair without value",
			csvWith(append(append([]string{}, allColumns...), "validation name", "validation value"),
				[]string{"PRODUCT", "Mat", "m", "single_line_text_field", "custom", "", "false", "", "", "", "", "", "", "", "min", "1", "max"}),
			"pair 2 must have both name and value",
		},
		{
			"duplicate non-validation column",
			csvWith(append(append([]string{}, allColumns[:3]...), "name"),
				[]string{"PRODUCT", "Mat", "m", "single_line_text_field"}),
			"Duplicate column",
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
