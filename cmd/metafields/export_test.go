package metafields

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestDefinitionExportRoundTrip(t *testing.T) {
	defs := []MetafieldDefinition{
		{
			OwnerType:   "PRODUCT",
			Name:        "Material",
			Key:         "material",
			Type:        "single_line_text_field",
			Namespace:   "custom",
			Description: "Fabric",
			Pin:         true,
			Access: MetafieldDefinitionAccess{
				Admin:           "MERCHANT_READ_WRITE",
				CustomerAccount: "NONE",
				Storefront:      "PUBLIC_READ",
			},
			Capabilities: MetafieldDefinitionCapabilities{AdminFilterable: true},
		},
		{
			OwnerType:    "PRODUCT",
			Name:         "Weight",
			Key:          "weight",
			Type:         "weight",
			Namespace:    "custom",
			Capabilities: MetafieldDefinitionCapabilities{UniqueValues: true},
			Constraints: &MetafieldDefinitionConstraints{
				Key: "category",
				// The API returns category constraints as bare category IDs;
				// the export writes them as GIDs (already-GID values are kept).
				Values: []string{"aa-8", "gid://shopify/TaxonomyCategory/aa-8-1"},
			},
			Validations: []MetafieldDefinitionValidation{
				{Name: "min", Value: "1"},
				{Name: "max", Value: "25"},
			},
		},
	}

	pairs := maxValidationPairs(defs)
	if pairs != 2 {
		t.Fatalf("want 2 validation pairs, got %d", pairs)
	}

	var b strings.Builder
	w := csv.NewWriter(&b)
	if err := w.Write(definitionExportHeader(pairs)); err != nil {
		t.Fatal(err)
	}
	for _, d := range defs {
		if err := w.Write(definitionExportRow(d, pairs)); err != nil {
			t.Fatal(err)
		}
	}
	w.Flush()

	parsed, err := parseDefinitionCSV(strings.NewReader(b.String()))
	if err != nil {
		t.Fatalf("export cannot be imported: %s", err)
	}
	if len(parsed) != 2 {
		t.Fatalf("want 2 definitions, got %d", len(parsed))
	}

	material := parsed[0]
	if material.OwnerType != "PRODUCT" || material.Name != "Material" || material.Key != "material" ||
		material.Type != "single_line_text_field" || material.Namespace != "custom" || material.Desc != "Fabric" {
		t.Errorf("material definition wrong: %+v", material)
	}
	if !material.Pin {
		t.Error("material pin not exported")
	}
	if material.Access["admin"] != "MERCHANT_READ_WRITE" || material.Access["customerAccount"] != "NONE" || material.Access["storefront"] != "PUBLIC_READ" {
		t.Errorf("material access wrong: %v", material.Access)
	}
	if !material.Capability["adminFilterable"] || material.Capability["smartCollectionCondition"] || material.Capability["uniqueValues"] {
		t.Errorf("material capabilities wrong: %v", material.Capability)
	}
	if len(material.Validations) != 0 {
		t.Errorf("material should have no validations, got %v", material.Validations)
	}

	weight := parsed[1]
	if weight.Pin {
		t.Error("weight pin should be false")
	}
	if !weight.Capability["uniqueValues"] || weight.Capability["adminFilterable"] {
		t.Errorf("weight capabilities wrong: %v", weight.Capability)
	}
	if weight.ConstraintsKey != "category" {
		t.Errorf("constraints key: want category, got %q", weight.ConstraintsKey)
	}
	if len(weight.ConstraintsValues) != 2 ||
		weight.ConstraintsValues[0] != "gid://shopify/TaxonomyCategory/aa-8" ||
		weight.ConstraintsValues[1] != "gid://shopify/TaxonomyCategory/aa-8-1" {
		t.Errorf("constraints values wrong: %v", weight.ConstraintsValues)
	}
	if len(weight.Validations) != 2 {
		t.Fatalf("want 2 validations, got %d", len(weight.Validations))
	}
	if weight.Validations[1].Name != "max" || weight.Validations[1].Value != "25" {
		t.Errorf("max validation wrong: %+v", weight.Validations[1])
	}
}

