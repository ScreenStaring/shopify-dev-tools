package metafields

import (
	"reflect"
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

func TestParseDefinitionCSVMultipleValidationPairs(t *testing.T) {
	// Each validation name/validation value column pair is one validation.
	// Repeated header names (no number suffix) pair by position, and a row
	// may fill only some of the pairs.
	cols := append(append([]string{}, allColumns...), colValidationName, colValidationValue)
	cols = append(cols, colValidationName, colValidationValue)

	threePairs := make([]string, len(cols))
	copy(threePairs, []string{"PRODUCT", "Weight", "weight", "weight", "custom", "", "false"})
	threePairs[indexOfColumn(colValidationName)] = "min"
	threePairs[indexOfColumn(colValidationValue)] = "0.1"
	threePairs[len(cols)-4] = "regex"
	threePairs[len(cols)-3] = "^(foo|bar)$"
	threePairs[len(cols)-2] = "max"
	threePairs[len(cols)-1] = "25"

	onePair := make([]string, len(cols))
	copy(onePair, []string{"PRODUCT", "Dimensions", "dimensions", "number_decimal", "custom", "", "false"})
	onePair[indexOfColumn(colValidationName)] = "min"
	onePair[indexOfColumn(colValidationValue)] = "1.5"

	defs, err := parseDefinitionCSV(strings.NewReader(csvWith(cols, threePairs, onePair)))
	if err != nil {
		t.Fatalf("parse failed: %s", err)
	}
	if len(defs) != 2 {
		t.Fatalf("want 2 definitions, got %d", len(defs))
	}

	want := []validationInput{
		{Name: "min", Value: "0.1"},
		{Name: "regex", Value: "^(foo|bar)$"},
		{Name: "max", Value: "25"},
	}
	if got := defs[0].Validations; !reflect.DeepEqual(got, want) {
		t.Errorf("three-pair row validations = %+v, want %+v", got, want)
	}

	want = []validationInput{{Name: "min", Value: "1.5"}}
	if got := defs[1].Validations; !reflect.DeepEqual(got, want) {
		t.Errorf("one-pair row validations = %+v, want %+v", got, want)
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

func TestParseDefinitionCSVNormalizesCategoryConstraints(t *testing.T) {
	// Category constraint values may be given as bare taxonomy category IDs,
	// as the API returns them, or as GIDs; both are sent as GIDs.
	csv := csvWith(allColumns,
		[]string{"PRODUCT", "Mat", "m", "single_line_text_field", "custom", "", "false", "", "", "", "", "", "category", "fb-1-1-5|gid://shopify/TaxonomyCategory/aa-1"},
	)

	defs, err := parseDefinitionCSV(strings.NewReader(csv))
	if err != nil {
		t.Fatalf("parse failed: %s", err)
	}
	if len(defs) != 1 {
		t.Fatalf("want 1 definition, got %d", len(defs))
	}

	got := defs[0].ConstraintsValues
	want := []string{"gid://shopify/TaxonomyCategory/fb-1-1-5", "gid://shopify/TaxonomyCategory/aa-1"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("constraints values = %v, want %v", got, want)
	}
}

func TestParseDefinitionCSVLeavesOtherConstraintsAlone(t *testing.T) {
	csv := csvWith(allColumns,
		[]string{"PRODUCT", "Mat", "m", "single_line_text_field", "custom", "", "false", "", "", "", "", "", "other", "fb-1-1-5"},
	)

	defs, err := parseDefinitionCSV(strings.NewReader(csv))
	if err != nil {
		t.Fatalf("parse failed: %s", err)
	}
	if len(defs) != 1 {
		t.Fatalf("want 1 definition, got %d", len(defs))
	}
	if got := defs[0].ConstraintsValues; len(got) != 1 || got[0] != "fb-1-1-5" {
		t.Errorf("constraints values = %v, want unchanged", got)
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

func TestDefinitionTargets(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantTargets []definitionTarget
		wantSets    []definitionSet
		wantErr     bool
	}{
		{
			name: "id and gid",
			args: []string{"123", "gid://shopify/MetafieldDefinition/456"},
			wantTargets: []definitionTarget{
				{label: "123", id: "gid://shopify/MetafieldDefinition/123"},
				{label: "gid://shopify/MetafieldDefinition/456", id: "gid://shopify/MetafieldDefinition/456"},
			},
		},
		{
			name:     "owner and namespace",
			args:     []string{"PRODUCT@custom"},
			wantSets: []definitionSet{{ownerType: "PRODUCT", namespace: "custom"}},
		},
		{
			name:     "owner type is uppercased",
			args:     []string{"product@custom"},
			wantSets: []definitionSet{{ownerType: "PRODUCT", namespace: "custom"}},
		},
		{
			name:     "namespace case is preserved",
			args:     []string{"SHOP@Mixed Case"},
			wantSets: []definitionSet{{ownerType: "SHOP", namespace: "Mixed Case"}},
		},
		{
			name: "multiple sets",
			args: []string{"PRODUCT@custom", "COLLECTION@other"},
			wantSets: []definitionSet{
				{ownerType: "PRODUCT", namespace: "custom"},
				{ownerType: "COLLECTION", namespace: "other"},
			},
		},
		{
			name: "mixed ids and sets",
			args: []string{"123", "PRODUCT@custom", "456"},
			wantTargets: []definitionTarget{
				{label: "123", id: "gid://shopify/MetafieldDefinition/123"},
				{label: "456", id: "gid://shopify/MetafieldDefinition/456"},
			},
			wantSets: []definitionSet{{ownerType: "PRODUCT", namespace: "custom"}},
		},
		{
			name:    "owner type missing",
			args:    []string{"@custom"},
			wantErr: true,
		},
		{
			name:    "namespace missing",
			args:    []string{"PRODUCT@"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			targets, sets, err := definitionTargets(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("definitionTargets(%v) = %v, %v; want error", tt.args, targets, sets)
				}
				return
			}
			if err != nil {
				t.Fatalf("definitionTargets(%v) unexpected error: %s", tt.args, err)
			}
			if !reflect.DeepEqual(targets, tt.wantTargets) {
				t.Errorf("definitionTargets(%v) targets = %v, want %v", tt.args, targets, tt.wantTargets)
			}
			if !reflect.DeepEqual(sets, tt.wantSets) {
				t.Errorf("definitionTargets(%v) sets = %v, want %v", tt.args, sets, tt.wantSets)
			}
		})
	}
}
