package charges

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/ScreenStaring/shopify-dev-tools/gql"
)

const shopCurrencyQuery = `
query {
  shop {
    currencyCode
  }
}
`

const oneTimeChargeCreateMutation = `
mutation($name: String!, $price: MoneyInput!, $returnUrl: URL!, $test: Boolean) {
  appPurchaseOneTimeCreate(name: $name, price: $price, returnUrl: $returnUrl, test: $test) {
    appPurchaseOneTime {
      id
      name
      status
      price {
        amount
        currencyCode
      }
      test
      createdAt
    }
    confirmationUrl
    userErrors {
      field
      message
    }
  }
}
`

const oneTimeChargesQuery = `
query($first: Int!, $after: String) {
  currentAppInstallation {
    oneTimePurchases(first: $first, after: $after) {
      edges {
        node {
          id
          name
          status
          price {
            amount
            currencyCode
          }
          test
          createdAt
        }
      }
      pageInfo {
        hasNextPage
        endCursor
      }
    }
  }
}
`

const recurringChargesQuery = `
query($first: Int!, $after: String) {
  currentAppInstallation {
    allSubscriptions(first: $first, after: $after) {
      edges {
        node {
          id
          name
          status
          test
          returnUrl
          createdAt
          lineItems {
            plan {
              pricingDetails {
                ... on AppRecurringPricing {
                  price {
                    amount
                    currencyCode
                  }
                }
              }
            }
          }
        }
      }
      pageInfo {
        hasNextPage
        endCursor
      }
    }
  }
}
`

const chargesQuery = `
query($ids: [ID!]!) {
  nodes(ids: $ids) {
    __typename
    ... on AppPurchaseOneTime {
      id
      name
      oneTimeStatus: status
      price {
        amount
        currencyCode
      }
      test
      createdAt
    }
    ... on AppSubscription {
      id
      name
      subscriptionStatus: status
      test
      returnUrl
      createdAt
      lineItems {
        plan {
          pricingDetails {
            ... on AppRecurringPricing {
              price {
                amount
                currencyCode
              }
            }
          }
        }
      }
    }
  }
}
`

// OneTimeCharge is the package's native one-time charge (Application
// Charge) shape as returned by the Admin GraphQL API. ConfirmationURL and
// ReturnURL are only populated by the create mutation; the GraphQL API
// does not expose them on existing purchases.
type OneTimeCharge struct {
	ID              int64  `json:"id"`
	Name            string `json:"name"`
	Status          string `json:"status"`
	Price           string `json:"price"`
	Test            bool   `json:"test"`
	CreatedAt       string `json:"createdAt"`
	ConfirmationURL string `json:"confirmationUrl,omitempty"`
	ReturnURL       string `json:"returnUrl,omitempty"`
}

// RecurringCharge is the package's native recurring charge
// (RecurringApplicationCharge) shape as returned by the Admin GraphQL
// API.
type RecurringCharge struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	Price     string `json:"price"`
	Test      bool   `json:"test"`
	ReturnURL string `json:"returnUrl"`
	CreatedAt string `json:"createdAt"`
}

type priceJSON struct {
	Amount string `json:"amount"`
}

type oneTimeChargeNodeJSON struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Status    string     `json:"status"`
	Price     *priceJSON `json:"price"`
	Test      bool       `json:"test"`
	CreatedAt string     `json:"createdAt"`
}

type oneTimeChargeCreateResponse struct {
	Data struct {
		AppPurchaseOneTimeCreate struct {
			AppPurchaseOneTime *oneTimeChargeNodeJSON `json:"appPurchaseOneTime"`
			ConfirmationURL    string                 `json:"confirmationUrl"`
			UserErrors         []struct {
				Field   []string `json:"field"`
				Message string   `json:"message"`
			} `json:"userErrors"`
		} `json:"appPurchaseOneTimeCreate"`
	} `json:"data"`
}

type oneTimeChargesResponse struct {
	Data struct {
		CurrentAppInstallation struct {
			OneTimePurchases struct {
				Edges []struct {
					Node oneTimeChargeNodeJSON `json:"node"`
				} `json:"edges"`
				PageInfo struct {
					HasNextPage bool   `json:"hasNextPage"`
					EndCursor   string `json:"endCursor"`
				} `json:"pageInfo"`
			} `json:"oneTimePurchases"`
		} `json:"currentAppInstallation"`
	} `json:"data"`
}

type recurringChargeNodeJSON struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	Test      bool   `json:"test"`
	ReturnURL string `json:"returnUrl"`
	CreatedAt string `json:"createdAt"`
	LineItems []struct {
		Plan *struct {
			PricingDetails *struct {
				Price *priceJSON `json:"price"`
			} `json:"pricingDetails"`
		} `json:"plan"`
	} `json:"lineItems"`
}

