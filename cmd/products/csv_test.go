package products

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
)

func writeCSV(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp("", "products*.csv")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	f.Close()
	t.Cleanup(func() { os.Remove(f.Name()) })
	return f.Name()
}

func TestParseCSVBasicProduct(t *testing.T) {
	csv := "handle,title,vendor,type,body (html),tags,status\n" +
		"chair,Chair,Acme Furniture,seating,<p>Comfy</p>,\"wood, oak\",active\n"
	prods, err := parseCSV(writeCSV(t, csv), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(prods) != 1 {
		t.Fatalf("len(prods) = %d, want 1", len(prods))
	}
	p := prods[0].Input
	if got := p.Handle; got != "chair" {
		t.Errorf("handle = %q, want %q", got, "chair")
	}
	if got := p.Title; got != "Chair" {
		t.Errorf("title = %q, want %q", got, "Chair")
	}
	if got := p.Vendor; got != "Acme Furniture" {
		t.Errorf("vendor = %q, want %q", got, "Acme Furniture")
	}
	if got := p.ProductType; got != "seating" {
		t.Errorf("type = %q, want %q", got, "seating")
	}
	if got := p.DescriptionHTML; got != "<p>Comfy</p>" {
		t.Errorf("body (html) = %q, want %q", got, "<p>Comfy</p>")
	}
	if got := p.Status; got != "ACTIVE" {
		t.Errorf("status = %q, want %q", got, "ACTIVE")
	}
	wantTags := []string{"wood", "oak"}
	if got := p.Tags; !reflect.DeepEqual(got, wantTags) {
		t.Errorf("tags = %v, want %v", got, wantTags)
	}
}

func TestParseCSVProductIDIdentifier(t *testing.T) {
	csv := "handle,product id\n" +
		"chair,123\n" +
		"desk,gid://shopify/Product/456\n"
	prods, err := parseCSV(writeCSV(t, csv), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(prods) != 2 {
		t.Fatalf("len(prods) = %d, want 2", len(prods))
	}
	if got := prods[0].Identifier; got == nil || got.ID != "gid://shopify/Product/123" {
		t.Errorf("numeric id identifier = %+v, want gid://shopify/Product/123", got)
	}
	if got := prods[1].Identifier; got == nil || got.ID != "gid://shopify/Product/456" {
		t.Errorf("gid identifier = %+v, want gid://shopify/Product/456", got)
	}
	b, err := json.Marshal(prods[0].Input)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), `"id"`) {
		t.Errorf("marshaled input = %s, want no id field", b)
	}
}

func TestParseCSVVariants(t *testing.T) {
	csv := "handle,Variant SKU,Variant Price,Variant Compare At Price,Variant Barcode,Variant Taxable,Variant Inventory Policy\n" +
		"chair,CHAIR-BLK,199.99,249.99,123456,true,CONTINUE\n" +
		",CHAIR-WHT,189.99,,654321,false,DENY\n"
	prods, err := parseCSV(writeCSV(t, csv), nil)
	if err != nil {
		t.Fatal(err)
	}
	variants := prods[0].Input.Variants
	if len(variants) != 2 {
		t.Fatalf("len(variants) = %d, want 2", len(variants))
	}
	v := variants[0]
	if got := v.SKU; got != "CHAIR-BLK" {
		t.Errorf("variant1 sku = %q, want %q", got, "CHAIR-BLK")
	}
	if got := v.Price; got != "199.99" {
		t.Errorf("variant1 price = %q, want %q", got, "199.99")
	}
	if got := v.CompareAtPrice; got != "249.99" {
		t.Errorf("variant1 compare at price = %q, want %q", got, "249.99")
	}
	if got := v.Barcode; got != "123456" {
		t.Errorf("variant1 barcode = %q, want %q", got, "123456")
	}
	if got := v.Taxable; got == nil || *got != true {
		t.Errorf("variant1 taxable = %v, want true", got)
	}
	if got := v.InventoryPolicy; got != "CONTINUE" {
		t.Errorf("variant1 inventory policy = %q, want %q", got, "CONTINUE")
	}
	v2 := variants[1]
	if got := v2.SKU; got != "CHAIR-WHT" {
		t.Errorf("variant2 sku = %q, want %q", got, "CHAIR-WHT")
	}
	if got := v2.Price; got != "189.99" {
		t.Errorf("variant2 price = %q, want %q", got, "189.99")
	}
	if got := v2.CompareAtPrice; got != "" {
		t.Errorf("variant2 compare at price = %q, want empty", got)
	}
	if got := v2.Barcode; got != "654321" {
		t.Errorf("variant2 barcode = %q, want %q", got, "654321")
	}
	if got := v2.Taxable; got == nil || *got != false {
		t.Errorf("variant2 taxable = %v, want false", got)
	}
	if got := v2.InventoryPolicy; got != "DENY" {
		t.Errorf("variant2 inventory policy = %q, want %q", got, "DENY")
	}
}

