package gql

import (
	"encoding/json"
	"fmt"
	"strings"

	gqlclient "github.com/ScreenStaring/shopify-dev-tools/gql"
)

const customerQuery = `
query($id: ID!) {
  customer(id: $id) {
    id
    firstName
    lastName
    defaultEmailAddress {
      emailAddress
    }
    defaultPhoneNumber {
      phoneNumber
    }
    numberOfOrders
    createdAt
  }
}
`

const customersQuery = `
query($first: Int!) {
  customers(first: $first, sortKey: CREATED_AT, reverse: true) {
    nodes {
      id
      firstName
      lastName
      defaultEmailAddress {
        emailAddress
      }
      defaultPhoneNumber {
        phoneNumber
      }
      numberOfOrders
      createdAt
    }
  }
}
`

type Customer struct {
	ID             string
	FirstName      string
	LastName       string
	Email          string
	Phone          string
	NumberOfOrders int
	CreatedAt      string
}

type customerEmailJSON struct {
	EmailAddress string `json:"emailAddress"`
}

type customerPhoneJSON struct {
	PhoneNumber string `json:"phoneNumber"`
}

type customerJSON struct {
	ID                  string             `json:"id"`
	FirstName           string             `json:"firstName"`
	LastName            string             `json:"lastName"`
	DefaultEmailAddress *customerEmailJSON `json:"defaultEmailAddress"`
	DefaultPhoneNumber  *customerPhoneJSON `json:"defaultPhoneNumber"`
	NumberOfOrders      int                `json:"numberOfOrders,string"`
	CreatedAt           string             `json:"createdAt"`
}

func CustomerToGID(id string) string {
	if strings.HasPrefix(id, "gid://") {
		return id
	}
	return "gid://shopify/Customer/" + id
}

func jsonToCustomer(n customerJSON) Customer {
	c := Customer{
		ID:             n.ID,
		FirstName:      n.FirstName,
		LastName:       n.LastName,
		NumberOfOrders: n.NumberOfOrders,
		CreatedAt:      n.CreatedAt,
	}
	if n.DefaultEmailAddress != nil {
		c.Email = n.DefaultEmailAddress.EmailAddress
	}
	if n.DefaultPhoneNumber != nil {
		c.Phone = n.DefaultPhoneNumber.PhoneNumber
	}
	return c
}

func GetCustomer(shop, token, id string) (*Customer, error) {
	client := gqlclient.NewClient(shop, token)

	data, err := client.Execute(customerQuery, map[string]interface{}{"id": CustomerToGID(id)})
	if err != nil {
		return nil, fmt.Errorf("Cannot get customer: %s", err)
	}

	b, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("Cannot re-encode customer response: %s", err)
	}

	var response struct {
		Data struct {
			Customer *customerJSON `json:"customer"`
		} `json:"data"`
	}

	if err := json.Unmarshal(b, &response); err != nil {
		return nil, fmt.Errorf("Cannot parse customer response: %s", err)
	}

	if response.Data.Customer == nil {
		return nil, fmt.Errorf("Customer not found")
	}

	c := jsonToCustomer(*response.Data.Customer)
	return &c, nil
}

func ListCustomers(shop, token string, limit int) ([]Customer, error) {
	client := gqlclient.NewClient(shop, token)

	data, err := client.Execute(customersQuery, map[string]interface{}{"first": limit})
	if err != nil {
		return nil, fmt.Errorf("Cannot list customers: %s", err)
	}

	b, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("Cannot re-encode customers response: %s", err)
	}

	var response struct {
		Data struct {
			Customers struct {
				Nodes []customerJSON `json:"nodes"`
			} `json:"customers"`
		} `json:"data"`
	}

	if err := json.Unmarshal(b, &response); err != nil {
		return nil, fmt.Errorf("Cannot parse customers response: %s", err)
	}

	var result []Customer
	for _, n := range response.Data.Customers.Nodes {
		result = append(result, jsonToCustomer(n))
	}

	return result, nil
}
