package themes

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"


	"github.com/urfave/cli/v2"
	shopify "github.com/bold-commerce/go-shopify/v3"

	"github.com/ScreenStaring/shopify-dev-tools/cmd"
)

var Cmd cli.Command
var srcURL = regexp.MustCompile(`(?i)\A(?:https?:)//[-a-z\d]+`)

const themePathSeperator = "/"

func isDir(path string) bool {
	stat, err := os.Stat(path)
	return err == nil && stat.IsDir()
}

func uploadFile(client *shopify.Client, themeID int64, source, destination string) error {
	asset := shopify.Asset{ThemeID: themeID, Key: destination}

	// Src does not work:
	// https://github.com/bold-commerce/go-shopify/issues/195
	if srcURL.MatchString(source) {
		asset.Src = source
		asset.Key = destination
	} else {
		bytes, err := os.ReadFile(source)
		if err != nil {
			return fmt.Errorf("Failed to read file '%s': %s", source, err)
		}

		// SVG issues? https://github.com/golang/go/issues/47492
		contentType := http.DetectContentType(bytes)
		if strings.HasPrefix(contentType, "image") || strings.HasPrefix(contentType, "video") || contentType == "application/octet-stream" {
			// Attachment does not work:
			// https://github.com/bold-commerce/go-shopify/issues/195
			asset.Attachment = base64.StdEncoding.EncodeToString(bytes)
		} else {
			// Bytes to String. Not sure it's okay. But for images we need Base64
			asset.Value = string(bytes)
		}

		if strings.Index(destination, ".") == -1 {
			if(destination[len(destination) - 1] != themePathSeperator[0]) {
				destination = destination + themePathSeperator
			}

			path := strings.Split(source, string(os.PathSeparator))
			asset.Key = destination + path[len(path) - 1]
		}
	}

	fmt.Printf("Uploading '%s' to '%s'\n", source, asset.Key)

	_, err := client.Asset.Update(themeID, asset)
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

	files, err := directory.Readdir(0)
	if err != nil {
		return fmt.Errorf("Failed to read directory '%s': %s", directory, err)
	}

	defer directory.Close()

	for _, file := range(files) {
		if !file.IsDir() {
			path := strings.Join([]string{source, file.Name()}, string(os.PathSeparator))

			err = uploadFile(client, themeID, path, destination)
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
