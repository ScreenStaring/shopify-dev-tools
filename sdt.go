package main

import (
	"fmt"
	"os"

	"github.com/ScreenStaring/shopify-dev-tools/cmd/admin"
	"github.com/ScreenStaring/shopify-dev-tools/cmd/gql"
	"github.com/ScreenStaring/shopify-dev-tools/cmd/metafields"
	"github.com/ScreenStaring/shopify-dev-tools/cmd/orders"
	"github.com/ScreenStaring/shopify-dev-tools/cmd/products"
	"github.com/ScreenStaring/shopify-dev-tools/cmd/scripttags"
	"github.com/ScreenStaring/shopify-dev-tools/cmd/shop"
	"github.com/ScreenStaring/shopify-dev-tools/cmd/themes"
	"github.com/ScreenStaring/shopify-dev-tools/cmd/webhooks"
	"github.com/urfave/cli/v2"
)

const version = "0.0.4"

func main() {
	app := &cli.App{
		Name:                   "sdt",
		Usage:                  "Shopify Development Tools",
		Version:                version,
		UseShortOptionHandling: true,
		Commands: []*cli.Command{
			&admin.Cmd,
			&metafields.Cmd,
			&orders.Cmd,
			&products.Cmd,
			&gql.Cmd,
			&shop.Cmd,
			&scripttags.Cmd,
			&themes.Cmd,
			&webhooks.Cmd,
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
	}
}