func TestParseCSVContinueSelling(t *testing.T) {
	csv := "handle,Variant SKU,Continue Selling When Out Of Stock\n" +
		"chair,CHAIR-BLK,true\n" +
		",CHAIR-WHT,false\n"
	prods, err := parseCSV(writeCSV(t, csv), nil)
	if err != nil {
		t.Fatal(err)
	}
	variants := prods[0].Input.Variants
	if len(variants) != 2 {
		t.Fatalf("len(variants) = %d, want 2", len(variants))
	}
	if got := variants[0].InventoryPolicy; got != "CONTINUE" {
		t.Errorf("continue=true policy = %q, want %q", got, "CONTINUE")
	}
	if got := variants[1].InventoryPolicy; got != "DENY" {
		t.Errorf("continue=false policy = %q, want %q", got, "DENY")
	}
}

func TestParseCSVOptions(t *testing.T) {
	csv := "handle,Option1 Name,Option1 Value,Option2 Name,Option2 Value,Variant SKU\n" +
		"chair,Color,Black,Size,L,CHAIR-BLK-L\n" +
		",Color,Black,Size,M,CHAIR-BLK-M\n" +
		",Color,White,Size,L,CHAIR-WHT-L\n"
	prods, err := parseCSV(writeCSV(t, csv), nil)
	if err != nil {
		t.Fatal(err)
	}
	p := prods[0].Input

	wantOptions := []optionCreateInput{
		{Name: "Color", Values: []optionValueInput{{Name: "Black"}, {Name: "White"}}},
		{Name: "Size", Values: []optionValueInput{{Name: "L"}, {Name: "M"}}},
	}
	if got := p.ProductOptions; !reflect.DeepEqual(got, wantOptions) {
		t.Errorf("product options = %+v, want %+v", got, wantOptions)
	}

	if len(p.Variants) != 3 {
		t.Fatalf("len(variants) = %d, want 3", len(p.Variants))
	}
	wantOpts := []variantOptionValue{
		{OptionName: "Color", Name: "Black"},
		{OptionName: "Size", Name: "L"},
	}
	if got := p.Variants[0].OptionValues; !reflect.DeepEqual(got, wantOpts) {
		t.Errorf("variant1 option values = %+v, want %+v", got, wantOpts)
	}
	if got := p.Variants[1].OptionValues; got[1].Name != "M" {
		t.Errorf("variant2 option2 value = %q, want %q", got[1].Name, "M")
	}
	if got := p.Variants[2].OptionValues; got[1].Name != "L" {
		t.Errorf("variant3 option2 value = %q, want %q", got[1].Name, "L")
	}
}

func TestParseCSVDefaultTitleVariant(t *testing.T) {
	csv := "handle,title,Variant SKU\n" +
		"chair,Chair,CHAIR\n"
	prods, err := parseCSV(writeCSV(t, csv), nil)
	if err != nil {
		t.Fatal(err)
	}
	p := prods[0].Input

	wantOptions := []optionCreateInput{
		{Name: "Title", Values: []optionValueInput{{Name: "Default Title"}}},
	}
	if got := p.ProductOptions; !reflect.DeepEqual(got, wantOptions) {
		t.Errorf("product options = %+v, want %+v", got, wantOptions)
	}
	if len(p.Variants) != 1 {
		t.Fatalf("len(variants) = %d, want 1", len(p.Variants))
	}
	wantOpts := []variantOptionValue{{OptionName: "Title", Name: "Default Title"}}
	if got := p.Variants[0].OptionValues; !reflect.DeepEqual(got, wantOpts) {
		t.Errorf("variant option values = %+v, want %+v", got, wantOpts)
	}
}

