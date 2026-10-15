package charges

import (
	"encoding/json"
	"io"
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestNormalizeCurrencyCode(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		want    string
		wantErr bool
	}{
		{name: "unset", code: "", want: ""},
		{name: "uppercase", code: "USD", want: "USD"},
		{name: "lowercase is upcased", code: "usd", want: "USD"},
		{name: "surrounding space is trimmed", code: " eur ", want: "EUR"},
		{name: "too short", code: "US", wantErr: true},
		{name: "too long", code: "USDX", wantErr: true},
		{name: "not alphabetic", code: "US1", wantErr: true},
		{name: "only space", code: " ", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeCurrencyCode(tt.code)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("want error for %q, got %q", tt.code, got)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("normalizeCurrencyCode(%q) = %q, want %q", tt.code, got, tt.want)
			}
		})
	}
}

func TestPriceWithCurrency(t *testing.T) {
	tests := []struct {
		name     string
		price    string
		currency string
		want     string
	}{
		{name: "amount and currency", price: "4.55", currency: "USD", want: "4.55 USD"},
		{name: "no currency", price: "4.55", currency: "", want: "4.55"},
		{name: "no amount", price: "", currency: "USD", want: " USD"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := priceWithCurrency(tt.price, tt.currency); got != tt.want {
				t.Errorf("priceWithCurrency(%q, %q) = %q, want %q", tt.price, tt.currency, got, tt.want)
			}
		})
	}
}

// The charge's currency is only in the money object the API returns, so the
// node conversions (and their JSON tags) must carry it through to the
// printable charge.
func TestOneTimeChargeFromNodeCurrency(t *testing.T) {
	node := oneTimeChargeNodeJSONFromJSON(t, `{
		"id": "gid://shopify/AppPurchaseOneTime/1234",
		"name": "Test",
		"price": {"amount": "4.55", "currencyCode": "USD"}
	}`)

	charge := oneTimeChargeFromNode(&node, "", "")

	if charge.Price != "4.55" || charge.Currency != "USD" {
		t.Errorf("got price %q currency %q, want price \"4.55\" currency \"USD\"", charge.Price, charge.Currency)
	}
}

func TestRecurringChargeFromNodeCurrency(t *testing.T) {
	var node recurringChargeNodeJSON
	unmarshalNode(t, `{
		"id": "gid://shopify/AppSubscription/1234",
		"name": "Test",
		"lineItems": [
			{"plan": {"pricingDetails": {"price": {"amount": "10.00", "currencyCode": "EUR"}, "interval": "ANNUAL"}}}
		]
	}`, &node)

	charge := recurringChargeFromNode(node)

	if charge.Price != "10.00" || charge.Currency != "EUR" {
		t.Errorf("got price %q currency %q, want price \"10.00\" currency \"EUR\"", charge.Price, charge.Currency)
	}
}

// A charge whose price the API didn't return a currency for must still
// print, just without one.
func TestOneTimeChargeFromNodeWithoutPrice(t *testing.T) {
	node := oneTimeChargeNodeJSONFromJSON(t, `{
		"id": "gid://shopify/AppPurchaseOneTime/1234",
		"name": "Test"
	}`)

	charge := oneTimeChargeFromNode(&node, "", "")

	if charge.Price != "" || charge.Currency != "" {
		t.Errorf("got price %q currency %q, want both empty", charge.Price, charge.Currency)
	}
}

func oneTimeChargeNodeJSONFromJSON(t *testing.T, raw string) oneTimeChargeNodeJSON {
	t.Helper()

	var node oneTimeChargeNodeJSON
	unmarshalNode(t, raw, &node)

	return node
}

func unmarshalNode(t *testing.T, raw string, node interface{}) {
	t.Helper()

	if err := json.Unmarshal([]byte(raw), node); err != nil {
		t.Fatalf("cannot unmarshal node: %v", err)
	}
}

// captureStdout runs f with os.Stdout redirected to a pipe and returns what
// it wrote. tabby binds os.Stdout when its writer is created, so f must
// create the tabby instance itself.
func captureStdout(t *testing.T, f func()) string {
	t.Helper()

	original := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("cannot create pipe: %v", err)
	}
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = original

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("cannot read captured output: %v", err)
	}

	return string(out)
}

// priceLineRe matches the "Price" line of formatted charge output. The
// price can contain a space (e.g. "4.55 USD"), so trailing whitespace is
// trimmed by the capture rather than excluded by the pattern.
var priceLineRe = regexp.MustCompile(`(?m)^Price\s+(\S.*?)\s*$`)

// priceLine returns the value of the "Price" line in formatted charge
// output, ignoring the column alignment tabby applies.
func priceLine(t *testing.T, out string) string {
	t.Helper()

	match := priceLineRe.FindStringSubmatch(out)
	if match == nil {
		t.Fatalf("no Price line in output: %q", out)
	}

	return match[1]
}

func TestPrintFormattedChargesShowCurrencyAfterAmount(t *testing.T) {
	out := captureStdout(t, func() {
		printFormattedApplicationCharge(&OneTimeCharge{Price: "4.55", Currency: "USD"})
	})

	if got := priceLine(t, out); got != "4.55 USD" {
		t.Errorf("one-time charge price = %q, want %q", got, "4.55 USD")
	}

	out = captureStdout(t, func() {
		printFormattedRecurringCharge(&RecurringCharge{Price: "10.00", Currency: "EUR", Interval: "ANNUAL"})
	})

	if got := priceLine(t, out); got != "10.00 EUR" {
		t.Errorf("recurring charge price = %q, want %q", got, "10.00 EUR")
	}
}

// Charges from shops whose currency the API didn't return keep printing
// just the amount.
func TestPrintFormattedChargeWithoutCurrency(t *testing.T) {
	out := captureStdout(t, func() {
		printFormattedApplicationCharge(&OneTimeCharge{Price: "4.55"})
	})

	if got := priceLine(t, out); got != "4.55" {
		t.Errorf("one-time charge price = %q, want %q", got, "4.55")
	}
}

// JSONL output carries the currency too, alongside the amount.
func TestChargeJSONLIncludesCurrency(t *testing.T) {
	tests := []struct {
		name    string
		charge  interface{}
		want    string
		notWant string
	}{
		{
			name:   "one-time",
			charge: OneTimeCharge{Price: "4.55", Currency: "USD"},
			want:   `"price":"4.55","currency":"USD"`,
		},
		{
			name:   "recurring",
			charge: RecurringCharge{Price: "10.00", Currency: "EUR"},
			want:   `"currency":"EUR"`,
		},
		{
			name:    "no currency",
			charge:  OneTimeCharge{Price: "4.55"},
			notWant: `"currency"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			line, err := json.Marshal(tt.charge)
			if err != nil {
				t.Fatalf("cannot marshal charge: %v", err)
			}

			if len(tt.want) > 0 && !strings.Contains(string(line), tt.want) {
				t.Errorf("JSONL %s missing %q", line, tt.want)
			}
			if len(tt.notWant) > 0 && strings.Contains(string(line), tt.notWant) {
				t.Errorf("JSONL %s should not contain %q", line, tt.notWant)
			}
		})
	}
}
