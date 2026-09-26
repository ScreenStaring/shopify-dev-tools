package metaobjects

import (
	"reflect"
	"testing"
)

func TestParseEntryRef(t *testing.T) {
	tests := []struct {
		name string
		arg  string
		want *entryRef
	}{
		{
			name: "numeric id",
			arg:  "12345",
			want: &entryRef{Arg: "12345", ID: "12345"},
		},
		{
			name: "GID",
			arg:  "gid://shopify/Metaobject/12345",
			want: &entryRef{Arg: "gid://shopify/Metaobject/12345", ID: "gid://shopify/Metaobject/12345"},
		},
		{
			name: "type and handle",
			arg:  "author:jane-austen",
			want: &entryRef{Arg: "author:jane-austen", Type: "author", Handle: "jane-austen"},
		},
		{
			name: "app owned type keeps its prefix",
			arg:  "$app:author:jane-austen",
			want: &entryRef{Arg: "$app:author:jane-austen", Type: "$app:author", Handle: "jane-austen"},
		},
		{
			name: "handle with slash",
			arg:  "author:parent/child",
			want: &entryRef{Arg: "author:parent/child", Type: "author", Handle: "parent/child"},
		},
		{
			name: "surrounding whitespace",
			arg:  " author:jane-austen ",
			want: &entryRef{Arg: "author:jane-austen", Type: "author", Handle: "jane-austen"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseEntryRef(tt.arg)
			if err != nil {
				t.Fatalf("parseEntryRef(%q): %s", tt.arg, err)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Got %+v, expected %+v", got, tt.want)
			}
		})
	}
}

func TestParseEntryRefInvalid(t *testing.T) {
	args := []string{
		"",
		"   ",
		"jane-austen",
		"author:",
		":jane-austen",
		"gid://shopify/MetaobjectDefinition/12345",
		"gid://shopify/Product/12345",
	}

	for _, arg := range args {
		t.Run(arg, func(t *testing.T) {
			if ref, err := parseEntryRef(arg); err == nil {
				t.Errorf("Expected error for %q, got %+v", arg, ref)
			}
		})
	}
}
