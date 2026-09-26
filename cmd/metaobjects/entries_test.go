package metaobjects

import (
	"reflect"
	"strings"
	"testing"

	"github.com/ScreenStaring/shopify-dev-tools/cmd/metaobjects/gql"
)

var entryColumns = []string{colEntryType, colEntryHandle, "name", "bio", "photo", colEntryID, colEntryDisplayName, colEntryUpdatedAt}

func entryRow(m map[string]string) []string {
	out := make([]string, len(entryColumns))
	for i, c := range entryColumns {
		out[i] = m[c]
	}

	return out
}

func entryCSV(rows ...map[string]string) string {
	out := make([][]string, len(rows))
	for i, r := range rows {
		out[i] = entryRow(r)
	}

	return csvRaw(entryColumns, out...)
}

// definitionWith builds a definition of the given type whose fields are the
// given keys.
func definitionWith(moType string, keys ...string) *gql.MetaobjectDefinition {
	d := &gql.MetaobjectDefinition{Type: moType}
	for _, k := range keys {
		d.Fields = append(d.Fields, gql.MetaobjectFieldDefinition{Key: k})
	}

	return d
}

func TestParseEntryCSVHappyPath(t *testing.T) {
	csv := entryCSV(
		map[string]string{
			colEntryType: "author", colEntryHandle: "jane-austen", "name": "Jane Austen", "bio": "Novelist",
			colEntryID: "123", colEntryDisplayName: "Jane Austen", colEntryUpdatedAt: "2026-01-01",
		},
		map[string]string{
			colEntryType: "faq", colEntryHandle: "shipping", "name": "Shipping",
		},
	)

	entries, err := parseEntryCSV(strings.NewReader(csv))
	if err != nil {
		t.Fatalf("parseEntryCSV: %s", err)
	}

	expected := []*entryInput{
		{
			Row: 2, Type: "author", Handle: "jane-austen",
			Cells: map[string]string{"name": "Jane Austen", "bio": "Novelist"},
		},
		{
			Row: 3, Type: "faq", Handle: "shipping",
			Cells: map[string]string{"name": "Shipping"},
		},
	}

	if !reflect.DeepEqual(entries, expected) {
		t.Errorf("Got %+v, expected %+v", entries, expected)
	}
}

func TestParseEntryCSVIgnoresUnimportableColumns(t *testing.T) {
	csv := entryCSV(
		map[string]string{
			colEntryType: " Author ", colEntryHandle: " jane-austen ", colEntryID: "123",
			colEntryDisplayName: "Jane", colEntryUpdatedAt: "2026-01-01",
		},
	)

	entries, err := parseEntryCSV(strings.NewReader(csv))
	if err != nil {
		t.Fatalf("parseEntryCSV: %s", err)
	}

	if len(entries) != 1 {
		t.Fatalf("Got %d entries, expected 1", len(entries))
	}

	if entries[0].Type != "Author" {
		t.Errorf("Got type '%s', expected 'Author'", entries[0].Type)
	}
	if entries[0].Handle != "jane-austen" {
		t.Errorf("Got handle '%s', expected 'jane-austen'", entries[0].Handle)
	}
	if len(entries[0].Cells) != 0 {
		t.Errorf("Got cells %+v, expected none", entries[0].Cells)
	}
}

func TestParseEntryCSVCaseInsensitiveHeaders(t *testing.T) {
	csv := "TYPE,HANDLE,Name,BIO\nAuthor,jane-austen,Jane Austen,Novelist\n"

	entries, err := parseEntryCSV(strings.NewReader(csv))
	if err != nil {
		t.Fatalf("parseEntryCSV: %s", err)
	}

	expected := map[string]string{"name": "Jane Austen", "bio": "Novelist"}
	if !reflect.DeepEqual(entries[0].Cells, expected) {
		t.Errorf("Got %+v, expected %+v", entries[0].Cells, expected)
	}
}

func TestParseEntryCSVSkipsEmptyRows(t *testing.T) {
	csv := strings.Join([]string{
		"Type,Handle,name",
		"author,jane-austen,Jane Austen",
		",,",
		"   ,   ,   ",
	}, "\n")

	entries, err := parseEntryCSV(strings.NewReader(csv))
	if err != nil {
		t.Fatalf("parseEntryCSV: %s", err)
	}

	if len(entries) != 1 {
		t.Errorf("Got %d entries, expected 1", len(entries))
	}
}

