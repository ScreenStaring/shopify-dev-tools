package events

import (
	"errors"
	"fmt"
	"strings"

	"github.com/cheynewallace/tabby"
	"github.com/urfave/cli/v2"

	"github.com/ScreenStaring/shopify-dev-tools/cmd"
)

var Cmd cli.Command

var supportedTypes = map[string]bool{
	"Product":        true,
	"ProductVariant": true,
}

func gidType(gid string) (string, error) {
	if !strings.HasPrefix(gid, "gid://shopify/") {
		return "", fmt.Errorf("invalid GID: %s", gid)
	}

	rest := strings.TrimPrefix(gid, "gid://shopify/")
	parts := strings.SplitN(rest, "/", 2)
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid GID: %s", gid)
	}

	return parts[0], nil
}


func listAction(c *cli.Context) error {
	if c.NArg() == 0 {
		return errors.New("GID required")
	}

	gid := c.Args().Get(0)
	resourceType, err := gidType(gid)
	if err != nil {
		return err
	}

	if !supportedTypes[resourceType] {
		return fmt.Errorf("unsupported resource type %q: supported types are Product and ProductVariant", resourceType)
	}

	shop := c.String("shop")
	token := cmd.LookupAccessToken(shop, c.String("access-token"))
	options := map[string]interface{}{"version": c.String("api-version")}

	events, err := fetchEvents(shop, token, gid, resourceType, options)
	if err != nil {
		return err
	}

	t := tabby.New()
	t.AddHeader("ID", "App", "Action", "Message", "Created")
	for _, event := range events {
		t.AddLine(event.ID, event.AppTitle, event.Action, event.Message, event.CreatedAt)
	}
	t.Print()

	return nil
}

func init() {
	Cmd = cli.Command{
		Name:      "events",
		Aliases:   []string{"e"},
		ArgsUsage: "GID",
		Usage:     "List events for a resource",
		Flags: append(cmd.Flags, &cli.StringFlag{
			Name:    "api-version",
			Aliases: []string{"a"},
			Usage:   "API version to use; default is a versionless call",
		}),
		Action: listAction,
	}
}