type recurringChargesResponse struct {
	Data struct {
		CurrentAppInstallation struct {
			AllSubscriptions struct {
				Edges []struct {
					Node recurringChargeNodeJSON `json:"node"`
				} `json:"edges"`
				PageInfo struct {
					HasNextPage bool   `json:"hasNextPage"`
					EndCursor   string `json:"endCursor"`
				} `json:"pageInfo"`
			} `json:"allSubscriptions"`
		} `json:"currentAppInstallation"`
	} `json:"data"`
}

type chargeNodeJSON struct {
	Typename           string     `json:"__typename"`
	ID                 string     `json:"id"`
	Name               string     `json:"name"`
	OneTimeStatus      string     `json:"oneTimeStatus"`
	SubscriptionStatus string     `json:"subscriptionStatus"`
	Test               bool       `json:"test"`
	ReturnURL          string     `json:"returnUrl"`
	CreatedAt          string     `json:"createdAt"`
	Price              *priceJSON `json:"price"`
	LineItems          []struct {
		Plan *struct {
			PricingDetails *struct {
				Price *priceJSON `json:"price"`
			} `json:"pricingDetails"`
		} `json:"plan"`
	} `json:"lineItems"`
}

func shopCurrencyCode(client *gql.Client) (string, error) {
	data, err := client.Execute(shopCurrencyQuery)
	if err != nil {
		return "", fmt.Errorf("Cannot get shop currency: %s", err)
	}

	values, _ := data.ValuesForPath("data.shop.currencyCode")
	if len(values) == 0 {
		return "", fmt.Errorf("Cannot get shop currency: no currencyCode in response")
	}

	return fmt.Sprint(values[0]), nil
}

func createOneTimeCharge(client *gql.Client, name, price string, test bool, returnURL string) (*OneTimeCharge, error) {
	currencyCode, err := shopCurrencyCode(client)
	if err != nil {
		return nil, err
	}

	data, err := client.Execute(oneTimeChargeCreateMutation, map[string]interface{}{
		"name":      name,
		"price":     map[string]interface{}{"amount": price, "currencyCode": currencyCode},
		"returnUrl": returnURL,
		"test":      test,
	})
	if err != nil {
		return nil, fmt.Errorf("Cannot create one-time charge: %s", err)
	}

	b, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("Cannot re-encode one-time charge response: %s", err)
	}

	var response oneTimeChargeCreateResponse
	if err := json.Unmarshal(b, &response); err != nil {
		return nil, fmt.Errorf("Cannot parse one-time charge response: %s", err)
	}

	if errs := response.Data.AppPurchaseOneTimeCreate.UserErrors; len(errs) > 0 {
		var messages []string
		for _, e := range errs {
			messages = append(messages, e.Message)
		}
		return nil, fmt.Errorf("Cannot create one-time charge: %s", strings.Join(messages, ", "))
	}

	n := response.Data.AppPurchaseOneTimeCreate.AppPurchaseOneTime
	if n == nil {
		return nil, fmt.Errorf("Cannot create one-time charge: no charge in response")
	}

	return oneTimeChargeFromNode(n, response.Data.AppPurchaseOneTimeCreate.ConfirmationURL, returnURL), nil
}

func oneTimeChargeFromNode(n *oneTimeChargeNodeJSON, confirmationURL, returnURL string) *OneTimeCharge {
	charge := &OneTimeCharge{
		ID:              chargeIDFromGID(n.ID),
		Name:            n.Name,
		Status:          n.Status,
		Test:            n.Test,
		CreatedAt:       n.CreatedAt,
		ConfirmationURL: confirmationURL,
		ReturnURL:       returnURL,
	}

	if n.Price != nil {
		charge.Price = n.Price.Amount
	}

	return charge
}

func recurringChargeFromNode(n recurringChargeNodeJSON) RecurringCharge {
	charge := RecurringCharge{
		ID:        chargeIDFromGID(n.ID),
		Name:      n.Name,
		Status:    n.Status,
		Test:      n.Test,
		ReturnURL: n.ReturnURL,
		CreatedAt: n.CreatedAt,
	}

	for _, lineItem := range n.LineItems {
		if lineItem.Plan == nil || lineItem.Plan.PricingDetails == nil || lineItem.Plan.PricingDetails.Price == nil {
			continue
		}

		charge.Price = lineItem.Plan.PricingDetails.Price.Amount
		break
	}

	return charge
}

