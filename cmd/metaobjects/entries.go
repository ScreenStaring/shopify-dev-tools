package metaobjects

import (
	"bufio"
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
	"github.com/ScreenStaring/shopify-dev-tools/cmd/metaobjects/gql"
	gqlclient "github.com/ScreenStaring/shopify-dev-tools/gql"
	"github.com/urfave/cli/v2"
)

// CSV columns for metaobject import. Type and Handle are the columns the
// export spreadsheet uses, as are the ignored ones. Every other column is the
// key of one of the type's fields, so an exported spreadsheet can be edited
// and imported back. Columns are matched by name, case-insensitively, so
// their order in the spreadsheet does not matter.
const (
	colEntryType        = "type"
	colEntryHandle      = "handle"
	colEntryID          = "id"
	colEntryDisplayName = "display name"
	colEntryUpdatedAt   = "updated at"
)

// ignoredEntryColumns are columns the export spreadsheet has that hold no
// importable data: Shopify assigns them.
var ignoredEntryColumns = []string{colEntryID, colEntryDisplayName, colEntryUpdatedAt}

// entryInput is one metaobject to import, assembled from one spreadsheet row.
type entryInput struct {
	Row    int
	Type   string
	Handle string
	Cells  map[string]string // lowercased column name -> value; field columns only
}

func importEntryAction(c *cli.Context) error {
	upsert := c.Bool("upsert")
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

	entries, err := parseEntryCSV(reader)
	if err != nil {
		return err
	}

	if len(entries) == 0 {
		return errors.New("No metaobjects found in input")
	}

	client := cmd.NewGraphQLClient(c)

	defs, err := entryDefinitions(client, entries)
	if err != nil {
		return err
	}

	done, failed := 0, 0
	verb, doneVerb := "created", "Created"
	if upsert {
		verb, doneVerb = "upserted", "Upserted"
	}
	jsonResults := make([]map[string]interface{}, 0, len(entries))

	for _, e := range entries {
		fields := entryFields(e, defs[e.Type])

		var id string
		if upsert {
			id, err = gql.UpsertMetaobject(client, e.Type, e.Handle, entryInputJSON(e, fields))
		} else {
			id, err = gql.CreateMetaobject(client, createEntryJSON(e, fields))
		}

		if err != nil {
			failed++
			if jsonOutput {
				jsonResults = append(jsonResults, map[string]interface{}{
					"row":    e.Row,
					"type":   e.Type,
					"handle": e.Handle,
					"id":     "",
					"status": "error",
					"errors": []string{err.Error()},
				})
			} else {
				fmt.Fprintf(os.Stderr, "Row %d %s: Error: %s\n", e.Row, entryLabel(e), err)
			}
			continue
		}

		done++
		id = strings.TrimPrefix(id, gql.MetaobjectGIDPrefix)
		if jsonOutput {
			jsonResults = append(jsonResults, map[string]interface{}{
				"row":    e.Row,
				"type":   e.Type,
				"handle": e.Handle,
				"id":     id,
				"status": "ok",
			})
		} else {
			fmt.Printf("Row %3d %s: %s (%s)\n", e.Row, entryLabel(e), doneVerb, id)
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
		return fmt.Errorf("%d of %d metaobjects %s, %d failed", done, len(entries), verb, failed)
	}

	fmt.Printf("%d metaobject(s) %s\n", done, verb)
	return nil
}

// parseEntryCSV reads metaobject entries from a CSV spreadsheet. Each row is
// one metaobject.
func parseEntryCSV(reader io.Reader) ([]*entryInput, error) {
	r := csv.NewReader(reader)
	r.FieldsPerRecord = -1
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("Invalid CSV: %s", err)
	}
	if len(records) < 2 {
		return nil, errors.New("CSV must have a header row and at least one data row")
	}

	cols := make(map[string]int, len(records[0]))
	for i, h := range records[0] {
		h = strings.ToLower(strings.TrimSpace(h))
		if h == "" {
			continue
		}
		if _, dup := cols[h]; dup {
			return nil, fmt.Errorf("Duplicate column '%s' in header", h)
		}
		cols[h] = i
	}

	if _, ok := cols[colEntryType]; !ok {
		return nil, fmt.Errorf("Missing required column '%s'", colEntryType)
	}

	var entries []*entryInput
	for n, row := range records[1:] {
		rowNum := n + 2
		moType := strings.TrimSpace(getCell(row, cols, colEntryType))
		if moType == "" {
			// Spreadsheets often end with rows of nothing but separators.
			if rowIsEmpty(row) {
				continue
			}
			return nil, fmt.Errorf("Row %d: required column '%s' empty", rowNum, colEntryType)
		}

		e := &entryInput{
			Row:    rowNum,
			Type:   moType,
			Handle: strings.TrimSpace(getCell(row, cols, colEntryHandle)),
			Cells:  map[string]string{},
		}

		for col, i := range cols {
			if reservedEntryColumn(col) || i >= len(row) {
				continue
			}
			// A blank cell doesn't set the field: it's left out of the request
			// so an existing value isn't cleared when updating.
			if v := strings.TrimSpace(row[i]); v != "" {
				e.Cells[col] = v
			}
		}

		entries = append(entries, e)
	}

	return entries, nil
}