func TestParseCSVInventory(t *testing.T) {
	locations := map[string]string{"Main": "gid://shopify/Location/1", "Second": "gid://shopify/Location/2"}
	csv := "handle,Location,Available,On Hand,Requires Shipping,Unit Cost,Variant SKU\n" +
		"chair,Main,5,,true,150.00,CHAIR-BLK\n" +
		",Second,,3,false,160.00,CHAIR-WHT\n"
	prods, err := parseCSV(writeCSV(t, csv), locations)
	if err != nil {
		t.Fatal(err)
	}
	variants := prods[0].Input.Variants
	if len(variants) != 2 {
		t.Fatalf("len(variants) = %d, want 2", len(variants))
	}

	v := variants[0]
	if got := v.InventoryItem.Tracked; got == nil || *got != true {
		t.Errorf("variant1 tracked = %v, want true", got)
	}
	if got := v.InventoryItem.RequiresShipping; got == nil || *got != true {
		t.Errorf("variant1 requires shipping = %v, want true", got)
	}
	if got := v.InventoryItem.Cost; got != "150.00" {
		t.Errorf("variant1 unit cost = %q, want %q", got, "150.00")
	}
	wantQty := []inventoryQuantityInput{
		{LocationID: "gid://shopify/Location/1", Name: "available", Quantity: 5},
	}
	if got := v.InventoryQuantities; !reflect.DeepEqual(got, wantQty) {
		t.Errorf("variant1 quantities = %+v, want %+v", got, wantQty)
	}

	v2 := variants[1]
	if got := v2.InventoryItem.Tracked; got == nil || *got != true {
		t.Errorf("variant2 tracked = %v, want true", got)
	}
	if got := v2.InventoryItem.RequiresShipping; got == nil || *got != false {
		t.Errorf("variant2 requires shipping = %v, want false", got)
	}
	if got := v2.InventoryItem.Cost; got != "160.00" {
		t.Errorf("variant2 unit cost = %q, want %q", got, "160.00")
	}
	wantQty2 := []inventoryQuantityInput{
		{LocationID: "gid://shopify/Location/2", Name: "on_hand", Quantity: 3},
	}
	if got := v2.InventoryQuantities; !reflect.DeepEqual(got, wantQty2) {
		t.Errorf("variant2 quantities = %+v, want %+v", got, wantQty2)
	}
}

func TestParseCSVProductImage(t *testing.T) {
	csv := "handle,Product Image URL\n" +
		"chair,https://example.com/chair.png\n"
	prods, err := parseCSV(writeCSV(t, csv), nil)
	if err != nil {
		t.Fatal(err)
	}
	wantFiles := []fileInput{{OriginalSource: "https://example.com/chair.png", ContentType: "IMAGE"}}
	if got := prods[0].Input.Files; !reflect.DeepEqual(got, wantFiles) {
		t.Errorf("files = %+v, want %+v", got, wantFiles)
	}
}

func TestParseCSVUnknownLocation(t *testing.T) {
	csv := "handle,Location,Available,Variant SKU\n" +
		"chair,Nowhere,5,CHAIR-BLK\n"
	_, err := parseCSV(writeCSV(t, csv), map[string]string{"Main": "gid://shopify/Location/1"})
	if err == nil {
		t.Fatal("err = nil, want error for unknown location")
	}
	if got := err.Error(); got != `Unknown location "Nowhere"` {
		t.Errorf("err = %q, want %q", got, `Unknown location "Nowhere"`)
	}
}

func TestParseCSVInvalidQuantity(t *testing.T) {
	csv := "handle,Location,Available,Variant SKU\n" +
		"chair,Main,abc,CHAIR-BLK\n"
	_, err := parseCSV(writeCSV(t, csv), map[string]string{"Main": "gid://shopify/Location/1"})
	if err == nil {
		t.Fatal("err = nil, want error for invalid quantity")
	}
	want := `Invalid available value "abc": strconv.Atoi: parsing "abc": invalid syntax`
	if got := err.Error(); got != want {
		t.Errorf("err = %q, want %q", got, want)
	}
}

