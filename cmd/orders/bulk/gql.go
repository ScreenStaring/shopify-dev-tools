package bulk

import (
	"encoding/json"
	"fmt"

	gqlclient "github.com/ScreenStaring/shopify-dev-tools/gql"
)

const orderByIDQuery = `
query($id: ID!) {
  order(id: $id) {
    id
    name
    note
    email
    displayFulfillmentStatus
    customer {
      id
      email
    }
    lineItems(first: 250) {
      edges {
        node {
          id
          sku
          quantity
          variant {
            id
            barcode
          }
        }
      }
    }
    fulfillmentOrders(first: 10) {
      edges {
        node {
          id
          status
        }
      }
    }
    fulfillments(first: 10) {
      id
      status
    }
  }
}
`

const orderByNameQuery = `
query($query: String!) {
  orders(first: 1, query: $query) {
    edges {
      node {
        id
        name
        note
        email
        displayFulfillmentStatus
        customer {
          id
          email
        }
        lineItems(first: 250) {
          edges {
            node {
              id
              sku
              quantity
              variant {
                id
                barcode
              }
            }
          }
        }
        fulfillmentOrders(first: 10) {
          edges {
            node {
              id
              status
            }
          }
        }
        fulfillments(first: 10) {
          id
          status
        }
      }
    }
  }
}
`

const variantLookupQuery = `
query($query: String!) {
  productVariants(first: 1, query: $query) {
    edges {
      node {
        id
      }
    }
  }
}
`

const orderEditBeginMutation = `
mutation orderEditBegin($id: ID!) {
  orderEditBegin(id: $id) {
    calculatedOrder {
      id
      lineItems(first: 250) {
        edges {
          node {
            id
          }
        }
      }
    }
    userErrors {
      field
      message
    }
  }
}
`

const orderEditSetQuantityMutation = `
mutation orderEditSetQuantity($id: ID!, $lineItemId: ID!, $quantity: Int!) {
  orderEditSetQuantity(id: $id, lineItemId: $lineItemId, quantity: $quantity) {
    calculatedOrder {
      id
    }
    userErrors {
      field
      message
    }
  }
}
`

const orderEditAddVariantMutation = `
mutation orderEditAddVariant($id: ID!, $variantId: ID!, $quantity: Int!) {
  orderEditAddVariant(id: $id, variantId: $variantId, quantity: $quantity) {
    calculatedLineItem {
      id
    }
    userErrors {
      field
      message
    }
  }
}
`

const orderEditCommitMutation = `
mutation orderEditCommit($id: ID!) {
  orderEditCommit(id: $id) {
    order {
      id
    }
    userErrors {
      field
      message
    }
  }
}
`

const orderUpdateMutation = `
mutation orderUpdate($input: OrderInput!) {
  orderUpdate(input: $input) {
    order {
      id
    }
    userErrors {
      field
      message
    }
  }
}
`

const fulfillmentCreateV2Mutation = `
mutation fulfillmentCreateV2($fulfillment: FulfillmentV2Input!) {
  fulfillmentCreateV2(fulfillment: $fulfillment) {
    fulfillment {
      id
    }
    userErrors {
      field
      message
    }
  }
}
`

