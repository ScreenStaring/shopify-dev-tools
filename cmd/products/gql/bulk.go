package gql

import (
	"encoding/json"
	"fmt"

	gqlclient "github.com/ScreenStaring/shopify-dev-tools/gql"
)

const stagedUploadsCreateMutation = `
mutation stagedUploadsCreate($input: [StagedUploadInput!]!) {
  stagedUploadsCreate(input: $input) {
    stagedTargets {
      url
      resourceUrl
      parameters {
        name
        value
      }
    }
    userErrors {
      field
      message
    }
  }
}
`

const productSetMutation = `
mutation productSet($input: ProductSetInput!, $identifier: ProductSetIdentifiers) {
  productSet(input: $input, identifier: $identifier) {
    product {
      id
    }
    userErrors {
      field
      message
    }
  }
}
`

const bulkOperationRunMutationQuery = `
mutation bulkOperationRunMutation($mutation: String!, $stagedUploadPath: String!) {
  bulkOperationRunMutation(mutation: $mutation, stagedUploadPath: $stagedUploadPath) {
    bulkOperation {
      id
      status
      url
    }
    userErrors {
      field
      message
    }
  }
}
`

const bulkOperationStatusQuery = `
query($id: ID!) {
  node(id: $id) {
    ... on BulkOperation {
      id
      status
      errorCode
      objectCount
      rootObjectCount
      url
      createdAt
      completedAt
    }
  }
}
`

const bulkOperationCancelMutation = `
mutation bulkOperationCancel($id: ID!) {
  bulkOperationCancel(id: $id) {
    bulkOperation {
      id
      status
    }
    userErrors {
      field
      message
    }
  }
}
`

type StagedUploadParameter struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type StagedTarget struct {
	URL         string                  `json:"url"`
	ResourceURL string                  `json:"resourceUrl"`
	Parameters  []StagedUploadParameter `json:"parameters"`
}

type stagedUploadsResponse struct {
	Data struct {
		StagedUploadsCreate struct {
			StagedTargets []StagedTarget `json:"stagedTargets"`
			UserErrors    []struct {
				Field   []string `json:"field"`
				Message string   `json:"message"`
			} `json:"userErrors"`
		} `json:"stagedUploadsCreate"`
	} `json:"data"`
}

type bulkOperationRunResponse struct {
	Data struct {
		BulkOperationRunMutation struct {
			BulkOperation struct {
				ID     string `json:"id"`
				Status string `json:"status"`
				URL    string `json:"url"`
			} `json:"bulkOperation"`
			UserErrors []struct {
				Field   []string `json:"field"`
				Message string   `json:"message"`
			} `json:"userErrors"`
		} `json:"bulkOperationRunMutation"`
	} `json:"data"`
}

type BulkOperationResult struct {
	ID              string `json:"id"`
	Status          string `json:"status"`
	ErrorCode       string `json:"errorCode"`
	ObjectCount     int    `json:"objectCount,string"`
	RootObjectCount int    `json:"rootObjectCount,string"`
	URL             string `json:"url"`
	CreatedAt       string `json:"createdAt"`
	CompletedAt     string `json:"completedAt"`
}

type bulkOperationStatusResponse struct {
	Data struct {
		Node BulkOperationResult `json:"node"`
	} `json:"data"`
}

func StagedUpload(shop, token string, fileSize int, options map[string]interface{}) (*StagedTarget, error) {
	client := gqlclient.NewClient(shop, token, options)

	input := []map[string]interface{}{
		{
			"resource":   "BULK_MUTATION_VARIABLES",
			"filename":   "bulk_import.jsonl",
			"mimeType":   "text/jsonl",
			"httpMethod": "POST",
			"fileSize":   fmt.Sprintf("%d", fileSize),
		},
	}

	data, err := client.Execute(stagedUploadsCreateMutation, map[string]interface{}{
		"input": input,
	})
	if err != nil {
		return nil, fmt.Errorf("Cannot create staged upload: %s", err)
	}

	b, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("Cannot re-encode staged upload response: %s", err)
	}

	var response stagedUploadsResponse
	if err := json.Unmarshal(b, &response); err != nil {
		return nil, fmt.Errorf("Cannot parse staged upload response: %s", err)
	}

	if len(response.Data.StagedUploadsCreate.UserErrors) > 0 {
		return nil, fmt.Errorf("Staged upload error: %s", response.Data.StagedUploadsCreate.UserErrors[0].Message)
	}

	if len(response.Data.StagedUploadsCreate.StagedTargets) == 0 {
		return nil, fmt.Errorf("No staged upload targets returned")
	}

	return &response.Data.StagedUploadsCreate.StagedTargets[0], nil
}

