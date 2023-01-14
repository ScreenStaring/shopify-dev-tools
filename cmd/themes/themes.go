package themes

import (
	"fmt"
	"os"
	"strings"

	"github.com/urfave/cli/v2"
	shopify "github.com/bold-commerce/go-shopify/v3"

	"github.com/ScreenStaring/shopify-dev-tools/cmd"
)

var Cmd cli.Command

func isDir(path string) bool {
	stat, err := os.Stat(path)
	return err == nil && stat.IsDir()
}

func uploadFile(client *shopify.Client, themeID int64, source, destination string) error {
	const themePathSeperator = "/"

	if strings.Index(destination, ".") == -1 {
		if(destination[len(destination) - 1] != themePathSeperator[0]) {
			destination = destination + themePathSeperator
		}

		path := strings.Split(source, string(os.PathSeparator))
		destination = destination + path[len(path) - 1]
	}

	fmt.Printf("Uploading '%s' to '%s'\n", source, destination)

	value, err := os.ReadFile(source)
	if err != nil {
		return fmt.Errorf("Failed to read file '%s': %s", source, err)
	}

	asset := shopify.Asset{
		Key: destination,
		Value: string(value),
		ThemeID: themeID,
	}

	_, err = client.Asset.Update(themeID, asset)
	if err != nil {
		return fmt.Errorf("Cannot upload asset '%s': %s", source, err)
	}

	return nil
}

func uploadDirectory(client *shopify.Client, themeID int64, source, destination string) error {
	directory, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("Failed to open directory '%s': %s", directory, err)
	}

	defer directory.Close()

	files, err := directory.Readdir(0)
	if err != nil {
		return fmt.Errorf("Failed to read directory '%s': %s", directory, err)
	}

	for _, file := range(files) {
		if !file.IsDir() {
			path := []string{source, file.Name()}

			err = uploadFile(client, themeID, strings.Join(path, string(os.PathSeparator)), destination)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func copyAction(c *cli.Context) error {
	if c.NArg() < 1 {
		return fmt.Errorf("You must supply a theme id")
	}

	if c.NArg() < 3 {
		return fmt.Errorf("You must supply a source and destination")
	}


	themeID, err := cmd.ParseIntAt(c, 0)
	if err != nil {
		return fmt.Errorf("Theme id '%s' invalid: must be an int", c.Args().Get(0))
	}

	client := cmd.NewShopifyClient(c)

	args := c.Args().Slice()
	sources := args[1:len(args) - 1]
	destination := args[len(args) - 1]

	for _, source := range(sources) {
		if isDir(source) {
			err = uploadDirectory(client, themeID, source, destination)
		} else {
			err = uploadFile(client, themeID, source, destination)
		}

		if err != nil {
			return err
		}
	}

	return nil
}

func init() {
	Cmd = cli.Command{
		Name:  "themes",
		Aliases: []string{"theme", "t"},
		Usage:   "Theme utilities",
		Subcommands: []*cli.Command{
			{
				Name: "cp",
				Aliases: []string{"copy"},
				Usage:   "Copy files to a theme",
				ArgsUsage: "themeid source [...] destination",
				Flags: cmd.Flags,
				Action: copyAction,
			},
		},
	}
}
