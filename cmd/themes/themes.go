package themes

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/cheynewallace/tabby"
	"github.com/urfave/cli/v2"

	"github.com/ScreenStaring/shopify-dev-tools/cmd"
	"github.com/ScreenStaring/shopify-dev-tools/gql"
)

var Cmd cli.Command

func isDir(path string) bool {
	stat, err := os.Stat(path)
	return err == nil && stat.IsDir()
}

func destinationPath(source, destination string) string {
	const themePathSeperator = "/"

	if strings.Index(destination, ".") == -1 {
		if destination[len(destination)-1] != themePathSeperator[0] {
			destination = destination + themePathSeperator
		}

		path := strings.Split(source, string(os.PathSeparator))
		destination = destination + path[len(path)-1]
	}

	return destination
}

func uploadFile(client *gql.Client, themeID int64, source, destination string) error {
	destination = destinationPath(source, destination)

	fmt.Printf("Uploading '%s' to '%s'\n", source, destination)

	value, err := os.ReadFile(source)
	if err != nil {
		return fmt.Errorf("Failed to read file '%s': %s", source, err)
	}

	contentType := http.DetectContentType(value)
	bodyType := "TEXT"
	bodyValue := string(value)
	if strings.HasPrefix(contentType, "image") || strings.HasPrefix(contentType, "video") || contentType == "application/octet-stream" {
		bodyType = "BASE64"
		bodyValue = base64.StdEncoding.EncodeToString(value)
	}

	if err := upsertThemeFiles(client, themeID, destination, bodyType, bodyValue); err != nil {
		return fmt.Errorf("Cannot upload asset '%s': %s", source, err)
	}

	return nil
}

func uploadDirectory(client *gql.Client, themeID int64, source, destination string) error {
	directory, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("Failed to open directory '%s': %s", source, err)
	}

	defer directory.Close()

	files, err := directory.Readdir(0)
	if err != nil {
		return fmt.Errorf("Failed to read directory '%s': %s", source, err)
	}

	for _, file := range files {
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

func listAction(c *cli.Context) error {
	themes, err := listThemes(cmd.NewGraphQLClient(c))
	if err != nil {
		return fmt.Errorf("Cannot list themes: %s", err)
	}

	t := tabby.New()
	t.AddHeader("ID", "Name", "Role", "Theme Store ID", "Created", "Updated")

	for _, theme := range themes {
		t.AddLine(theme.ID, theme.Name, theme.Role, theme.ThemeStoreID, theme.CreatedAt, theme.UpdatedAt)
	}

	t.Print()

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

	client := cmd.NewGraphQLClient(c)

	args := c.Args().Slice()
	sources := args[1 : len(args)-1]
	destination := args[len(args)-1]

	for _, source := range sources {
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
	apiVersionFlag := cmd.APIVersionFlag

	Cmd = cli.Command{
		Name:    "themes",
		Aliases: []string{"theme", "t"},
		Usage:   "Theme utilities",
		Subcommands: []*cli.Command{
			{
				Name:   "ls",
				Usage:  "List the shop's themes",
				Flags:  append(cmd.Flags, apiVersionFlag),
				Action: listAction,
			},
			{
				Name:      "cp",
				Aliases:   []string{"copy"},
				Usage:     "Copy files to a theme",
				ArgsUsage: "themeid source [...] destination",
				Flags:     append(cmd.Flags, apiVersionFlag),
				Action:    copyAction,
			},
		},
	}
}