const fulfillmentCancelMutation = `
mutation fulfillmentCancel($id: ID!) {
  fulfillmentCancel(id: $id) {
    fulfillment {
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

const customerByEmailQuery = `
query($query: String!) {
  customers(first: 2, query: $query) {
    edges {
      node {
        id
        email
      }
    }
  }
}
`

const orderCustomerSetMutation = `
mutation orderCustomerSet($orderId: ID!, $customerId: ID!) {
  orderCustomerSet(orderId: $orderId, customerId: $customerId) {
    order {
      id
    }
    userErrors {
      field
      message
    }
  }
}
`

// Response types

type VariantRef struct {
	ID      string `json:"id"`
	Barcode string `json:"barcode"`
}

type LineItem struct {
	ID       string      `json:"id"`
	SKU      string      `json:"sku"`
	Quantity int         `json:"quantity"`
	Variant  *VariantRef `json:"variant"`
}

type Customer struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

type FulfillmentOrder struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type Fulfillment struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type Order struct {
	ID                       string `json:"id"`
	Name                     string `json:"name"`
	Note                     string `json:"note"`
	Email                    string `json:"email"`
	DisplayFulfillmentStatus string    `json:"displayFulfillmentStatus"`
	Customer                *Customer `json:"customer"`
	LineItems               struct {
		Edges []struct {
			Node LineItem `json:"node"`
		} `json:"edges"`
	} `json:"lineItems"`
	FulfillmentOrders struct {
		Edges []struct {
			Node FulfillmentOrder `json:"node"`
		} `json:"edges"`
	} `json:"fulfillmentOrders"`
	Fulfillments []Fulfillment `json:"fulfillments"`
}

type orderByIDResponse struct {
	Data struct {
		Order Order `json:"order"`
	} `json:"data"`
}

type orderByNameResponse struct {
	Data struct {
		Orders struct {
			Edges []struct {
				Node Order `json:"node"`
			} `json:"edges"`
		} `json:"orders"`
	} `json:"data"`
}

type variantLookupResponse struct {
	Data struct {
		ProductVariants struct {
			Edges []struct {
				Node struct {
					ID string `json:"id"`
				} `json:"node"`
			} `json:"edges"`
		} `json:"productVariants"`
	} `json:"data"`
}

type userError struct {
	Field   []string `json:"field"`
	Message string   `json:"message"`
}

type calculatedLineItem struct {
	ID string `json:"id"`
}

type orderEditBeginResponse struct {
	Data struct {
		OrderEditBegin struct {
			CalculatedOrder struct {
				ID        string `json:"id"`
				LineItems struct {
					Edges []struct {
						Node calculatedLineItem `json:"node"`
					} `json:"edges"`
				} `json:"lineItems"`
			} `json:"calculatedOrder"`
			UserErrors []userError `json:"userErrors"`
		} `json:"orderEditBegin"`
	} `json:"data"`
}

type orderEditSetQuantityResponse struct {
	Data struct {
		OrderEditSetQuantity struct {
			UserErrors []userError `json:"userErrors"`
		} `json:"orderEditSetQuantity"`
	} `json:"data"`
}

type orderEditAddVariantResponse struct {
	Data struct {
		OrderEditAddVariant struct {
			UserErrors []userError `json:"userErrors"`
		} `json:"orderEditAddVariant"`
	} `json:"data"`
}

type orderEditCommitResponse struct {
	Data struct {
		OrderEditCommit struct {
			UserErrors []userError `json:"userErrors"`
		} `json:"orderEditCommit"`
	} `json:"data"`
}

type orderUpdateResponse struct {
	Data struct {
		OrderUpdate struct {
			UserErrors []userError `json:"userErrors"`
		} `json:"orderUpdate"`
	} `json:"data"`
}

type fulfillmentCreateResponse struct {
	Data struct {
		FulfillmentCreateV2 struct {
			UserErrors []userError `json:"userErrors"`
		} `json:"fulfillmentCreateV2"`
	} `json:"data"`
}

type fulfillmentCancelResponse struct {
	Data struct {
		FulfillmentCancel struct {
			UserErrors []userError `json:"userErrors"`
		} `json:"fulfillmentCancel"`
	} `json:"data"`
}

type customerLookupResponse struct {
	Data struct {
		Customers struct {
			Edges []struct {
				Node Customer `json:"node"`
			} `json:"edges"`
		} `json:"customers"`
	} `json:"data"`
}

type orderCustomerSetResponse struct {
	Data struct {
		OrderCustomerSet struct {
			UserErrors []userError `json:"userErrors"`
		} `json:"orderCustomerSet"`
	} `json:"data"`
}

// Helper

func unmarshal(data interface{}, target interface{}) error {
	b, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("cannot re-encode response: %s", err)
	}

	if err := json.Unmarshal(b, target); err != nil {
		return fmt.Errorf("cannot parse response: %s", err)
	}

	return nil
}

// API functions

func FetchOrder(client *gqlclient.Client, orderID, orderName string) (*Order, error) {
	if orderID != "" {
		gid := fmt.Sprintf("gid://shopify/Order/%s", orderID)
		vars := map[string]interface{}{"id": gid}

		data, err := client.Execute(orderByIDQuery, vars)
		if err != nil {
			return nil, fmt.Errorf("cannot fetch order %s: %s", orderID, err)
		}

		var resp orderByIDResponse
		if err := unmarshal(data, &resp); err != nil {
			return nil, err
		}

		if resp.Data.Order.ID == "" {
			return nil, fmt.Errorf("order %s not found", orderID)
		}

		return &resp.Data.Order, nil
	}

	vars := map[string]interface{}{"query": fmt.Sprintf("name:%s", orderName)}

	data, err := client.Execute(orderByNameQuery, vars)
	if err != nil {
		return nil, fmt.Errorf("cannot fetch order %s: %s", orderName, err)
	}

	var resp orderByNameResponse
	if err := unmarshal(data, &resp); err != nil {
		return nil, err
	}

	if len(resp.Data.Orders.Edges) == 0 {
		return nil, fmt.Errorf("order %s not found", orderName)
	}

	return &resp.Data.Orders.Edges[0].Node, nil
}

func FindVariantBySKU(client *gqlclient.Client, sku string) (string, error) {
	vars := map[string]interface{}{"query": fmt.Sprintf("sku:%s", sku)}

	data, err := client.Execute(variantLookupQuery, vars)
	if err != nil {
		return "", fmt.Errorf("cannot lookup variant by SKU %s: %s", sku, err)
	}

	var resp variantLookupResponse
	if err := unmarshal(data, &resp); err != nil {
		return "", err
	}

	if len(resp.Data.ProductVariants.Edges) == 0 {
		return "", fmt.Errorf("no variant found with SKU %s", sku)
	}

	return resp.Data.ProductVariants.Edges[0].Node.ID, nil
}

func FindVariantByBarcode(client *gqlclient.Client, barcode string) (string, error) {
	vars := map[string]interface{}{"query": fmt.Sprintf("barcode:%s", barcode)}

	data, err := client.Execute(variantLookupQuery, vars)
	if err != nil {
		return "", fmt.Errorf("cannot lookup variant by barcode %s: %s", barcode, err)
	}

	var resp variantLookupResponse
	if err := unmarshal(data, &resp); err != nil {
		return "", err
	}

	if len(resp.Data.ProductVariants.Edges) == 0 {
		return "", fmt.Errorf("no variant found with barcode %s", barcode)
	}

	return resp.Data.ProductVariants.Edges[0].Node.ID, nil
}

func BeginOrderEdit(client *gqlclient.Client, orderGID string) (string, []string, error) {
	vars := map[string]interface{}{"id": orderGID}

	data, err := client.Execute(orderEditBeginMutation, vars)
	if err != nil {
		return "", nil, fmt.Errorf("cannot begin order edit: %s", err)
	}

	var resp orderEditBeginResponse
	if err := unmarshal(data, &resp); err != nil {
		return "", nil, err
	}

	if len(resp.Data.OrderEditBegin.UserErrors) > 0 {
		return "", nil, fmt.Errorf("cannot begin order edit: %s", resp.Data.OrderEditBegin.UserErrors[0].Message)
	}

	calcOrder := resp.Data.OrderEditBegin.CalculatedOrder
	var calcLineItemIDs []string
	for _, edge := range calcOrder.LineItems.Edges {
		calcLineItemIDs = append(calcLineItemIDs, edge.Node.ID)
	}

	return calcOrder.ID, calcLineItemIDs, nil
}

func SetLineItemQuantity(client *gqlclient.Client, calcOrderID, lineItemID string, quantity int) error {
	vars := map[string]interface{}{
		"id":         calcOrderID,
		"lineItemId": lineItemID,
		"quantity":   quantity,
	}

	data, err := client.Execute(orderEditSetQuantityMutation, vars)
	if err != nil {
		return fmt.Errorf("cannot set quantity: %s", err)
	}

	var resp orderEditSetQuantityResponse
	if err := unmarshal(data, &resp); err != nil {
		return err
	}

	if len(resp.Data.OrderEditSetQuantity.UserErrors) > 0 {
		return fmt.Errorf("cannot set quantity: %s", resp.Data.OrderEditSetQuantity.UserErrors[0].Message)
	}

	return nil
}

func AddLineItemVariant(client *gqlclient.Client, calcOrderID, variantGID string, quantity int) error {
	vars := map[string]interface{}{
		"id":        calcOrderID,
		"variantId": variantGID,
		"quantity":  quantity,
	}

	data, err := client.Execute(orderEditAddVariantMutation, vars)
	if err != nil {
		return fmt.Errorf("cannot add variant: %s", err)
	}

	var resp orderEditAddVariantResponse
	if err := unmarshal(data, &resp); err != nil {
		return err
	}

	if len(resp.Data.OrderEditAddVariant.UserErrors) > 0 {
		return fmt.Errorf("cannot add variant: %s", resp.Data.OrderEditAddVariant.UserErrors[0].Message)
	}

	return nil
}

func CommitOrderEdit(client *gqlclient.Client, calcOrderID string) error {
	vars := map[string]interface{}{"id": calcOrderID}

	data, err := client.Execute(orderEditCommitMutation, vars)
	if err != nil {
		return fmt.Errorf("cannot commit order edit: %s", err)
	}

	var resp orderEditCommitResponse
	if err := unmarshal(data, &resp); err != nil {
		return err
	}

	if len(resp.Data.OrderEditCommit.UserErrors) > 0 {
		return fmt.Errorf("cannot commit order edit: %s", resp.Data.OrderEditCommit.UserErrors[0].Message)
	}

	return nil
}

func UpdateOrderNote(client *gqlclient.Client, orderGID, note string) error {
	vars := map[string]interface{}{
		"input": map[string]interface{}{
			"id":   orderGID,
			"note": note,
		},
	}

	data, err := client.Execute(orderUpdateMutation, vars)
	if err != nil {
		return fmt.Errorf("cannot update order: %s", err)
	}

	var resp orderUpdateResponse
	if err := unmarshal(data, &resp); err != nil {
		return err
	}

	if len(resp.Data.OrderUpdate.UserErrors) > 0 {
		return fmt.Errorf("cannot update order: %s", resp.Data.OrderUpdate.UserErrors[0].Message)
	}

	return nil
}

func UpdateOrderEmail(client *gqlclient.Client, orderGID, email string) error {
	vars := map[string]interface{}{
		"input": map[string]interface{}{
			"id":    orderGID,
			"email": email,
		},
	}

	data, err := client.Execute(orderUpdateMutation, vars)
	if err != nil {
		return fmt.Errorf("cannot update order email: %s", err)
	}

	var resp orderUpdateResponse
	if err := unmarshal(data, &resp); err != nil {
		return err
	}

	if len(resp.Data.OrderUpdate.UserErrors) > 0 {
		return fmt.Errorf("cannot update order email: %s", resp.Data.OrderUpdate.UserErrors[0].Message)
	}

	return nil
}

func CreateFulfillment(client *gqlclient.Client, fulfillmentOrderID string) error {
	vars := map[string]interface{}{
		"fulfillment": map[string]interface{}{
			"lineItemsByFulfillmentOrder": []map[string]interface{}{
				{"fulfillmentOrderId": fulfillmentOrderID},
			},
		},
	}

	data, err := client.Execute(fulfillmentCreateV2Mutation, vars)
	if err != nil {
		return fmt.Errorf("cannot create fulfillment: %s", err)
	}

	var resp fulfillmentCreateResponse
	if err := unmarshal(data, &resp); err != nil {
		return err
	}

	if len(resp.Data.FulfillmentCreateV2.UserErrors) > 0 {
		return fmt.Errorf("cannot create fulfillment: %s", resp.Data.FulfillmentCreateV2.UserErrors[0].Message)
	}

	return nil
}

func CancelFulfillment(client *gqlclient.Client, fulfillmentID string) error {
	vars := map[string]interface{}{"id": fulfillmentID}

	data, err := client.Execute(fulfillmentCancelMutation, vars)
	if err != nil {
		return fmt.Errorf("cannot cancel fulfillment: %s", err)
	}

	var resp fulfillmentCancelResponse
	if err := unmarshal(data, &resp); err != nil {
		return err
	}

	if len(resp.Data.FulfillmentCancel.UserErrors) > 0 {
		return fmt.Errorf("cannot cancel fulfillment: %s", resp.Data.FulfillmentCancel.UserErrors[0].Message)
	}

	return nil
}

func FindCustomerByEmail(client *gqlclient.Client, email string) (string, error) {
	vars := map[string]interface{}{"query": fmt.Sprintf("email:%s", email)}

	data, err := client.Execute(customerByEmailQuery, vars)
	if err != nil {
		return "", fmt.Errorf("cannot lookup customer by email %s: %s", email, err)
	}

	var resp customerLookupResponse
	if err := unmarshal(data, &resp); err != nil {
		return "", err
	}

	if len(resp.Data.Customers.Edges) == 0 {
		return "", fmt.Errorf("no customer found with email %s", email)
	}

	if len(resp.Data.Customers.Edges) > 1 {
		return "", fmt.Errorf("more than one customer found with email %s", email)
	}

	return resp.Data.Customers.Edges[0].Node.ID, nil
}

func SetOrderCustomer(client *gqlclient.Client, orderGID, customerGID string) error {
	vars := map[string]interface{}{
		"orderId":    orderGID,
		"customerId": customerGID,
	}

	data, err := client.Execute(orderCustomerSetMutation, vars)
	if err != nil {
		return fmt.Errorf("cannot set order customer: %s", err)
	}

	var resp orderCustomerSetResponse
	if err := unmarshal(data, &resp); err != nil {
		return err
	}

	if len(resp.Data.OrderCustomerSet.UserErrors) > 0 {
		return fmt.Errorf("cannot set order customer: %s", resp.Data.OrderCustomerSet.UserErrors[0].Message)
	}

	return nil
}