func listOneTimeChargesGQL(client *gql.Client) ([]OneTimeCharge, error) {
	vars := map[string]interface{}{
		"first": 250,
	}

	var charges []OneTimeCharge

	for {
		data, err := client.Execute(oneTimeChargesQuery, vars)
		if err != nil {
			return nil, fmt.Errorf("Cannot list one-time charges: %s", err)
		}

		b, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("Cannot list one-time charges: %s", err)
		}

		var response oneTimeChargesResponse
		if err := json.Unmarshal(b, &response); err != nil {
			return nil, fmt.Errorf("Cannot list one-time charges: %s", err)
		}

		for _, edge := range response.Data.CurrentAppInstallation.OneTimePurchases.Edges {
			charges = append(charges, *oneTimeChargeFromNode(&edge.Node, "", ""))
		}

		if !response.Data.CurrentAppInstallation.OneTimePurchases.PageInfo.HasNextPage {
			break
		}

		vars["after"] = response.Data.CurrentAppInstallation.OneTimePurchases.PageInfo.EndCursor
	}

	return charges, nil
}

func listRecurringChargesGQL(client *gql.Client) ([]RecurringCharge, error) {
	vars := map[string]interface{}{
		"first": 250,
	}

	var charges []RecurringCharge

	for {
		data, err := client.Execute(recurringChargesQuery, vars)
		if err != nil {
			return nil, fmt.Errorf("Cannot list recurring charges: %s", err)
		}

		b, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("Cannot list recurring charges: %s", err)
		}

		var response recurringChargesResponse
		if err := json.Unmarshal(b, &response); err != nil {
			return nil, fmt.Errorf("Cannot list recurring charges: %s", err)
		}

		for _, edge := range response.Data.CurrentAppInstallation.AllSubscriptions.Edges {
			charges = append(charges, recurringChargeFromNode(edge.Node))
		}

		if !response.Data.CurrentAppInstallation.AllSubscriptions.PageInfo.HasNextPage {
			break
		}

		vars["after"] = response.Data.CurrentAppInstallation.AllSubscriptions.PageInfo.EndCursor
	}

	return charges, nil
}

// fetchChargesByID returns the charges for the given ids, which must all
// be of the given GraphQL type (AppPurchaseOneTime or AppSubscription).
func fetchChargesByID(client *gql.Client, ids []int64, gidPrefix string) ([]chargeNodeJSON, error) {
	gids := make([]string, 0, len(ids))
	for _, id := range ids {
		gids = append(gids, fmt.Sprintf("gid://shopify/%s/%d", gidPrefix, id))
	}

	data, err := client.Execute(chargesQuery, map[string]interface{}{"ids": gids})
	if err != nil {
		return nil, err
	}

	b, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	var response struct {
		Data struct {
			Nodes []*chargeNodeJSON `json:"nodes"`
		} `json:"data"`
	}

	if err := json.Unmarshal(b, &response); err != nil {
		return nil, err
	}

	var nodes []chargeNodeJSON
	for i, node := range response.Data.Nodes {
		if node == nil {
			return nil, fmt.Errorf("no charge found with id %d", ids[i])
		}

		nodes = append(nodes, *node)
	}

	return nodes, nil
}

func getOneTimeChargesByID(client *gql.Client, ids []int64) ([]OneTimeCharge, error) {
	nodes, err := fetchChargesByID(client, ids, "AppPurchaseOneTime")
	if err != nil {
		return nil, err
	}

	var charges []OneTimeCharge
	for i, node := range nodes {
		if node.Typename != "AppPurchaseOneTime" {
			return nil, fmt.Errorf("Cannot get one-time charge %d: no charge found with that id", ids[i])
		}

		n := &oneTimeChargeNodeJSON{
			ID:        node.ID,
			Name:      node.Name,
			Status:    node.OneTimeStatus,
			Price:     node.Price,
			Test:      node.Test,
			CreatedAt: node.CreatedAt,
		}
		charges = append(charges, *oneTimeChargeFromNode(n, "", ""))
	}

	return charges, nil
}

func getRecurringChargesByID(client *gql.Client, ids []int64) ([]RecurringCharge, error) {
	nodes, err := fetchChargesByID(client, ids, "AppSubscription")
	if err != nil {
		return nil, err
	}

	var charges []RecurringCharge
	for i, node := range nodes {
		if node.Typename != "AppSubscription" {
			return nil, fmt.Errorf("Cannot get recurring charge %d: no charge found with that id", ids[i])
		}

		n := recurringChargeNodeJSON{
			ID:        node.ID,
			Name:      node.Name,
			Status:    node.SubscriptionStatus,
			Test:      node.Test,
			ReturnURL: node.ReturnURL,
			CreatedAt: node.CreatedAt,
		}
		n.LineItems = node.LineItems
		charges = append(charges, recurringChargeFromNode(n))
	}

	return charges, nil
}

func chargeIDFromGID(gid string) int64 {
	// gid://shopify/AppPurchaseOneTime/123456
	parts := strings.Split(gid, "/")
	if len(parts) == 0 {
		return 0
	}

	id, err := strconv.ParseInt(parts[len(parts)-1], 10, 64)
	if err != nil {
		return 0
	}

	return id
}