func StartBulkMutation(shop, token, stagedUploadPath string, options map[string]interface{}) (string, string, error) {
	client := gqlclient.NewClient(shop, token, options)

	data, err := client.Execute(bulkOperationRunMutationQuery, map[string]interface{}{
		"mutation":         productSetMutation,
		"stagedUploadPath": stagedUploadPath,
	})
	if err != nil {
		return "", "", fmt.Errorf("Cannot start bulk mutation: %s", err)
	}

	b, err := json.Marshal(data)
	if err != nil {
		return "", "", fmt.Errorf("Cannot re-encode bulk mutation response: %s", err)
	}

	var response bulkOperationRunResponse
	if err := json.Unmarshal(b, &response); err != nil {
		return "", "", fmt.Errorf("Cannot parse bulk mutation response: %s", err)
	}

	if len(response.Data.BulkOperationRunMutation.UserErrors) > 0 {
		return "", "", fmt.Errorf("Bulk mutation error: %s", response.Data.BulkOperationRunMutation.UserErrors[0].Message)
	}

	op := response.Data.BulkOperationRunMutation.BulkOperation
	return op.ID, op.Status, nil
}

func FetchBulkOperationStatus(shop, token, operationID string, options map[string]interface{}) (*BulkOperationResult, error) {
	client := gqlclient.NewClient(shop, token, options)

	data, err := client.Execute(bulkOperationStatusQuery, map[string]interface{}{
		"id": operationID,
	})
	if err != nil {
		return nil, fmt.Errorf("Cannot fetch bulk operation status: %s", err)
	}

	b, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("Cannot re-encode bulk operation status response: %s", err)
	}

	var response bulkOperationStatusResponse
	if err := json.Unmarshal(b, &response); err != nil {
		return nil, fmt.Errorf("Cannot parse bulk operation status response: %s", err)
	}

	return &response.Data.Node, nil
}

type bulkOperationCancelResponse struct {
	Data struct {
		BulkOperationCancel struct {
			BulkOperation struct {
				ID     string `json:"id"`
				Status string `json:"status"`
			} `json:"bulkOperation"`
			UserErrors []struct {
				Field   []string `json:"field"`
				Message string   `json:"message"`
			} `json:"userErrors"`
		} `json:"bulkOperationCancel"`
	} `json:"data"`
}

func CancelBulkOperation(shop, token, operationID string, options map[string]interface{}) (string, string, error) {
	client := gqlclient.NewClient(shop, token, options)

	data, err := client.Execute(bulkOperationCancelMutation, map[string]interface{}{
		"id": operationID,
	})
	if err != nil {
		return "", "", fmt.Errorf("Cannot cancel bulk operation: %s", err)
	}

	b, err := json.Marshal(data)
	if err != nil {
		return "", "", fmt.Errorf("Cannot re-encode bulk operation cancel response: %s", err)
	}

	var response bulkOperationCancelResponse
	if err := json.Unmarshal(b, &response); err != nil {
		return "", "", fmt.Errorf("Cannot parse bulk operation cancel response: %s", err)
	}

	if len(response.Data.BulkOperationCancel.UserErrors) > 0 {
		return "", "", fmt.Errorf("Bulk operation cancel error: %s", response.Data.BulkOperationCancel.UserErrors[0].Message)
	}

	op := response.Data.BulkOperationCancel.BulkOperation
	return op.ID, op.Status, nil
}
