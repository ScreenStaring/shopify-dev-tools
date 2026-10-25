package companies

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ScreenStaring/shopify-dev-tools/cmd/draftorders"
	"github.com/ScreenStaring/shopify-dev-tools/cmd/orders"
	gqlclient "github.com/ScreenStaring/shopify-dev-tools/gql"
)

const companyGIDPrefix = "gid://shopify/Company/"

// recentLimit is the number of a company's most recent orders and draft
// orders that are shown with it.
const recentLimit = 5

// companySelection is the fields and connections fetched for a company. The
// draft orders selection is only included when wanted: it requires the
// read_draft_orders scope, so querying it unconditionally would fail for
// tokens without it.
const companySelection = `
      id
      name
      externalId
      note
      createdAt
      updatedAt
      customerSince
`

const companyOrdersSelection = `      orders(first: $ordersFirst, sortKey: CREATED_AT, reverse: true) {
        nodes {
          legacyResourceId
          name
          createdAt
          displayFinancialStatus
          displayFulfillmentStatus
          lineItems(first: 250) {
            nodes {
              sku
              name
              quantity
              fulfillmentStatus
            }
          }
        }
      }
`

const companyDraftOrdersSelection = `      draftOrders(first: $draftOrdersFirst, sortKey: UPDATED_AT, reverse: true) {
        nodes {
          legacyResourceId
          name
          status
          createdAt
          lineItems(first: 250) {
            nodes {
              sku
              title
              quantity
            }
          }
        }
      }
`

// companyFields returns the selection for a company, including its recent
// draft orders only when includeDrafts is true.
func companyFields(includeDrafts bool) string {
	fields := companySelection

	if includeDrafts {
		fields += companyDraftOrdersSelection
	}

	return fields + companyOrdersSelection
}

// queryVariables returns the operation's variable declarations, which are
// ordered to match the connection arguments each of them feeds.
func queryVariables(base string, includeDrafts bool) string {
	vars := base + ", $ordersFirst: Int!"

	if includeDrafts {
		vars += ", $draftOrdersFirst: Int!"
	}

	return vars
}

// companiesQuery returns the query listing a shop's companies.
func companiesQuery(includeDrafts bool) string {
	return fmt.Sprintf(`query Companies(%s) {
  companies(first: $first) {
    nodes {%s    }
  }
}
`, queryVariables("$first: Int!", includeDrafts), companyFields(includeDrafts))
}

// companiesByIDsQuery returns the query fetching companies by GID.
func companiesByIDsQuery(includeDrafts bool) string {
	return fmt.Sprintf(`query CompaniesByIds(%s) {
  nodes(ids: $ids) {
    ... on Company {%s    }
  }
}
`, queryVariables("$ids: [ID!]!", includeDrafts), companyFields(includeDrafts))
}

// Company is a B2B company, along with its most recent draft orders and orders.
type Company struct {
	ID            string
	Name          string
	ExternalID    string
	Note          string
	CreatedAt     string
	UpdatedAt     string
	CustomerSince string
	DraftOrders   []draftorders.DraftOrder
	Orders        []orders.Order
}

type companyLineItemJSON struct {
	SKU               string `json:"sku"`
	Name              string `json:"name"`
	Title             string `json:"title"`
	Quantity          int    `json:"quantity"`
	FulfillmentStatus string `json:"fulfillmentStatus"`
}

type companyDraftOrderJSON struct {
	LegacyResourceId int64  `json:"legacyResourceId,string"`
	Name             string `json:"name"`
	Status           string `json:"status"`
	CreatedAt        string `json:"createdAt"`
	LineItems        struct {
		Nodes []companyLineItemJSON `json:"nodes"`
	} `json:"lineItems"`
}

type companyOrderJSON struct {
	LegacyResourceId         int64  `json:"legacyResourceId,string"`
	Name                     string `json:"name"`
	CreatedAt                string `json:"createdAt"`
	DisplayFinancialStatus   string `json:"displayFinancialStatus"`
	DisplayFulfillmentStatus string `json:"displayFulfillmentStatus"`
	LineItems                struct {
		Nodes []companyLineItemJSON `json:"nodes"`
	} `json:"lineItems"`
}

type companyJSON struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	ExternalID    string `json:"externalId"`
	Note          string `json:"note"`
	CreatedAt     string `json:"createdAt"`
	UpdatedAt     string `json:"updatedAt"`
	CustomerSince string `json:"customerSince"`
	DraftOrders   struct {
		Nodes []companyDraftOrderJSON `json:"nodes"`
	} `json:"draftOrders"`
	Orders struct {
		Nodes []companyOrderJSON `json:"nodes"`
	} `json:"orders"`
}

type companiesResponse struct {
	Data struct {
		Companies struct {
			Nodes []companyJSON `json:"nodes"`
		} `json:"companies"`
	} `json:"data"`
}

