package metafields

import (
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/ScreenStaring/shopify-dev-tools/cmd"
)

// definitionExportColumns are the export's leading column names: those of the
// definition import CSV, in the order the README documents them. A Validation
// Name/Validation Value column pair per validation is appended to them.
var definitionExportColumns = []string{
	"Owner Type",
	"Name",
	"Key",
	"Type",
	"Namespace",
	"Description",
	"Pin",
	"Access Admin",
	"Access Customer Account",
	"Access Storefront",
	"Capability Admin Filterable",
	"Capability Smart Collection Condition",
	"Capability Unique Values",
	"Constraint Key",
	"Constraint Value",
}

// definitionExportHeader returns the export's columns with a Validation
// Name/Validation Value pair per validation. The import pairs validation
// columns by position, so repeating them exports every validation.
func definitionExportHeader(validationPairs int) []string {
	header := make([]string, len(definitionExportColumns), len(definitionExportColumns)+2*validationPairs)
	copy(header, definitionExportColumns)

	for i := 0; i < validationPairs; i++ {
		header = append(header, "Validation Name", "Validation Value")
	}

	return header
}

// definitionExportRow returns the spreadsheet row for one definition. Rows are
// padded so that every row has the same number of validation pairs.
func definitionExportRow(d MetafieldDefinition, validationPairs int) []string {
	constraintKey, constraintValues := "", ""
	if d.Constraints != nil {
		constraintKey = d.Constraints.Key
		// The import reads constraint values as a pipe-separated list.
		constraintValues = strings.Join(exportConstraintValues(d.Constraints), "|")
	}

	row := []string{
		d.OwnerType,
		d.Name,
		d.Key,
		d.Type,
		d.Namespace,
		d.Description,
		strconv.FormatBool(d.Pin),
		d.Access.Admin,
		d.Access.CustomerAccount,
		d.Access.Storefront,
		strconv.FormatBool(d.Capabilities.AdminFilterable),
		strconv.FormatBool(d.Capabilities.SmartCollectionCondition),
		strconv.FormatBool(d.Capabilities.UniqueValues),
		constraintKey,
		constraintValues,
	}

	for i := 0; i < validationPairs; i++ {
		if i < len(d.Validations) {
			row = append(row, d.Validations[i].Name, d.Validations[i].Value)
		} else {
			row = append(row, "", "")
		}
	}

	return row
}

// maxValidationPairs returns the number of validation column pairs the export
// needs: that of the definition with the most validations, at least one.
func maxValidationPairs(definitions []MetafieldDefinition) int {
	max := 1
	for _, d := range definitions {
		if len(d.Validations) > max {
			max = len(d.Validations)
		}
	}
	return max
}

func writeDefinitionsCSV(filename string, definitions []MetafieldDefinition) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("Cannot create CSV file: %s", err)
	}
	defer file.Close()

	w := csv.NewWriter(file)
	pairs := maxValidationPairs(definitions)

	if err := w.Write(definitionExportHeader(pairs)); err != nil {
		return fmt.Errorf("Cannot write CSV header: %s", err)
	}

	for _, d := range definitions {
		if err := w.Write(definitionExportRow(d, pairs)); err != nil {
			return fmt.Errorf("Cannot write CSV row for %s.%s: %s", d.Namespace, d.Key, err)
		}
	}

	w.Flush()

	return w.Error()
}

func exportDefinitionsAction(c *cli.Context) error {
	if c.NArg() == 0 {
		return errors.New("Resource name required")
	}

	client := cmd.NewGraphQLClient(c)
	namespace := c.String("namespace")

	var definitions []MetafieldDefinition
	var failures []string

	for _, arg := range c.Args().Slice() {
		// Owner types are case-insensitive; the API requires the uppercase enum.
		ownerType := strings.ToUpper(arg)

		defs, err := listMetafieldDefinitions(client, ownerType, namespace)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: %s", ownerType, err))
			continue
		}

		fmt.Fprintf(os.Stderr, "%s: %d definition(s)\n", ownerType, len(defs))
		definitions = append(definitions, defs...)
	}

	// Don't write an incomplete spreadsheet: the export is meant to be edited
	// and imported back.
	if len(failures) > 0 {
		return fmt.Errorf("Cannot export metafield definition(s): %s", strings.Join(failures, ", "))
	}

	filename := strings.SplitN(c.String("shop"), ".", 2)[0] + "-metafield-definitions.csv"

	if err := writeDefinitionsCSV(filename, definitions); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Wrote %s (%d definition(s))\n", filename, len(definitions))

	return nil
}