func TestParseEntryCSVErrors(t *testing.T) {
	tests := []struct {
		name string
		csv  string
		want string
	}{
		{
			name: "empty input",
			csv:  "",
			want: "CSV must have a header row and at least one data row",
		},
		{
			name: "only a header row",
			csv:  "Type,Handle,name",
			want: "CSV must have a header row and at least one data row",
		},
		{
			name: "missing type column",
			csv:  "Handle,name\njane-austen,Jane Austen\n",
			want: "Missing required column 'type'",
		},
		{
			name: "duplicate column",
			csv:  "Type,Handle,name,Name\nauthor,jane-austen,Jane Austen,Jane\n",
			want: "Duplicate column 'name' in header",
		},
		{
			name: "duplicate column ignoring case",
			csv:  "Type,Handle,name,NAME\nauthor,jane-austen,Jane Austen,Jane\n",
			want: "Duplicate column 'name' in header",
		},
		{
			name: "empty type",
			csv:  "Type,Handle,name\n,jane-austen,Jane Austen\n",
			want: "Row 2: required column 'type' empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseEntryCSV(strings.NewReader(tt.csv))
			if err == nil {
				t.Fatalf("Expected error containing '%s', got none", tt.want)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("Got error '%s', expected it to contain '%s'", err, tt.want)
			}
		})
	}
}

func TestEntryFields(t *testing.T) {
	entries, err := parseEntryCSV(strings.NewReader("Type,Handle,Bio,name\nauthor,jane-austen,Novelist,Jane Austen\n"))
	if err != nil {
		t.Fatalf("parseEntryCSV: %s", err)
	}

	// Columns match field keys case-insensitively and are ordered by key.
	def := definitionWith("author", "name", "Bio")
	expected := []gql.MetaobjectField{
		{Key: "Bio", Value: "Novelist"},
		{Key: "name", Value: "Jane Austen"},
	}

	if fields := entryFields(entries[0], def); !reflect.DeepEqual(fields, expected) {
		t.Errorf("Got %+v, expected %+v", fields, expected)
	}
}

func TestCreateEntryJSON(t *testing.T) {
	e := &entryInput{Type: "author", Handle: "jane-austen", Cells: map[string]string{"name": "Jane Austen"}}
	fields := []gql.MetaobjectField{{Key: "name", Value: "Jane Austen"}}

	expected := map[string]interface{}{
		"type":   "author",
		"handle": "jane-austen",
		"fields": []map[string]interface{}{
			{"key": "name", "value": "Jane Austen"},
		},
	}

	if got := createEntryJSON(e, fields); !reflect.DeepEqual(got, expected) {
		t.Errorf("Got %+v, expected %+v", got, expected)
	}
}

func TestEntryInputJSONOmitsEmpty(t *testing.T) {
	// The upsert's type and handle are given to the mutation, not the input.
	e := &entryInput{Type: "author"}

	if got := entryInputJSON(e, nil); len(got) != 0 {
		t.Errorf("Got %+v, expected an empty input", got)
	}

	// A blank cell leaves the field alone instead of clearing it.
	fields := entryFields(&entryInput{Cells: map[string]string{}}, definitionWith("author", "name"))
	if len(fields) != 0 {
		t.Errorf("Got %+v, expected no fields", fields)
	}
}

func TestParseEntryCSVReadmeExample(t *testing.T) {
	csv := strings.Join([]string{
		"Type,Handle,name,bio,question,answer",
		`author,jane-austen,Jane Austen,"English novelist, wrote ""Pride and Prejudice""",,`,
		"faq,shipping-times,,,How long does shipping take?,Orders ship within 1-2 business days.",
	}, "\n")

	entries, err := parseEntryCSV(strings.NewReader(csv))
	if err != nil {
		t.Fatalf("parseEntryCSV: %s", err)
	}

	if len(entries) != 2 {
		t.Fatalf("Got %d entries, expected 2", len(entries))
	}

	if entries[1].Cells["answer"] != "Orders ship within 1-2 business days." {
		t.Errorf("Got '%s'", entries[1].Cells["answer"])
	}

}
