package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
	"github.com/ScreenStaring/shopify-dev-tools/cmd/admin"
	"github.com/ScreenStaring/shopify-dev-tools/cmd/metafields"
	"github.com/ScreenStaring/shopify-dev-tools/cmd/shop"
	"github.com/ScreenStaring/shopify-dev-tools/cmd/webhooks"
)

const version = "0.0.1"

func main() {
	app := &cli.App{
		Name: "sdt",
		Usage: "Shopify Development Tools",
		Version: version,
		UseShortOptionHandling: true,
		Commands: []*cli.Command{
			&admin.Cmd,
			&metafields.Cmd,
			&shop.Cmd,
			&webhooks.Cmd,
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
	}
}
