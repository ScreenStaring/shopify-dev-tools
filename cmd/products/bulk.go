package products

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"regexp"

	"github.com/cheynewallace/tabby"
	"github.com/urfave/cli/v2"

	"github.com/ScreenStaring/shopify-dev-tools/cmd"
	"github.com/ScreenStaring/shopify-dev-tools/cmd/products/gql"
)

type importError struct {
	row     int
	message string
}

type bulkResultLine struct {
	Data struct {
		ProductSet struct {
			UserErrors []struct {
				Message string   `json:"message"`
				Field   []string `json:"field"`
			} `json:"userErrors"`
		} `json:"productSet"`
	} `json:"data"`
	LineNumber int `json:"__lineNumber"`
}

func setProductIdentifiers(products []importProductInput, identifyBy string) {
	if identifyBy == "" {
		return
	}

	numericID := regexp.MustCompile(`^\d+$`)
	for i := range products {
		p := &products[i]
		if p.Identifier != nil {
			continue
		}
		if identifyBy == "id" {
			id := p.Input.ID
			if id != "" {
				if numericID.MatchString(id) {
					id = "gid://shopify/Product/" + id
				}
				p.Identifier = &productSetIdentifier{ID: id}
			}
		} else if identifyBy == "handle" {
			if p.Input.Handle != "" {
				p.Identifier = &productSetIdentifier{Handle: p.Input.Handle}
			}
		}
	}
}

func buildJSONL(products []importProductInput) ([]byte, error) {
	var buf bytes.Buffer

	for _, p := range products {
		line, err := json.Marshal(p)
		if err != nil {
			return nil, fmt.Errorf("Cannot marshal product to JSONL: %s", err)
		}
		buf.Write(line)
		buf.WriteByte('\n')
	}

	return buf.Bytes(), nil
}

func uploadFile(target *gql.StagedTarget, data []byte) error {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	for _, param := range target.Parameters {
		if err := writer.WriteField(param.Name, param.Value); err != nil {
			return fmt.Errorf("Cannot write multipart field %s: %s", param.Name, err)
		}
	}

	part, err := writer.CreateFormFile("file", "bulk_import.jsonl")
	if err != nil {
		return fmt.Errorf("Cannot create multipart file field: %s", err)
	}

	if _, err := part.Write(data); err != nil {
		return fmt.Errorf("Cannot write file data: %s", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("Cannot close multipart writer: %s", err)
	}

	req, err := http.NewRequest("POST", target.URL, &buf)
	if err != nil {
		return fmt.Errorf("Cannot create upload request: %s", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("Upload request failed: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("Upload failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func importProducts(c *cli.Context) error {
	if c.NArg() < 1 {
		return fmt.Errorf("CSV file path required")
	}

	csvFile := c.Args().First()
	shop := c.String("shop")
	token := cmd.LookupAccessToken(shop, c.String("access-token"))
	options := map[string]interface{}{"version": c.String("api-version")}

	locations, err := gql.FetchLocations(shop, token, options)
	if err != nil {
		return err
	}

	fmt.Printf("Parsing %s...\n", csvFile)

	products, err := parseCSV(csvFile, locations)
	if err != nil {
		return err
	}

	if len(products) == 0 {
		return fmt.Errorf("No products found in CSV. Does the identifier column exist?")
	}

	setProductIdentifiers(products, c.String("identify-by"))

	fmt.Printf("Found %d products\n", len(products))

	jsonlData, err := buildJSONL(products)
	if err != nil {
		return err
	}

	fmt.Println("Creating staged upload...")

	target, err := gql.StagedUpload(shop, token, len(jsonlData), options)
	if err != nil {
		return err
	}

	fmt.Println("Uploading JSONL file...")

	if err := uploadFile(target, jsonlData); err != nil {
		return err
	}

	fmt.Println("Starting bulk operation...")

	var stagedUploadPath string
	for _, param := range target.Parameters {
		if param.Name == "key" {
			stagedUploadPath = param.Value
			break
		}
	}

	if stagedUploadPath == "" {
		stagedUploadPath = target.ResourceURL
	}

	operationID, status, err := gql.StartBulkMutation(shop, token, stagedUploadPath, options)
	if err != nil {
		return err
	}

	fmt.Printf("Bulk operation started\n")
	fmt.Printf("  Operation ID: %s\n", operationID)
	fmt.Printf("  Status: %s\n", status)
	fmt.Printf("\nCheck status with: sdt products bulk status %s\n", operationID)

	return nil
}

func downloadBulkResults(url string) ([]importError, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("Cannot download result file: %s", err)
	}
	defer resp.Body.Close()

	var errors []importError
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		var line bulkResultLine
		if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
			continue
		}
		for _, ue := range line.Data.ProductSet.UserErrors {
			errors = append(errors, importError{
				row:     line.LineNumber + 1,
				message: ue.Message,
			})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("Error reading result file: %s", err)
	}

	return errors, nil
}

func importStatus(c *cli.Context) error {
	if c.NArg() < 1 {
		return fmt.Errorf("Operation ID required")
	}

	operationID := c.Args().First()
	if matched, _ := regexp.MatchString(`^\d+$`, operationID); matched {
		operationID = "gid://shopify/BulkOperation/" + operationID
	}

	shop := c.String("shop")
	token := cmd.LookupAccessToken(shop, c.String("access-token"))
	options := map[string]interface{}{"version": c.String("api-version")}

	result, err := gql.FetchBulkOperationStatus(shop, token, operationID, options)
	if err != nil {
		return err
	}

	t := tabby.New()
	t.AddLine("ID", result.ID)
	t.AddLine("Status", result.Status)
	if result.ErrorCode != "" {
		t.AddLine("Error Code", result.ErrorCode)
	}
	t.AddLine("Object Count", result.ObjectCount)
	t.AddLine("Root Object Count", result.RootObjectCount)
	t.AddLine("Created At", result.CreatedAt)
	if result.CompletedAt != "" {
		t.AddLine("Completed At", result.CompletedAt)
	}
	if result.URL != "" {
		t.AddLine("Result URL", result.URL)
	}
	t.Print()

	if result.Status == "COMPLETED" && result.URL != "" {
		errors, err := downloadBulkResults(result.URL)
		if err != nil {
			return err
		}

		if len(errors) > 0 {
			fmt.Println("Errors")
			et := tabby.New()
			et.AddHeader("Row", "Message")
			for _, e := range errors {
				et.AddLine(e.row, e.message)
			}
			et.Print()
		}
	}

	return nil
}

func cancelBulkOperation(c *cli.Context) error {
	if c.NArg() < 1 {
		return fmt.Errorf("Operation ID required")
	}

	operationID := c.Args().First()
	if matched, _ := regexp.MatchString(`^\d+$`, operationID); matched {
		operationID = "gid://shopify/BulkOperation/" + operationID
	}

	shop := c.String("shop")
	token := cmd.LookupAccessToken(shop, c.String("access-token"))
	options := map[string]interface{}{"version": c.String("api-version")}

	id, status, err := gql.CancelBulkOperation(shop, token, operationID, options)
	if err != nil {
		return err
	}

	fmt.Printf("Operation %s: %s\n", id, status)

	return nil
}
