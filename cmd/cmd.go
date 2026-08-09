package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/cheynewallace/tabby"
	"github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"

	"github.com/ScreenStaring/shopify-dev-tools/gql"
)

var Flags []cli.Flag
var accessTokenCommand = regexp.MustCompile(`\A\s*<\s*(.+)\z`)

func NewGraphQLClient(c *cli.Context) *gql.Client {
	shop := c.String("shop")
	token := LookupAccessToken(shop, c.String("access-token"))
	return gql.NewClient(shop, token, map[string]interface{}{"version": c.String("api-version")})
}

func ParseIntAt(c *cli.Context, pos int) (int64, error) {
	return strconv.ParseInt(c.Args().Get(pos), 10, 64)
}

func PrintSeparator() {
	fmt.Printf("%s\n", strings.Repeat("-", 20))
}

// MetafieldPrintable exists because callers source metafields from
// different gql-only result types that share no common shape. This is the
// shape each converts into so print logic lives once.
// Fields are interface{} so callers can pass through native types (e.g. int64
// ID, *time.Time timestamps) as-is; a nil ID omits the "Id" line entirely.
type MetafieldPrintable struct {
	ID          interface{}
	Gid         interface{}
	Namespace   interface{}
	Key         interface{}
	Description interface{}
	Value       interface{}
	Type        interface{}
	CreatedAt   interface{}
	UpdatedAt   interface{}
}

func PrintMetafields(items []MetafieldPrintable) {
	t := tabby.New()
	for _, mf := range items {
		if mf.ID != nil {
			t.AddLine("Id", mf.ID)
		}
		t.AddLine("Gid", mf.Gid)
		t.AddLine("Namespace", mf.Namespace)
		t.AddLine("Key", mf.Key)
		t.AddLine("Description", mf.Description)
		t.AddLine("Value", mf.Value)
		t.AddLine("Type", mf.Type)
		t.AddLine("Created", mf.CreatedAt)
		t.AddLine("Updated", mf.UpdatedAt)
		t.Print()
		PrintSeparator()
	}
}

func LookupAccessToken(shop, token string) string {
	match := accessTokenCommand.FindStringSubmatch(token)
	if len(match) == 0 {
		return token
	}

	out, err := exec.Command(match[1], shop).Output()
	// FIXME: return an error. Exit should be done in caller
	if err != nil {
		fmt.Fprintf(os.Stderr, "access token command failed: %s\n", err)
		os.Exit(2)
	}

	return strings.TrimSuffix(string(out), "\n")
}

func init() {
	Flags = []cli.Flag{
		&cli.BoolFlag{
			Name:  "verbose",
			Usage: "Output Shopify API request/response",
		},
		altsrc.NewStringFlag(
			&cli.StringFlag{
				Name:     "shop",
				Usage:    "Shopify domain or shop name to perform command against",
				Required: true,
				EnvVars:  []string{"SHOPIFY_SHOP"},
			},
		),
		&cli.StringFlag{
			Name:    "api-password",
			Usage:   "Shopify API password",
			EnvVars: []string{"SHOPIFY_API_PASSWORD"},
		},
		&cli.StringFlag{
			Name:    "access-token",
			Usage:   "Shopify access token for shop",
			EnvVars: []string{"SHOPIFY_ACCESS_TOKEN", "SHOPIFY_API_TOKEN"},
		},
		&cli.StringFlag{
			Name:    "api-key",
			Usage:   "Shopify API key to for shop",
			EnvVars: []string{"SHOPIFY_API_KEY"},
		},
	}
}
