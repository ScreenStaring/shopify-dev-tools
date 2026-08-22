package metafields

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/ScreenStaring/shopify-dev-tools/gql"
)

const metafieldDefinitionsQuery = `
query($ownerType: MetafieldOwnerType!, $first: Int!, $after: String, $namespace: String) {
  metafieldDefinitions(ownerType: $ownerType, first: $first, after: $after, namespace: $namespace) {
    edges {
      node {
        id
        name
        namespace
        key
        description
        type {
          name
        }
        ownerType
      }
    }
    pageInfo {
      hasNextPage
      endCursor
    }
  }
}
`

type MetafieldDefinition struct {
	ID          string
	Name        string
	Namespace   string
	Key         string
	Description string
	Type        string
	OwnerType   string
}

// Metafield is the package's native metafield shape as returned by the
// Admin GraphQL API (gid string, string timestamps).
type Metafield struct {
	ID          string `json:"id"`
	Namespace   string `json:"namespace"`
	Key         string `json:"key"`
	Description string `json:"description"`
	Value       string `json:"value"`
	Type        string `json:"type"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

// orderFilter holds the criteria used to resolve order IDs by name/sku.
type orderFilter struct {
	IDs   []int64
	Names []string
	SKUs  []string
}

type metafieldDefinitionsResponse struct {
	Data struct {
		MetafieldDefinitions struct {
			Edges []struct {
				Node struct {
					ID          string `json:"id"`
					Name        string `json:"name"`
					Namespace   string `json:"namespace"`
					Key         string `json:"key"`
					Description string `json:"description"`
					Type        struct {
						Name string `json:"name"`
					} `json:"type"`
					OwnerType string `json:"ownerType"`
				} `json:"node"`
			} `json:"edges"`
			PageInfo struct {
				HasNextPage bool   `json:"hasNextPage"`
				EndCursor   string `json:"endCursor"`
			} `json:"pageInfo"`
		} `json:"metafieldDefinitions"`
	} `json:"data"`
}

func listMetafieldDefinitions(client *gql.Client, ownerType, namespace string) ([]MetafieldDefinition, error) {
	vars := map[string]interface{}{
		"ownerType": ownerType,
		"first":     250,
	}

	if namespace != "" {
		vars["namespace"] = namespace
	}

	var definitions []MetafieldDefinition

	for {
		data, err := client.Execute(metafieldDefinitionsQuery, vars)
		if err != nil {
			return nil, fmt.Errorf("Cannot list metafield definitions: %s", err)
		}

		b, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("Cannot list metafield definitions: %s", err)
		}

		var response metafieldDefinitionsResponse
		if err := json.Unmarshal(b, &response); err != nil {
			return nil, fmt.Errorf("Cannot list metafield definitions: %s", err)
		}

		for _, edge := range response.Data.MetafieldDefinitions.Edges {
			n := edge.Node
			definitions = append(definitions, MetafieldDefinition{
				ID:          n.ID,
				Name:        n.Name,
				Namespace:   n.Namespace,
				Key:         n.Key,
				Description: n.Description,
				Type:        n.Type.Name,
				OwnerType:   n.OwnerType,
			})
		}

		if !response.Data.MetafieldDefinitions.PageInfo.HasNextPage {
			break
		}

		vars["after"] = response.Data.MetafieldDefinitions.PageInfo.EndCursor
	}

	return definitions, nil
}

const appInstallationMetafieldsQuery = `
query($first: Int!, $after: String, $namespace: String) {
  currentAppInstallation {
    id
    metafields(first: $first, after: $after, namespace: $namespace) {
      edges {
        node {
          id
          namespace
          key
          description
          value
          type
          createdAt
          updatedAt
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

type appInstallationMetafieldsResponse struct {
	Data struct {
		CurrentAppInstallation struct {
			ID         string `json:"id"`
			Metafields struct {
				Edges []struct {
					Node struct {
						ID          string `json:"id"`
						Namespace   string `json:"namespace"`
						Key         string `json:"key"`
						Description string `json:"description"`
						Value       string `json:"value"`
						Type        string `json:"type"`
						CreatedAt   string `json:"createdAt"`
						UpdatedAt   string `json:"updatedAt"`
					} `json:"node"`
				} `json:"edges"`
				PageInfo struct {
					HasNextPage bool   `json:"hasNextPage"`
					EndCursor   string `json:"endCursor"`
				} `json:"pageInfo"`
			} `json:"metafields"`
		} `json:"currentAppInstallation"`
	} `json:"data"`
}

func listAppInstallationMetafields(client *gql.Client, namespace string) ([]Metafield, error) {
	vars := map[string]interface{}{
		"first": 250,
	}

	if namespace != "" {
		vars["namespace"] = namespace
	}

	var metafields []Metafield

	for {
		data, err := client.Execute(appInstallationMetafieldsQuery, vars)
		if err != nil {
			return nil, fmt.Errorf("Cannot list metafields for app installation: %s", err)
		}

		b, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("Cannot list metafields for app installation: %s", err)
		}

		var response appInstallationMetafieldsResponse
		if err := json.Unmarshal(b, &response); err != nil {
			return nil, fmt.Errorf("Cannot list metafields for app installation: %s", err)
		}

		for _, edge := range response.Data.CurrentAppInstallation.Metafields.Edges {
			n := edge.Node
			metafields = append(metafields, Metafield{
				ID:          n.ID,
				Namespace:   n.Namespace,
				Key:         n.Key,
				Description: n.Description,
				Value:       n.Value,
				Type:        n.Type,
				CreatedAt:   n.CreatedAt,
				UpdatedAt:   n.UpdatedAt,
			})
		}

		if !response.Data.CurrentAppInstallation.Metafields.PageInfo.HasNextPage {
			break
		}

		vars["after"] = response.Data.CurrentAppInstallation.Metafields.PageInfo.EndCursor
	}

	return metafields, nil
}

const customerMetafieldsQuery = `
query($ownerId: ID!, $first: Int!, $after: String, $namespace: String, $keys: [String!], $reverse: Boolean) {
  customer(id: $ownerId) {
    id
    metafields(first: $first, after: $after, namespace: $namespace, keys: $keys, reverse: $reverse) {
      edges {
        node {
          id
          namespace
          key
          description
          value
          type
          createdAt
          updatedAt
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

type customerMetafieldsResponse struct {
	Data struct {
		Customer struct {
			ID         string `json:"id"`
			Metafields struct {
				Edges []struct {
					Node struct {
						ID          string `json:"id"`
						Namespace   string `json:"namespace"`
						Key         string `json:"key"`
						Description string `json:"description"`
						Value       string `json:"value"`
						Type        string `json:"type"`
						CreatedAt   string `json:"createdAt"`
						UpdatedAt   string `json:"updatedAt"`
					} `json:"node"`
				} `json:"edges"`
				PageInfo struct {
					HasNextPage bool   `json:"hasNextPage"`
					EndCursor   string `json:"endCursor"`
				} `json:"pageInfo"`
			} `json:"metafields"`
		} `json:"customer"`
	} `json:"data"`
}

func listCustomerMetafields(client *gql.Client, customerID int64, namespace, key string, reverse bool) ([]Metafield, error) {
	vars := map[string]interface{}{
		"ownerId": fmt.Sprintf("gid://shopify/Customer/%d", customerID),
		"first":   250,
	}

	if namespace != "" {
		vars["namespace"] = namespace
	}

	if reverse {
		vars["reverse"] = true
	}

	// The GraphQL keys argument requires the namespace.key format, so a bare
	// key filter (no namespace) is applied client-side below.
	filterByKey := false
	if key != "" {
		if namespace != "" {
			vars["keys"] = []string{namespace + "." + key}
		} else {
			filterByKey = true
		}
	}

	var metafields []Metafield

	for {
		data, err := client.Execute(customerMetafieldsQuery, vars)
		if err != nil {
			return nil, fmt.Errorf("Cannot list metafields for customer: %s", err)
		}

		b, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("Cannot list metafields for customer: %s", err)
		}

		var response customerMetafieldsResponse
		if err := json.Unmarshal(b, &response); err != nil {
			return nil, fmt.Errorf("Cannot list metafields for customer: %s", err)
		}

		for _, edge := range response.Data.Customer.Metafields.Edges {
			n := edge.Node
			if filterByKey && n.Key != key {
				continue
			}

			metafields = append(metafields, Metafield{
				ID:          n.ID,
				Namespace:   n.Namespace,
				Key:         n.Key,
				Description: n.Description,
				Value:       n.Value,
				Type:        n.Type,
				CreatedAt:   n.CreatedAt,
				UpdatedAt:   n.UpdatedAt,
			})
		}

		if !response.Data.Customer.Metafields.PageInfo.HasNextPage {
			break
		}

		vars["after"] = response.Data.Customer.Metafields.PageInfo.EndCursor
	}

	return metafields, nil
}

const productMetafieldsQuery = `
query($ownerId: ID!, $first: Int!, $after: String, $namespace: String, $keys: [String!], $reverse: Boolean) {
  product(id: $ownerId) {
    id
    metafields(first: $first, after: $after, namespace: $namespace, keys: $keys, reverse: $reverse) {
      edges {
        node {
          id
          namespace
          key
          description
          value
          type
          createdAt
          updatedAt
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

type productMetafieldsResponse struct {
	Data struct {
		Product struct {
			ID         string `json:"id"`
			Metafields struct {
				Edges []struct {
					Node struct {
						ID          string `json:"id"`
						Namespace   string `json:"namespace"`
						Key         string `json:"key"`
						Description string `json:"description"`
						Value       string `json:"value"`
						Type        string `json:"type"`
						CreatedAt   string `json:"createdAt"`
						UpdatedAt   string `json:"updatedAt"`
					} `json:"node"`
				} `json:"edges"`
				PageInfo struct {
					HasNextPage bool   `json:"hasNextPage"`
					EndCursor   string `json:"endCursor"`
				} `json:"pageInfo"`
			} `json:"metafields"`
		} `json:"product"`
	} `json:"data"`
}

// listProductMetafields lists metafields for the given product. When the
// product doesn't exist (e.g. it was deleted or access is denied) the query
// returns null and the error is non-nil.
func listProductMetafields(client *gql.Client, productID int64, namespace, key string, reverse bool) ([]Metafield, error) {
	vars := map[string]interface{}{
		"ownerId": fmt.Sprintf("gid://shopify/Product/%d", productID),
		"first":   250,
	}

	if namespace != "" {
		vars["namespace"] = namespace
	}

	if reverse {
		vars["reverse"] = true
	}

	// The GraphQL keys argument requires the namespace.key format, so a bare
	// key filter (no namespace) is applied client-side below.
	filterByKey := false
	if key != "" {
		if namespace != "" {
			vars["keys"] = []string{namespace + "." + key}
		} else {
			filterByKey = true
		}
	}

	var metafields []Metafield
	found := false

	for {
		data, err := client.Execute(productMetafieldsQuery, vars)
		if err != nil {
			return nil, fmt.Errorf("Cannot list metafields for product: %s", err)
		}

		b, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("Cannot list metafields for product: %s", err)
		}

		var response productMetafieldsResponse
		if err := json.Unmarshal(b, &response); err != nil {
			return nil, fmt.Errorf("Cannot list metafields for product: %s", err)
		}

		if response.Data.Product.ID != "" {
			found = true
		}

		for _, edge := range response.Data.Product.Metafields.Edges {
			n := edge.Node
			if filterByKey && n.Key != key {
				continue
			}

			metafields = append(metafields, Metafield{
				ID:          n.ID,
				Namespace:   n.Namespace,
				Key:         n.Key,
				Description: n.Description,
				Value:       n.Value,
				Type:        n.Type,
				CreatedAt:   n.CreatedAt,
				UpdatedAt:   n.UpdatedAt,
			})
		}

		if !response.Data.Product.Metafields.PageInfo.HasNextPage {
			break
		}

		vars["after"] = response.Data.Product.Metafields.PageInfo.EndCursor
	}

	if !found {
		return nil, errors.New("not found")
	}

	return metafields, nil
}

const productMetafieldsBySkuQuery = `
query($query: String!, $first: Int!, $after: String, $namespace: String, $keys: [String!], $reverse: Boolean) {
  products(first: $first, after: $after, query: $query) {
    edges {
      node {
        id
        variants(first: 250) {
          nodes {
            sku
          }
        }
        metafields(first: 250, namespace: $namespace, keys: $keys, reverse: $reverse) {
          edges {
            node {
              id
              namespace
              key
              description
              value
              type
              createdAt
              updatedAt
            }
          }
          pageInfo {
            hasNextPage
            endCursor
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
`

type productMetafieldsBySkuResponse struct {
	Data struct {
		Products struct {
			Edges []struct {
				Node struct {
					ID       string `json:"id"`
					Variants struct {
						Nodes []struct {
							SKU string `json:"sku"`
						} `json:"nodes"`
					} `json:"variants"`
					Metafields struct {
						Edges []struct {
							Node struct {
								ID          string `json:"id"`
								Namespace   string `json:"namespace"`
								Key         string `json:"key"`
								Description string `json:"description"`
								Value       string `json:"value"`
								Type        string `json:"type"`
								CreatedAt   string `json:"createdAt"`
								UpdatedAt   string `json:"updatedAt"`
							} `json:"node"`
						} `json:"edges"`
						PageInfo struct {
							HasNextPage bool   `json:"hasNextPage"`
							EndCursor   string `json:"endCursor"`
						} `json:"pageInfo"`
					} `json:"metafields"`
				} `json:"node"`
			} `json:"edges"`
			PageInfo struct {
				HasNextPage bool   `json:"hasNextPage"`
				EndCursor   string `json:"endCursor"`
			} `json:"pageInfo"`
		} `json:"products"`
	} `json:"data"`
}

// listProductMetafieldsBySku lists metafields for the products whose variants
// have any of the given SKUs, filtering the products connection directly by
// sku instead of resolving SKUs to product IDs first. Metafields are limited
// to the first 250 per product. The returned slice reports which of the
// requested SKUs matched a variant; products with more than 250 variants may
// report a false miss.
func listProductMetafieldsBySku(client *gql.Client, skus []string, namespace, key string, reverse bool) ([]Metafield, []string, error) {
	vars := map[string]interface{}{
		"query": skuQuery(skus),
		"first": 250,
	}

	// The GraphQL keys argument requires the namespace.key format, so a bare
	// key filter (no namespace) is applied client-side below.
	filterByKey := false
	if key != "" {
		if namespace != "" {
			vars["keys"] = []string{namespace + "." + key}
		} else {
			filterByKey = true
		}
	}

	if namespace != "" {
		vars["namespace"] = namespace
	}

	if reverse {
		vars["reverse"] = true
	}

	var metafields []Metafield
	foundSkus := make(map[string]bool)

	for {
		data, err := client.Execute(productMetafieldsBySkuQuery, vars)
		if err != nil {
			return nil, nil, fmt.Errorf("Cannot list metafields for product: %s", err)
		}

		b, err := json.Marshal(data)
		if err != nil {
			return nil, nil, fmt.Errorf("Cannot list metafields for product: %s", err)
		}

		var response productMetafieldsBySkuResponse
		if err := json.Unmarshal(b, &response); err != nil {
			return nil, nil, fmt.Errorf("Cannot list metafields for product: %s", err)
		}

		for _, edge := range response.Data.Products.Edges {
			for _, v := range edge.Node.Variants.Nodes {
				if v.SKU != "" {
					foundSkus[v.SKU] = true
				}
			}

			for _, mfEdge := range edge.Node.Metafields.Edges {
				n := mfEdge.Node
				if filterByKey && n.Key != key {
					continue
				}

				metafields = append(metafields, Metafield{
					ID:          n.ID,
					Namespace:   n.Namespace,
					Key:         n.Key,
					Description: n.Description,
					Value:       n.Value,
					Type:        n.Type,
					CreatedAt:   n.CreatedAt,
					UpdatedAt:   n.UpdatedAt,
				})
			}
		}

		if !response.Data.Products.PageInfo.HasNextPage {
			break
		}

		vars["after"] = response.Data.Products.PageInfo.EndCursor
	}

	var found []string
	for _, sku := range skus {
		if foundSkus[sku] {
			found = append(found, sku)
		}
	}

	return metafields, found, nil
}

const variantMetafieldsBySkuQuery = `
query($query: String!, $first: Int!, $after: String, $namespace: String, $keys: [String!], $reverse: Boolean) {
  productVariants(first: $first, after: $after, query: $query) {
    edges {
      node {
        id
        sku
        metafields(first: 250, namespace: $namespace, keys: $keys, reverse: $reverse) {
          edges {
            node {
              id
              namespace
              key
              description
              value
              type
              createdAt
              updatedAt
            }
          }
          pageInfo {
            hasNextPage
            endCursor
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
`

type variantMetafieldsBySkuResponse struct {
	Data struct {
		ProductVariants struct {
			Edges []struct {
				Node struct {
					ID         string `json:"id"`
					SKU        string `json:"sku"`
					Metafields struct {
						Edges []struct {
							Node struct {
								ID          string `json:"id"`
								Namespace   string `json:"namespace"`
								Key         string `json:"key"`
								Description string `json:"description"`
								Value       string `json:"value"`
								Type        string `json:"type"`
								CreatedAt   string `json:"createdAt"`
								UpdatedAt   string `json:"updatedAt"`
							} `json:"node"`
						} `json:"edges"`
						PageInfo struct {
							HasNextPage bool   `json:"hasNextPage"`
							EndCursor   string `json:"endCursor"`
						} `json:"pageInfo"`
					} `json:"metafields"`
				} `json:"node"`
			} `json:"edges"`
			PageInfo struct {
				HasNextPage bool   `json:"hasNextPage"`
				EndCursor   string `json:"endCursor"`
			} `json:"pageInfo"`
		} `json:"productVariants"`
	} `json:"data"`
}

// listVariantMetafieldsBySku lists metafields for the variants with any of the
// given SKUs, filtering the productVariants connection directly by sku instead
// of resolving SKUs to variant IDs first. Metafields are limited to the first
// 250 per variant. The returned slice reports which of the requested SKUs
// matched a variant.
func listVariantMetafieldsBySku(client *gql.Client, skus []string, namespace, key string, reverse bool) ([]Metafield, []string, error) {
	vars := map[string]interface{}{
		"query": skuQuery(skus),
		"first": 250,
	}

	// The GraphQL keys argument requires the namespace.key format, so a bare
	// key filter (no namespace) is applied client-side below.
	filterByKey := false
	if key != "" {
		if namespace != "" {
			vars["keys"] = []string{namespace + "." + key}
		} else {
			filterByKey = true
		}
	}

	if namespace != "" {
		vars["namespace"] = namespace
	}

	if reverse {
		vars["reverse"] = true
	}

	var metafields []Metafield
	foundSkus := make(map[string]bool)

	for {
		data, err := client.Execute(variantMetafieldsBySkuQuery, vars)
		if err != nil {
			return nil, nil, fmt.Errorf("Cannot list metafields for variant: %s", err)
		}

		b, err := json.Marshal(data)
		if err != nil {
			return nil, nil, fmt.Errorf("Cannot list metafields for variant: %s", err)
		}

		var response variantMetafieldsBySkuResponse
		if err := json.Unmarshal(b, &response); err != nil {
			return nil, nil, fmt.Errorf("Cannot list metafields for variant: %s", err)
		}

		for _, edge := range response.Data.ProductVariants.Edges {
			if edge.Node.SKU != "" {
				foundSkus[edge.Node.SKU] = true
			}

			for _, mfEdge := range edge.Node.Metafields.Edges {
				n := mfEdge.Node
				if filterByKey && n.Key != key {
					continue
				}

				metafields = append(metafields, Metafield{
					ID:          n.ID,
					Namespace:   n.Namespace,
					Key:         n.Key,
					Description: n.Description,
					Value:       n.Value,
					Type:        n.Type,
					CreatedAt:   n.CreatedAt,
					UpdatedAt:   n.UpdatedAt,
				})
			}
		}

		if !response.Data.ProductVariants.PageInfo.HasNextPage {
			break
		}

		vars["after"] = response.Data.ProductVariants.PageInfo.EndCursor
	}

	var found []string
	for _, sku := range skus {
		if foundSkus[sku] {
			found = append(found, sku)
		}
	}

	return metafields, found, nil
}

// skuQuery builds a GraphQL search query matching any of the given SKUs,
// e.g. "sku:FOO OR sku:BAR".
func skuQuery(skus []string) string {
	parts := make([]string, len(skus))
	for i, sku := range skus {
		parts[i] = "sku:" + sku
	}
	return strings.Join(parts, " OR ")
}

const variantMetafieldsQuery = `
query($ownerId: ID!, $first: Int!, $after: String, $namespace: String, $keys: [String!], $reverse: Boolean) {
  productVariant(id: $ownerId) {
    id
    metafields(first: $first, after: $after, namespace: $namespace, keys: $keys, reverse: $reverse) {
      edges {
        node {
          id
          namespace
          key
          description
          value
          type
          createdAt
          updatedAt
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

type variantMetafieldsResponse struct {
	Data struct {
		ProductVariant struct {
			ID         string `json:"id"`
			Metafields struct {
				Edges []struct {
					Node struct {
						ID          string `json:"id"`
						Namespace   string `json:"namespace"`
						Key         string `json:"key"`
						Description string `json:"description"`
						Value       string `json:"value"`
						Type        string `json:"type"`
						CreatedAt   string `json:"createdAt"`
						UpdatedAt   string `json:"updatedAt"`
					} `json:"node"`
				} `json:"edges"`
				PageInfo struct {
					HasNextPage bool   `json:"hasNextPage"`
					EndCursor   string `json:"endCursor"`
				} `json:"pageInfo"`
			} `json:"metafields"`
		} `json:"productVariant"`
	} `json:"data"`
}

// listVariantMetafields lists metafields for the given variant. When the
// variant doesn't exist (e.g. it was deleted or access is denied) the query
// returns null and the error is non-nil.
func listVariantMetafields(client *gql.Client, variantID int64, namespace, key string, reverse bool) ([]Metafield, error) {
	vars := map[string]interface{}{
		"ownerId": fmt.Sprintf("gid://shopify/ProductVariant/%d", variantID),
		"first":   250,
	}

	if namespace != "" {
		vars["namespace"] = namespace
	}

	if reverse {
		vars["reverse"] = true
	}

	// The GraphQL keys argument requires the namespace.key format, so a bare
	// key filter (no namespace) is applied client-side below.
	filterByKey := false
	if key != "" {
		if namespace != "" {
			vars["keys"] = []string{namespace + "." + key}
		} else {
			filterByKey = true
		}
	}

	var metafields []Metafield
	found := false

	for {
		data, err := client.Execute(variantMetafieldsQuery, vars)
		if err != nil {
			return nil, fmt.Errorf("Cannot list metafields for variant: %s", err)
		}

		b, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("Cannot list metafields for variant: %s", err)
		}

		var response variantMetafieldsResponse
		if err := json.Unmarshal(b, &response); err != nil {
			return nil, fmt.Errorf("Cannot list metafields for variant: %s", err)
		}

		if response.Data.ProductVariant.ID != "" {
			found = true
		}

		for _, edge := range response.Data.ProductVariant.Metafields.Edges {
			n := edge.Node
			if filterByKey && n.Key != key {
				continue
			}

			metafields = append(metafields, Metafield{
				ID:          n.ID,
				Namespace:   n.Namespace,
				Key:         n.Key,
				Description: n.Description,
				Value:       n.Value,
				Type:        n.Type,
				CreatedAt:   n.CreatedAt,
				UpdatedAt:   n.UpdatedAt,
			})
		}

		if !response.Data.ProductVariant.Metafields.PageInfo.HasNextPage {
			break
		}

		vars["after"] = response.Data.ProductVariant.Metafields.PageInfo.EndCursor
	}

	if !found {
		return nil, errors.New("not found")
	}

	return metafields, nil
}

const orderMetafieldsQuery = `
query($ownerId: ID!, $first: Int!, $after: String, $namespace: String, $keys: [String!], $reverse: Boolean) {
  order(id: $ownerId) {
    id
    metafields(first: $first, after: $after, namespace: $namespace, keys: $keys, reverse: $reverse) {
      edges {
        node {
          id
          namespace
          key
          description
          value
          type
          createdAt
          updatedAt
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

type orderMetafieldsResponse struct {
	Data struct {
		Order struct {
			ID         string `json:"id"`
			Metafields struct {
				Edges []struct {
					Node struct {
						ID          string `json:"id"`
						Namespace   string `json:"namespace"`
						Key         string `json:"key"`
						Description string `json:"description"`
						Value       string `json:"value"`
						Type        string `json:"type"`
						CreatedAt   string `json:"createdAt"`
						UpdatedAt   string `json:"updatedAt"`
					} `json:"node"`
				} `json:"edges"`
				PageInfo struct {
					HasNextPage bool   `json:"hasNextPage"`
					EndCursor   string `json:"endCursor"`
				} `json:"pageInfo"`
			} `json:"metafields"`
		} `json:"order"`
	} `json:"data"`
}

// listOrderMetafields lists metafields for the given order. When the order
// doesn't exist (e.g. it was deleted or access is denied) the query returns
// null and the error is non-nil.
func listOrderMetafields(client *gql.Client, orderID int64, namespace, key string, reverse bool) ([]Metafield, error) {
	vars := map[string]interface{}{
		"ownerId": fmt.Sprintf("gid://shopify/Order/%d", orderID),
		"first":   250,
	}

	if namespace != "" {
		vars["namespace"] = namespace
	}

	if reverse {
		vars["reverse"] = true
	}

	// The GraphQL keys argument requires the namespace.key format, so a bare
	// key filter (no namespace) is applied client-side below.
	filterByKey := false
	if key != "" {
		if namespace != "" {
			vars["keys"] = []string{namespace + "." + key}
		} else {
			filterByKey = true
		}
	}

	var metafields []Metafield
	found := false

	for {
		data, err := client.Execute(orderMetafieldsQuery, vars)
		if err != nil {
			return nil, fmt.Errorf("Cannot list metafields for order: %s", err)
		}

		b, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("Cannot list metafields for order: %s", err)
		}

		var response orderMetafieldsResponse
		if err := json.Unmarshal(b, &response); err != nil {
			return nil, fmt.Errorf("Cannot list metafields for order: %s", err)
		}

		if response.Data.Order.ID != "" {
			found = true
		}

		for _, edge := range response.Data.Order.Metafields.Edges {
			n := edge.Node
			if filterByKey && n.Key != key {
				continue
			}

			metafields = append(metafields, Metafield{
				ID:          n.ID,
				Namespace:   n.Namespace,
				Key:         n.Key,
				Description: n.Description,
				Value:       n.Value,
				Type:        n.Type,
				CreatedAt:   n.CreatedAt,
				UpdatedAt:   n.UpdatedAt,
			})
		}

		if !response.Data.Order.Metafields.PageInfo.HasNextPage {
			break
		}

		vars["after"] = response.Data.Order.Metafields.PageInfo.EndCursor
	}

	if !found {
		return nil, errors.New("not found")
	}

	return metafields, nil
}

const ordersMetafieldsQuery = `
query($query: String!, $first: Int!, $namespace: String, $keys: [String!], $reverse: Boolean) {
  orders(first: $first, query: $query) {
    edges {
      node {
        id
        metafields(first: 250, namespace: $namespace, keys: $keys, reverse: $reverse) {
          edges {
            node {
              id
              namespace
              key
              description
              value
              type
              createdAt
              updatedAt
            }
          }
          pageInfo {
            hasNextPage
            endCursor
          }
        }
      }
    }
  }
}
`

type ordersMetafieldsResponse struct {
	Data struct {
		Orders struct {
			Edges []struct {
				Node struct {
					ID         string `json:"id"`
					Metafields struct {
						Edges []struct {
							Node struct {
								ID          string `json:"id"`
								Namespace   string `json:"namespace"`
								Key         string `json:"key"`
								Description string `json:"description"`
								Value       string `json:"value"`
								Type        string `json:"type"`
								CreatedAt   string `json:"createdAt"`
								UpdatedAt   string `json:"updatedAt"`
							} `json:"node"`
						} `json:"edges"`
						PageInfo struct {
							HasNextPage bool   `json:"hasNextPage"`
							EndCursor   string `json:"endCursor"`
						} `json:"pageInfo"`
					} `json:"metafields"`
				} `json:"node"`
			} `json:"edges"`
		} `json:"orders"`
	} `json:"data"`
}

// listOrdersMetafields lists metafields for the orders matching any of the
// given names and/or product SKUs, using the same search query syntax as the
// orders ls command. The orders are selected by the query and their metafields
// come back in the same response; limit caps how many orders are returned.
// Metafields are limited to the first 250 per order.
func listOrdersMetafields(client *gql.Client, names, skus []string, limit int, namespace, key string, reverse bool) ([]Metafield, error) {
	var parts []string
	for _, name := range names {
		parts = append(parts, "name:"+name)
	}
	for _, sku := range skus {
		parts = append(parts, "sku:"+sku)
	}

	vars := map[string]interface{}{
		"query": strings.Join(parts, " OR "),
		"first": limit,
	}

	if namespace != "" {
		vars["namespace"] = namespace
	}

	if reverse {
		vars["reverse"] = true
	}

	// The GraphQL keys argument requires the namespace.key format, so a bare
	// key filter (no namespace) is applied client-side below.
	filterByKey := false
	if key != "" {
		if namespace != "" {
			vars["keys"] = []string{namespace + "." + key}
		} else {
			filterByKey = true
		}
	}

	data, err := client.Execute(ordersMetafieldsQuery, vars)
	if err != nil {
		return nil, fmt.Errorf("Cannot list metafields for order: %s", err)
	}

	b, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("Cannot list metafields for order: %s", err)
	}

	var response ordersMetafieldsResponse
	if err := json.Unmarshal(b, &response); err != nil {
		return nil, fmt.Errorf("Cannot list metafields for order: %s", err)
	}

	var metafields []Metafield
	for _, edge := range response.Data.Orders.Edges {
		for _, mfEdge := range edge.Node.Metafields.Edges {
			n := mfEdge.Node
			if filterByKey && n.Key != key {
				continue
			}

			metafields = append(metafields, Metafield{
				ID:          n.ID,
				Namespace:   n.Namespace,
				Key:         n.Key,
				Description: n.Description,
				Value:       n.Value,
				Type:        n.Type,
				CreatedAt:   n.CreatedAt,
				UpdatedAt:   n.UpdatedAt,
			})
		}
	}

	return metafields, nil
}

const draftOrderMetafieldsQuery = `
query($ownerId: ID!, $first: Int!, $after: String, $namespace: String, $keys: [String!], $reverse: Boolean) {
  draftOrder(id: $ownerId) {
    id
    metafields(first: $first, after: $after, namespace: $namespace, keys: $keys, reverse: $reverse) {
      edges {
        node {
          id
          namespace
          key
          description
          value
          type
          createdAt
          updatedAt
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

type draftOrderMetafieldsResponse struct {
	Data struct {
		DraftOrder struct {
			ID         string `json:"id"`
			Metafields struct {
				Edges []struct {
					Node struct {
						ID          string `json:"id"`
						Namespace   string `json:"namespace"`
						Key         string `json:"key"`
						Description string `json:"description"`
						Value       string `json:"value"`
						Type        string `json:"type"`
						CreatedAt   string `json:"createdAt"`
						UpdatedAt   string `json:"updatedAt"`
					} `json:"node"`
				} `json:"edges"`
				PageInfo struct {
					HasNextPage bool   `json:"hasNextPage"`
					EndCursor   string `json:"endCursor"`
				} `json:"pageInfo"`
			} `json:"metafields"`
		} `json:"draftOrder"`
	} `json:"data"`
}

// listDraftOrderMetafields lists metafields for the given draft order. When
// the draft order doesn't exist (e.g. it was deleted or access is denied) the
// query returns null and the error is non-nil.
func listDraftOrderMetafields(client *gql.Client, draftOrderID int64, namespace, key string, reverse bool) ([]Metafield, error) {
	vars := map[string]interface{}{
		"ownerId": fmt.Sprintf("gid://shopify/DraftOrder/%d", draftOrderID),
		"first":   250,
	}

	if namespace != "" {
		vars["namespace"] = namespace
	}

	if reverse {
		vars["reverse"] = true
	}

	// The GraphQL keys argument requires the namespace.key format, so a bare
	// key filter (no namespace) is applied client-side below.
	filterByKey := false
	if key != "" {
		if namespace != "" {
			vars["keys"] = []string{namespace + "." + key}
		} else {
			filterByKey = true
		}
	}

	var metafields []Metafield
	found := false

	for {
		data, err := client.Execute(draftOrderMetafieldsQuery, vars)
		if err != nil {
			return nil, fmt.Errorf("Cannot list metafields for draft order: %s", err)
		}

		b, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("Cannot list metafields for draft order: %s", err)
		}

		var response draftOrderMetafieldsResponse
		if err := json.Unmarshal(b, &response); err != nil {
			return nil, fmt.Errorf("Cannot list metafields for draft order: %s", err)
		}

		if response.Data.DraftOrder.ID != "" {
			found = true
		}

		for _, edge := range response.Data.DraftOrder.Metafields.Edges {
			n := edge.Node
			if filterByKey && n.Key != key {
				continue
			}

			metafields = append(metafields, Metafield{
				ID:          n.ID,
				Namespace:   n.Namespace,
				Key:         n.Key,
				Description: n.Description,
				Value:       n.Value,
				Type:        n.Type,
				CreatedAt:   n.CreatedAt,
				UpdatedAt:   n.UpdatedAt,
			})
		}

		if !response.Data.DraftOrder.Metafields.PageInfo.HasNextPage {
			break
		}

		vars["after"] = response.Data.DraftOrder.Metafields.PageInfo.EndCursor
	}

	if !found {
		return nil, errors.New("not found")
	}

	return metafields, nil
}

const draftOrdersMetafieldsQuery = `
query($query: String!, $first: Int!, $namespace: String, $keys: [String!], $reverse: Boolean) {
  draftOrders(first: $first, query: $query) {
    edges {
      node {
        id
        metafields(first: 250, namespace: $namespace, keys: $keys, reverse: $reverse) {
          edges {
            node {
              id
              namespace
              key
              description
              value
              type
              createdAt
              updatedAt
            }
          }
          pageInfo {
            hasNextPage
            endCursor
          }
        }
      }
    }
  }
}
`

type draftOrdersMetafieldsResponse struct {
	Data struct {
		DraftOrders struct {
			Edges []struct {
				Node struct {
					ID         string `json:"id"`
					Metafields struct {
						Edges []struct {
							Node struct {
								ID          string `json:"id"`
								Namespace   string `json:"namespace"`
								Key         string `json:"key"`
								Description string `json:"description"`
								Value       string `json:"value"`
								Type        string `json:"type"`
								CreatedAt   string `json:"createdAt"`
								UpdatedAt   string `json:"updatedAt"`
							} `json:"node"`
						} `json:"edges"`
						PageInfo struct {
							HasNextPage bool   `json:"hasNextPage"`
							EndCursor   string `json:"endCursor"`
						} `json:"pageInfo"`
					} `json:"metafields"`
				} `json:"node"`
			} `json:"edges"`
		} `json:"draftOrders"`
	} `json:"data"`
}

// listDraftOrdersMetafields lists metafields for the draft orders matching any
// of the given names and/or product SKUs, using the same search query syntax
// as the draftorders ls command. The draft orders are selected by the query
// and their metafields come back in the same response; limit caps how many
// draft orders are returned. Metafields are limited to the first 250 per
// draft order.
func listDraftOrdersMetafields(client *gql.Client, names, skus []string, limit int, namespace, key string, reverse bool) ([]Metafield, error) {
	var parts []string
	for _, name := range names {
		parts = append(parts, "name:"+name)
	}
	for _, sku := range skus {
		parts = append(parts, "sku:"+sku)
	}

	vars := map[string]interface{}{
		"query": strings.Join(parts, " OR "),
		"first": limit,
	}

	if namespace != "" {
		vars["namespace"] = namespace
	}

	if reverse {
		vars["reverse"] = true
	}

	// The GraphQL keys argument requires the namespace.key format, so a bare
	// key filter (no namespace) is applied client-side below.
	filterByKey := false
	if key != "" {
		if namespace != "" {
			vars["keys"] = []string{namespace + "." + key}
		} else {
			filterByKey = true
		}
	}

	data, err := client.Execute(draftOrdersMetafieldsQuery, vars)
	if err != nil {
		return nil, fmt.Errorf("Cannot list metafields for draft order: %s", err)
	}

	b, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("Cannot list metafields for draft order: %s", err)
	}

	var response draftOrdersMetafieldsResponse
	if err := json.Unmarshal(b, &response); err != nil {
		return nil, fmt.Errorf("Cannot list metafields for draft order: %s", err)
	}

	var metafields []Metafield
	for _, edge := range response.Data.DraftOrders.Edges {
		for _, mfEdge := range edge.Node.Metafields.Edges {
			n := mfEdge.Node
			if filterByKey && n.Key != key {
				continue
			}

			metafields = append(metafields, Metafield{
				ID:          n.ID,
				Namespace:   n.Namespace,
				Key:         n.Key,
				Description: n.Description,
				Value:       n.Value,
				Type:        n.Type,
				CreatedAt:   n.CreatedAt,
				UpdatedAt:   n.UpdatedAt,
			})
		}
	}

	return metafields, nil
}

const shopMetafieldsQuery = `
query($first: Int!, $after: String, $namespace: String, $keys: [String!], $reverse: Boolean) {
  shop {
    id
    metafields(first: $first, after: $after, namespace: $namespace, keys: $keys, reverse: $reverse) {
      edges {
        node {
          id
          namespace
          key
          description
          value
          type
          createdAt
          updatedAt
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

type shopMetafieldsResponse struct {
	Data struct {
		Shop struct {
			ID         string `json:"id"`
			Metafields struct {
				Edges []struct {
					Node struct {
						ID          string `json:"id"`
						Namespace   string `json:"namespace"`
						Key         string `json:"key"`
						Description string `json:"description"`
						Value       string `json:"value"`
						Type        string `json:"type"`
						CreatedAt   string `json:"createdAt"`
						UpdatedAt   string `json:"updatedAt"`
					} `json:"node"`
				} `json:"edges"`
				PageInfo struct {
					HasNextPage bool   `json:"hasNextPage"`
					EndCursor   string `json:"endCursor"`
				} `json:"pageInfo"`
			} `json:"metafields"`
		} `json:"shop"`
	} `json:"data"`
}

func listShopMetafields(client *gql.Client, namespace, key string, reverse bool) ([]Metafield, error) {
	vars := map[string]interface{}{
		"first": 250,
	}

	if namespace != "" {
		vars["namespace"] = namespace
	}

	if reverse {
		vars["reverse"] = true
	}

	// The GraphQL keys argument requires the namespace.key format, so a bare
	// key filter (no namespace) is applied client-side below.
	filterByKey := false
	if key != "" {
		if namespace != "" {
			vars["keys"] = []string{namespace + "." + key}
		} else {
			filterByKey = true
		}
	}

	var metafields []Metafield

	for {
		data, err := client.Execute(shopMetafieldsQuery, vars)
		if err != nil {
			return nil, fmt.Errorf("Cannot list metafields for shop: %s", err)
		}

		b, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("Cannot list metafields for shop: %s", err)
		}

		var response shopMetafieldsResponse
		if err := json.Unmarshal(b, &response); err != nil {
			return nil, fmt.Errorf("Cannot list metafields for shop: %s", err)
		}

		for _, edge := range response.Data.Shop.Metafields.Edges {
			n := edge.Node
			if filterByKey && n.Key != key {
				continue
			}

			metafields = append(metafields, Metafield{
				ID:          n.ID,
				Namespace:   n.Namespace,
				Key:         n.Key,
				Description: n.Description,
				Value:       n.Value,
				Type:        n.Type,
				CreatedAt:   n.CreatedAt,
				UpdatedAt:   n.UpdatedAt,
			})
		}

		if !response.Data.Shop.Metafields.PageInfo.HasNextPage {
			break
		}

		vars["after"] = response.Data.Shop.Metafields.PageInfo.EndCursor
	}

	return metafields, nil
}

const metafieldsDeleteMutation = `
mutation metafieldsDelete($metafields: [MetafieldIdentifierInput!]!) {
  metafieldsDelete(metafields: $metafields) {
    deletedMetafields {
      key
      namespace
      ownerId
    }
    userErrors {
      field
      message
    }
  }
}
`

var fieldIndexRe = regexp.MustCompile(`\.(\d+)`)

type metafieldInput struct {
	OwnerID   string
	Namespace string
	Key       string
}

type DeletedMetafield struct {
	Key       string
	Namespace string
	OwnerID   string
	Error     string
}

// indexFromField extracts the numeric input index from a userError field path.
// The field value is a []interface{} of path segments (e.g. ["metafields", "0", "key"]);
// joined with "." that becomes "metafields.0.key" and we match the first ".N" segment.
func indexFromField(field interface{}) (int, bool) {
	items, ok := field.([]interface{})
	if !ok {
		return 0, false
	}

	parts := make([]string, len(items))
	for i, p := range items {
		parts[i] = fmt.Sprint(p)
	}

	match := fieldIndexRe.FindStringSubmatch(strings.Join(parts, "."))
	if match == nil {
		return 0, false
	}

	idx, err := strconv.Atoi(match[1])
	if err != nil {
		return 0, false
	}

	return idx, true
}

func deleteMetafields(shop, token string, metafields []metafieldInput) ([]DeletedMetafield, error) {
	inputs := make([]map[string]interface{}, len(metafields))
	for i, mf := range metafields {
		inputs[i] = map[string]interface{}{
			"ownerId":   mf.OwnerID,
			"namespace": mf.Namespace,
			"key":       mf.Key,
		}
	}

	client := gql.NewClient(shop, token)

	data, err := client.Execute(metafieldsDeleteMutation, map[string]interface{}{
		"metafields": inputs,
	})
	if err != nil {
		return nil, fmt.Errorf("Cannot delete metafields: %s", err)
	}

	result := make([]DeletedMetafield, len(metafields))

	// Map user errors back to their input positions via the index embedded in the field path.
	erroredIndices := make(map[int]bool)
	userErrors, _ := data.ValuesForPath("data.metafieldsDelete.userErrors")
	for _, ue := range userErrors {
		ueMap := ue.(map[string]interface{})
		message := fmt.Sprint(ueMap["message"])
		idx, ok := indexFromField(ueMap["field"])
		if ok && idx < len(result) {
			erroredIndices[idx] = true
			result[idx] = DeletedMetafield{Error: message, OwnerID: metafields[idx].OwnerID, Namespace: metafields[idx].Namespace, Key: metafields[idx].Key}
		} else {
			// No index in field path: general error, apply to all non-errored slots.
			for i := range result {
				if !erroredIndices[i] {
					erroredIndices[i] = true
					result[i] = DeletedMetafield{Error: message, OwnerID: metafields[i].OwnerID, Namespace: metafields[i].Namespace, Key: metafields[i].Key}
				}
			}
		}
	}

	// deletedMetafields is ordered to match the non-errored inputs.
	var successIndices []int
	for i := range metafields {
		if !erroredIndices[i] {
			successIndices = append(successIndices, i)
		}
	}

	nodes, _ := data.ValuesForPath("data.metafieldsDelete.deletedMetafields")
	for i, node := range nodes {
		if i >= len(successIndices) {
			break
		}
		n, ok := node.(map[string]interface{})
		if !ok {
			result[successIndices[i]] = DeletedMetafield{Error: "Not found or access denied", OwnerID: metafields[successIndices[i]].OwnerID, Namespace: metafields[successIndices[i]].Namespace, Key: metafields[successIndices[i]].Key}
			continue
		}
		result[successIndices[i]] = DeletedMetafield{
			Key:       fmt.Sprint(n["key"]),
			Namespace: fmt.Sprint(n["namespace"]),
			OwnerID:   fmt.Sprint(n["ownerId"]),
		}
	}

	return result, nil
}