func TestDefinitionExportMultipleValidationPairs(t *testing.T) {
	defs := []MetafieldDefinition{
		{
			OwnerType: "PRODUCT", Name: "Weight", Key: "weight", Type: "weight", Namespace: "custom",
			Validations: []MetafieldDefinitionValidation{
				{Name: "min", Value: "1"},
				{Name: "max", Value: "25"},
				{Name: "regex", Value: "^(foo|bar)$"},
			},
		},
		{
			OwnerType: "PRODUCT", Name: "Length", Key: "length", Type: "number_decimal", Namespace: "custom",
			Validations: []MetafieldDefinitionValidation{{Name: "min", Value: "0.5"}},
		},
	}

	pairs := maxValidationPairs(defs)
	if pairs != 3 {
		t.Fatalf("want 3 validation pairs, got %d", pairs)
	}

	header := definitionExportHeader(pairs)
	nameColumns, valueColumns := 0, 0
	for _, col := range header {
		switch strings.ToLower(col) {
		case colValidationName:
			nameColumns++
		case colValidationValue:
			valueColumns++
		}
	}
	if nameColumns != 3 || valueColumns != 3 {
		t.Fatalf("header has %d validation name and %d validation value columns, want 3 each", nameColumns, valueColumns)
	}

	var b strings.Builder
	w := csv.NewWriter(&b)
	if err := w.Write(header); err != nil {
		t.Fatal(err)
	}
	for _, d := range defs {
		if err := w.Write(definitionExportRow(d, pairs)); err != nil {
			t.Fatal(err)
		}
	}
	w.Flush()

	records, err := csv.NewReader(strings.NewReader(b.String())).ReadAll()
	if err != nil {
		t.Fatalf("cannot read exported CSV: %s", err)
	}
	if len(records) != 3 {
		t.Fatalf("want header and 2 rows, got %d records", len(records))
	}
	for i, record := range records {
		if len(record) != len(header) {
			t.Fatalf("record %d has %d cells, header has %d", i, len(record), len(header))
		}
	}

	// Validation cells start after the fixed columns, two per pair.
	base := len(definitionExportColumns)
	want := [][2]string{{"min", "1"}, {"max", "25"}, {"regex", "^(foo|bar)$"}}
	for i, pair := range want {
		name, value := records[1][base+2*i], records[1][base+2*i+1]
		if name != pair[0] || value != pair[1] {
			t.Errorf("pair %d = %q/%q, want %q/%q", i+1, name, value, pair[0], pair[1])
		}
	}

	// A definition with fewer validations leaves the extra pairs empty.
	if records[2][base] != "min" || records[2][base+1] != "0.5" {
		t.Errorf("first pair of padded row = %q/%q, want min/0.5", records[2][base], records[2][base+1])
	}
	for i := 1; i < pairs; i++ {
		if records[2][base+2*i] != "" || records[2][base+2*i+1] != "" {
			t.Errorf("pair %d of padded row not empty: %q/%q", i+1, records[2][base+2*i], records[2][base+2*i+1])
		}
	}

	// All three pairs survive the import.
	parsed, err := parseDefinitionCSV(strings.NewReader(b.String()))
	if err != nil {
		t.Fatalf("export cannot be imported: %s", err)
	}
	if len(parsed) != 2 {
		t.Fatalf("want 2 parsed definitions, got %d", len(parsed))
	}
	wantValidations := []validationInput{
		{Name: "min", Value: "1"},
		{Name: "max", Value: "25"},
		{Name: "regex", Value: "^(foo|bar)$"},
	}
	if !reflect.DeepEqual(parsed[0].Validations, wantValidations) {
		t.Errorf("imported validations = %+v, want %+v", parsed[0].Validations, wantValidations)
	}
	if !reflect.DeepEqual(parsed[1].Validations, []validationInput{{Name: "min", Value: "0.5"}}) {
		t.Errorf("imported padded row validations = %+v", parsed[1].Validations)
	}
}