// entryDefinitions fetches the definition of every type the spreadsheet
// imports. A type the shop has no definition for maps to nil, which leaves the
// entry without fields and the API to reject the type.
func entryDefinitions(client *gqlclient.Client, entries []*entryInput) (map[string]*gql.MetaobjectDefinition, error) {
	defs := map[string]*gql.MetaobjectDefinition{}

	for _, e := range entries {
		if _, ok := defs[e.Type]; ok {
			continue
		}

		d, err := gql.MetaobjectDefinitionByType(client, e.Type)
		if err != nil {
			return nil, err
		}
		defs[e.Type] = d
	}

	return defs, nil
}

// entryFields returns the fields the spreadsheet sets for an entry, ordered by
// key so requests and output are stable. A type with no definition has no
// fields to map, so the request carries none and the API reports the type.
func entryFields(e *entryInput, d *gql.MetaobjectDefinition) []gql.MetaobjectField {
	if d == nil {
		return nil
	}

	keys := entryFieldKeys(d)

	columns := make([]string, 0, len(e.Cells))
	for col := range e.Cells {
		columns = append(columns, col)
	}
	sort.Strings(columns)

	fields := make([]gql.MetaobjectField, 0, len(columns))
	for _, col := range columns {
		fields = append(fields, gql.MetaobjectField{Key: keys[col], Value: e.Cells[col]})
	}

	return fields
}

// entryInputJSON builds an entry's MetaobjectFields as a MetaobjectCreateInput,
// or as a MetaobjectUpsertInput when upserting (the upsert's type and handle
// are given to the mutation instead).
func entryInputJSON(e *entryInput, fields []gql.MetaobjectField) map[string]interface{} {
	metaobject := map[string]interface{}{}

	if e.Handle != "" {
		metaobject["handle"] = e.Handle
	}

	if len(fields) > 0 {
		values := make([]map[string]interface{}, 0, len(fields))
		for _, f := range fields {
			values = append(values, map[string]interface{}{"key": f.Key, "value": f.Value})
		}
		metaobject["fields"] = values
	}

	return metaobject
}

// createEntryJSON builds the MetaobjectCreateInput for an entry.
func createEntryJSON(e *entryInput, fields []gql.MetaobjectField) map[string]interface{} {
	metaobject := entryInputJSON(e, fields)
	metaobject["type"] = e.Type

	return metaobject
}

// entryFieldKeys maps each of a definition's field keys, lowercased, to the
// key itself, so spreadsheet columns match them case-insensitively.
func entryFieldKeys(d *gql.MetaobjectDefinition) map[string]string {
	keys := make(map[string]string, len(d.Fields))
	for _, f := range d.Fields {
		keys[strings.ToLower(f.Key)] = f.Key
	}

	return keys
}

// entryLabel identifies an entry in output: its type, and its handle when it
// has one.
func entryLabel(e *entryInput) string {
	if e.Handle == "" {
		return e.Type
	}

	return e.Type + "/" + e.Handle
}