func TestParseCSVNoIDColumns(t *testing.T) {
	csv := "Title,Variant SKU,Option1 Name,Option1 Value,Metafield Owner,Metafield Type,Metafield Namespace,Metafield Key,Metafield Value,Vendor,Body\n" +
		"Add Metafield,ADD,Title,Default Title,Product,string,shopifyapi-metafields,testsuite,foo bar,shopifyapi-metafields,Used by tests\n" +
		"Update Metafield,UPDATE,Title,Default Title,Product,integer,shopifyapi-metafields,testsuite,123,shopifyapi-metafields,Used by tests\n"
	prods, err := parseCSV(writeCSV(t, csv), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(prods) != 2 {
		t.Fatalf("len(prods) = %d, want 2", len(prods))
	}

	p := prods[0].Input
	if got := p.Title; got != "Add Metafield" {
		t.Errorf("title = %q, want %q", got, "Add Metafield")
	}
	if got := p.Vendor; got != "shopifyapi-metafields" {
		t.Errorf("vendor = %q, want %q", got, "shopifyapi-metafields")
	}
	if got := p.DescriptionHTML; got != "Used by tests" {
		t.Errorf("body = %q, want %q", got, "Used by tests")
	}
	if len(p.Variants) != 1 {
		t.Fatalf("len(variants) = %d, want 1", len(p.Variants))
	}
	if got := p.Variants[0].SKU; got != "ADD" {
		t.Errorf("variant sku = %q, want %q", got, "ADD")
	}
	wantMfs := []metafieldInput{
		{Namespace: "shopifyapi-metafields", Key: "testsuite", Value: "foo bar", Type: "string"},
	}
	if got := p.Metafields; !reflect.DeepEqual(got, wantMfs) {
		t.Errorf("metafields = %+v, want %+v", got, wantMfs)
	}

	if got := prods[1].Input.Title; got != "Update Metafield" {
		t.Errorf("product2 title = %q, want %q", got, "Update Metafield")
	}
	wantMfs2 := []metafieldInput{
		{Namespace: "shopifyapi-metafields", Key: "testsuite", Value: "123", Type: "integer"},
	}
	if got := prods[1].Input.Metafields; !reflect.DeepEqual(got, wantMfs2) {
		t.Errorf("product2 metafields = %+v, want %+v", got, wantMfs2)
	}
}

func TestParseCSVNoIDColumnsMetafieldRow(t *testing.T) {
	csv := "Title,Variant SKU,Metafield Owner,Metafield Type,Metafield Namespace,Metafield Key,Metafield Value\n" +
		"Add Metafield,ADD,,,,,\n" +
		",,Product,string,shopifyapi-metafields,testsuite,foo bar\n"
	prods, err := parseCSV(writeCSV(t, csv), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(prods) != 1 {
		t.Fatalf("len(prods) = %d, want 1", len(prods))
	}
	p := prods[0].Input
	if got := p.Title; got != "Add Metafield" {
		t.Errorf("title = %q, want %q", got, "Add Metafield")
	}
	wantMfs := []metafieldInput{
		{Namespace: "shopifyapi-metafields", Key: "testsuite", Value: "foo bar", Type: "string"},
	}
	if got := p.Metafields; !reflect.DeepEqual(got, wantMfs) {
		t.Errorf("metafields = %+v, want %+v", got, wantMfs)
	}
}

func TestParseCSVMetafields(t *testing.T) {
	csv := "handle,title,Variant SKU,Metafield Type,Metafield Namespace,Metafield Key,Metafield Value,Metafield Owner\n" +
		"chair,Chair,CHAIR-BLK,,,,,\n" +
		",,,single_line_text_field,custom,material,oak,Product\n" +
		",,,number_integer,custom,weight,12,Product\n" +
		",,,single_line_text_field,custom,finish,black,Variant\n" +
		",,CHAIR-WHT,,,,,\n" +
		",,,single_line_text_field,custom,finish,white,Variant\n"
	prods, err := parseCSV(writeCSV(t, csv), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(prods) != 1 {
		t.Fatalf("len(prods) = %d, want 1", len(prods))
	}

	wantMfs := []metafieldInput{
		{Namespace: "custom", Key: "material", Value: "oak", Type: "single_line_text_field"},
		{Namespace: "custom", Key: "weight", Value: "12", Type: "number_integer"},
	}
	if got := prods[0].Input.Metafields; !reflect.DeepEqual(got, wantMfs) {
		t.Errorf("product metafields = %+v, want %+v", got, wantMfs)
	}

	variants := prods[0].Input.Variants
	if len(variants) != 2 {
		t.Fatalf("len(variants) = %d, want 2", len(variants))
	}
	if got := variants[0].SKU; got != "CHAIR-BLK" {
		t.Errorf("variant1 sku = %q, want %q", got, "CHAIR-BLK")
	}
	wantVmf := []metafieldInput{
		{Namespace: "custom", Key: "finish", Value: "black", Type: "single_line_text_field"},
	}
	if got := variants[0].Metafields; !reflect.DeepEqual(got, wantVmf) {
		t.Errorf("variant1 metafields = %+v, want %+v", got, wantVmf)
	}
	if got := variants[1].SKU; got != "CHAIR-WHT" {
		t.Errorf("variant2 sku = %q, want %q", got, "CHAIR-WHT")
	}
	wantVmf2 := []metafieldInput{
		{Namespace: "custom", Key: "finish", Value: "white", Type: "single_line_text_field"},
	}
	if got := variants[1].Metafields; !reflect.DeepEqual(got, wantVmf2) {
		t.Errorf("variant2 metafields = %+v, want %+v", got, wantVmf2)
	}
}