func TestDefinitionExportNormalizesCategoryConstraints(t *testing.T) {
	index := func(name string) int {
		for i, col := range definitionExportColumns {
			if col == name {
				return i
			}
		}
		t.Fatalf("column %q not found", name)
		return -1
	}

	category := definitionExportRow(MetafieldDefinition{
		Constraints: &MetafieldDefinitionConstraints{Key: "category", Values: []string{"fb-1-1-5"}},
	}, 0)
	if got := category[index("Constraint Value")]; got != "gid://shopify/TaxonomyCategory/fb-1-1-5" {
		t.Errorf("category constraint value = %q, want GID", got)
	}

	// Values of other constraint subtypes are passed through as returned.
	other := definitionExportRow(MetafieldDefinition{
		Constraints: &MetafieldDefinitionConstraints{Key: "other", Values: []string{"fb-1-1-5"}},
	}, 0)
	if got := other[index("Constraint Value")]; got != "fb-1-1-5" {
		t.Errorf("other constraint value = %q, want unchanged", got)
	}
}

func TestDefinitionExportHeaderUsesImportColumns(t *testing.T) {
	header := definitionExportHeader(2)

	seen := make(map[string]bool, len(header))
	for _, col := range header {
		name := strings.ToLower(col)
		seen[name] = true
		if name == colValidationName || name == colValidationValue {
			continue
		}
		if !knownColumn(name) {
			t.Errorf("export column %q is not an import column", col)
		}
	}

	for _, col := range requiredColumns {
		if !seen[col] {
			t.Errorf("export missing required import column %q", col)
		}
	}

	validationPairs := 0
	for _, col := range header {
		if strings.ToLower(col) == colValidationName {
			validationPairs++
		}
	}
	if validationPairs != 2 {
		t.Errorf("want 2 validation name columns, got %d", validationPairs)
	}
}

func TestDefinitionExportRowPadsValidations(t *testing.T) {
	d := MetafieldDefinition{
		OwnerType:   "PRODUCT",
		Name:        "Weight",
		Key:         "weight",
		Type:        "weight",
		Validations: []MetafieldDefinitionValidation{{Name: "min", Value: "1"}},
	}

	row := definitionExportRow(d, 2)
	header := definitionExportHeader(2)

	if len(row) != len(header) {
		t.Fatalf("row has %d cells, header has %d", len(row), len(header))
	}
	if row[len(row)-2] != "" || row[len(row)-1] != "" {
		t.Errorf("row not padded: %v", row[len(row)-2:])
	}
}

func TestMaxValidationPairs(t *testing.T) {
	if got := maxValidationPairs(nil); got != 1 {
		t.Errorf("want 1 pair for no definitions, got %d", got)
	}

	defs := []MetafieldDefinition{
		{},
		{Validations: []MetafieldDefinitionValidation{{Name: "min", Value: "1"}}},
	}
	if got := maxValidationPairs(defs); got != 1 {
		t.Errorf("want 1 pair, got %d", got)
	}

	defs = append(defs, MetafieldDefinition{Validations: make([]MetafieldDefinitionValidation, 3)})
	if got := maxValidationPairs(defs); got != 3 {
		t.Errorf("want 3 pairs, got %d", got)
	}
}

func TestWriteDefinitionsCSV(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "shop-metafield-definitions.csv")

	defs := []MetafieldDefinition{
		{OwnerType: "PRODUCT", Name: "Material", Key: "material", Type: "single_line_text_field", Namespace: "custom"},
	}

	if err := writeDefinitionsCSV(filename, defs); err != nil {
		t.Fatalf("write failed: %s", err)
	}

	file, err := os.Open(filename)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	records, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatalf("cannot read CSV: %s", err)
	}
	if len(records) != 2 {
		t.Fatalf("want header and 1 row, got %d records", len(records))
	}
	if len(records[0]) != len(definitionExportHeader(1)) {
		t.Errorf("header has %d columns, want %d", len(records[0]), len(definitionExportHeader(1)))
	}
	if records[1][0] != "PRODUCT" || records[1][1] != "Material" || records[1][2] != "material" {
		t.Errorf("row wrong: %v", records[1])
	}
}
