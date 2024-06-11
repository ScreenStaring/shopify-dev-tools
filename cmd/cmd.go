package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	shopify "github.com/bold-commerce/go-shopify/v3"
	"github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"
)

var Flags []cli.Flag
var accessTokenCommand = regexp.MustCompile(`\A\s*<\s*(.+)\z`)

func NewShopifyClient(c *cli.Context) *shopify.Client {
	var logging shopify.Option

	app := shopify.App{
		ApiKey:   c.String("api-key"),
		Password: c.String("api-password"),
	}

	shop := c.String("shop")

	if c.Bool("verbose") {
		logging = shopify.WithLogger(&shopify.LeveledLogger{Level: shopify.LevelDebug})
		return shopify.NewClient(app, shop, LookupAccessToken(shop, c.String("access-token")), logging)
	}

	return shopify.NewClient(app, shop, LookupAccessToken(shop, c.String("access-token")))
}

func ParseIntAt(c *cli.Context, pos int) (int64, error) {
	return strconv.ParseInt(c.Args().Get(pos), 10, 64)
}

func PrintSeparator() {
	fmt.Printf("%s\n", strings.Repeat("-", 20))
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