type companiesByIDsResponse struct {
	Data struct {
		Nodes []*companyJSON `json:"nodes"`
	} `json:"data"`
}

// ToCompanyGID returns the id as a Company GID, prepending the
// gid://shopify/Company/ prefix when given a bare id.
func ToCompanyGID(id string) string {
	if strings.HasPrefix(id, "gid://") {
		return id
	}

	return companyGIDPrefix + id
}

// FetchCompanies returns the shop's most recently created companies. Draft
// orders are only fetched, and need the read_draft_orders scope, when
// includeDrafts is true.
func FetchCompanies(client *gqlclient.Client, limit int, includeDrafts bool) ([]Company, error) {
	variables := map[string]interface{}{
		"first":       limit,
		"ordersFirst": recentLimit,
	}
	if includeDrafts {
		variables["draftOrdersFirst"] = recentLimit
	}

	data, err := client.Execute(companiesQuery(includeDrafts), variables)
	if err != nil {
		return nil, fmt.Errorf("Cannot list companies: %s", err)
	}

	b, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("Cannot re-encode companies response: %s", err)
	}

	var response companiesResponse
	if err := json.Unmarshal(b, &response); err != nil {
		return nil, fmt.Errorf("Cannot parse companies response: %s", err)
	}

	nodes := response.Data.Companies.Nodes
	companies := make([]Company, 0, len(nodes))
	for _, node := range nodes {
		companies = append(companies, companyFromJSON(node))
	}

	return companies, nil
}

// FetchCompaniesByIDs returns the companies with the given Company GIDs. The
// second return value holds the ids for which no company was found. Draft
// orders are only fetched, and need the read_draft_orders scope, when
// includeDrafts is true.
func FetchCompaniesByIDs(client *gqlclient.Client, ids []string, includeDrafts bool) ([]Company, []string, error) {
	variables := map[string]interface{}{
		"ids":         ids,
		"ordersFirst": recentLimit,
	}
	if includeDrafts {
		variables["draftOrdersFirst"] = recentLimit
	}

	data, err := client.Execute(companiesByIDsQuery(includeDrafts), variables)
	if err != nil {
		return nil, nil, fmt.Errorf("Cannot fetch companies: %s", err)
	}

	b, err := json.Marshal(data)
	if err != nil {
		return nil, nil, fmt.Errorf("Cannot re-encode companies response: %s", err)
	}

	var response companiesByIDsResponse
	if err := json.Unmarshal(b, &response); err != nil {
		return nil, nil, fmt.Errorf("Cannot parse companies response: %s", err)
	}

	var companies []Company
	var missing []string

	for i, node := range response.Data.Nodes {
		if node == nil {
			missing = append(missing, ids[i])
			continue
		}

		companies = append(companies, companyFromJSON(*node))
	}

	return companies, missing, nil
}

func companyFromJSON(n companyJSON) Company {
	company := Company{
		ID:            n.ID,
		Name:          n.Name,
		ExternalID:    n.ExternalID,
		Note:          n.Note,
		CreatedAt:     n.CreatedAt,
		UpdatedAt:     n.UpdatedAt,
		CustomerSince: n.CustomerSince,
	}

	for _, node := range n.DraftOrders.Nodes {
		company.DraftOrders = append(company.DraftOrders, draftOrderFromJSON(node))
	}

	for _, node := range n.Orders.Nodes {
		company.Orders = append(company.Orders, orderFromJSON(node))
	}

	return company
}

func draftOrderFromJSON(n companyDraftOrderJSON) draftorders.DraftOrder {
	draftOrder := draftorders.DraftOrder{
		ID:        n.LegacyResourceId,
		Name:      n.Name,
		Status:    n.Status,
		CreatedAt: n.CreatedAt,
	}

	for _, line := range n.LineItems.Nodes {
		draftOrder.LineItems = append(draftOrder.LineItems, draftorders.LineItem{
			SKU:      line.SKU,
			Name:     line.Title,
			Quantity: line.Quantity,
		})
	}

	return draftOrder
}

func orderFromJSON(n companyOrderJSON) orders.Order {
	order := orders.Order{
		ID:                       n.LegacyResourceId,
		Name:                     n.Name,
		CreatedAt:                n.CreatedAt,
		DisplayFinancialStatus:   n.DisplayFinancialStatus,
		DisplayFulfillmentStatus: n.DisplayFulfillmentStatus,
	}

	for _, line := range n.LineItems.Nodes {
		order.LineItems = append(order.LineItems, orders.LineItem{
			SKU:               line.SKU,
			Name:              line.Name,
			Quantity:          line.Quantity,
			FulfillmentStatus: line.FulfillmentStatus,
		})
	}

	return order
}
