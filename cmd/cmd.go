package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"
	shopify "github.com/bold-commerce/go-shopify/v3"
)

var Flags []cli.Flag
var accessTokenCommand = regexp.MustCompile(`\A\s*<\s*(.+)\z`)

func NewShopifyClient(c *cli.Context) *shopify.Client {
	app := shopify.App{
		ApiKey: c.String("api-key"),
		Password: c.String("api-password"),
	}

	//logger := &shopify.LeveledLogger{Level: shopify.LevelDebug}
	//return shopify.NewClient(app, c.String("shop"), c.String("access-token"), shopify.WithLogger(logger))

	shop := c.String("shop")
	return shopify.NewClient(app, shop, lookupAccessToken(shop, c.String("access-token")))
}

func ParseIntAt(c *cli.Context, pos int) (int64, error) {
	return strconv.ParseInt(c.Args().Get(pos), 10, 64)
}

func lookupAccessToken(shop, token string) string {
	match := accessTokenCommand.FindStringSubmatch(token)
	if len(match) == 0 {
		return token
	}

	out, err := exec.Command(match[1], shop).Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "access token command failed: %s\n", err)
		os.Exit(2)
	}

	return strings.TrimSuffix(string(out), "\n")
}

func init() {
	Flags = []cli.Flag{
		altsrc.NewStringFlag(
			&cli.StringFlag{
				Name:    "shop",
				Usage:   "Shopify domain or shop name to perform command against",
				Required: true,
				EnvVars: []string{"SHOPIFY_SHOP"},
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
			EnvVars: []string{"SHOPIFY_ACCESS_TOKEN", "SHOPIFY_API_TOKEN",},
		},
		&cli.StringFlag{
			Name:    "api-key",
			Usage:   "Shopify API key to for shop",
			EnvVars: []string{"SHOPIFY_API_KEY"},
		},
	}
}