// reservedEntryColumn reports whether a column holds metadata rather than a
// field value.
func reservedEntryColumn(name string) bool {
	if name == colEntryType || name == colEntryHandle {
		return true
	}

	for _, col := range ignoredEntryColumns {
		if col == name {
			return true
		}
	}

	return false
}

func rowIsEmpty(row []string) bool {
	for _, cell := range row {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}

	return true
}

// entryRef is one metaobject to delete, as given on the command line or read
// from stdin: an id, a GID, or a type and handle.
type entryRef struct {
	Arg    string // as given, used in output and errors
	ID     string // numeric id or GID, empty when the reference gave a handle
	Type   string
	Handle string
}

// parseEntryRef parses a metaobject reference. A reference is a numeric id, a
// GID, or a 'type:handle' pair. The type is everything before the last ':' so
// app owned types keep their '$app:' prefix.
func parseEntryRef(arg string) (*entryRef, error) {
	arg = strings.TrimSpace(arg)
	if arg == "" {
		return nil, errors.New("Metaobject id, GID, or type:handle required")
	}

	if strings.HasPrefix(arg, "gid://") {
		if !strings.HasPrefix(arg, gql.MetaobjectGIDPrefix) {
			return nil, fmt.Errorf("Argument '%s' invalid: must be a metaobject id, GID, or 'type:handle'", arg)
		}

		return &entryRef{Arg: arg, ID: arg}, nil
	}

	if _, err := strconv.ParseInt(arg, 10, 64); err == nil {
		return &entryRef{Arg: arg, ID: arg}, nil
	}

	i := strings.LastIndex(arg, ":")
	if i < 0 || i == 0 || i == len(arg)-1 {
		return nil, fmt.Errorf("Argument '%s' invalid: must be a metaobject id, GID, or 'type:handle'", arg)
	}

	return &entryRef{Arg: arg, Type: arg[:i], Handle: arg[i+1:]}, nil
}

// deleteEntryAction deletes each of the given metaobjects, by numeric id, GID
// or 'type:handle'.
func deleteEntryAction(c *cli.Context) error {
	refs, err := entryRefs(c.Args().Slice())
	if err != nil {
		return err
	}

	if len(refs) == 0 {
		return errors.New("Metaobject id, GID, or type:handle required")
	}

	client := cmd.NewGraphQLClient(c)

	var failures []string
	for _, ref := range refs {
		id, err := deleteEntry(client, ref)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: %s", ref.Arg, err))
			continue
		}

		if ref.ID == "" {
			fmt.Printf("Deleted %s (%s)\n", ref.Arg, id)
		} else {
			fmt.Printf("Deleted %s\n", id)
		}
	}

	if len(failures) > 0 {
		return fmt.Errorf("Cannot delete metaobject(s): %s", strings.Join(failures, ", "))
	}

	return nil
}

// entryRefs returns the metaobjects the command deletes: its arguments, or one
// per line of stdin when it has none.
func entryRefs(args []string) ([]*entryRef, error) {
	if len(args) > 0 {
		refs := make([]*entryRef, 0, len(args))
		for _, arg := range args {
			ref, err := parseEntryRef(arg)
			if err != nil {
				return nil, err
			}
			refs = append(refs, ref)
		}

		return refs, nil
	}

	var refs []*entryRef
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		ref, err := parseEntryRef(line)
		if err != nil {
			return nil, err
		}
		refs = append(refs, ref)
	}

	return refs, scanner.Err()
}

// deleteEntry deletes one metaobject: the one of the reference's id, or the one
// of its type and handle, and returns the deleted metaobject's numeric id.
func deleteEntry(client *gqlclient.Client, ref *entryRef) (string, error) {
	id := ref.ID
	if id == "" {
		mo, err := gql.MetaobjectByHandle(client, ref.Type, ref.Handle)
		if err != nil {
			return "", err
		}
		if mo == nil {
			return "", errors.New("no metaobject with that type and handle")
		}

		id = mo.ID
	}

	deletedID, err := gql.DeleteMetaobject(client, id)
	if err != nil {
		return "", err
	}

	return strings.TrimPrefix(deletedID, gql.MetaobjectGIDPrefix), nil
}
