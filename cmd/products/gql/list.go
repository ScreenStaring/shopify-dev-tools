package gql

import (
	"encoding/json"
	"fmt"
	"strings"

	gqlclient "github.com/ScreenStaring/shopify-dev-tools/gql"
)

const productsCountQuery = `
query($query: String) {
  productsCount(query: $query) {
    count
  }
}
`

type productsCountResponse struct {
	Data struct {
		ProductsCount struct {
			Count int `json:"count"`
		} `json:"productsCount"`
	} `json:"data"`
}

func FetchProductCount(shop, token, status string, options map[string]interface{}) (int, error) {
	client := gqlclient.NewClient(shop, token, options)

	vars := map[string]interface{}{}
	if len(status) > 0 {
		vars["query"] = "status:" + status
	}

	data, err := client.Execute(productsCountQuery, vars)
	if err != nil {
		return 0, fmt.Errorf("Cannot fetch product count: %s", err)
	}

	b, err := json.Marshal(data)
	if err != nil {
		return 0, fmt.Errorf("Cannot re-encode product count response: %s", err)
	}

	var response productsCountResponse
	if err := json.Unmarshal(b, &response); err != nil {
		return 0, fmt.Errorf("Cannot parse product count response: %s", err)
	}

	return response.Data.ProductsCount.Count, nil
}

const productsExportQuery = `
query($first: Int!, $after: String, $query: String) {
  products(first: $first, after: $after, query: $query) {
    pageInfo {
      hasNextPage
      endCursor
    }
    edges {
      node {
        legacyResourceId
        title
        productType
        handle
        variants(first: 250) {
          edges {
            node {
              legacyResourceId
              title
              sku
              barcode
            }
          }
        }
      }
    }
  }
}
`

type productsExportResponse struct {
	Data struct {
		Products struct {
			PageInfo struct {
				HasNextPage bool   `json:"hasNextPage"`
				EndCursor   string `json:"endCursor"`
			} `json:"pageInfo"`
			Edges []struct {
				Node struct {
					LegacyResourceId int64  `json:"legacyResourceId,string"`
					Title            string `json:"title"`
					ProductType      string `json:"productType"`
					Handle           string `json:"handle"`
					Variants         struct {
						Edges []struct {
							Node struct {
								LegacyResourceId int64  `json:"legacyResourceId,string"`
								Title            string `json:"title"`
								SKU              string `json:"sku"`
								Barcode          string `json:"barcode"`
							} `json:"node"`
						} `json:"edges"`
					} `json:"variants"`
				} `json:"node"`
			} `json:"edges"`
		} `json:"products"`
	} `json:"data"`
}

func FetchAllProducts(shop, token, status string, fn func(Product) error, options map[string]interface{}) error {
	client := gqlclient.NewClient(shop, token, options)

	vars := map[string]interface{}{"first": 250}
	if len(status) > 0 {
		vars["query"] = "status:" + status
	}

	for {
		data, err := client.Execute(productsExportQuery, vars)
		if err != nil {
			return fmt.Errorf("Cannot fetch products: %s", err)
		}

		b, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("Cannot re-encode products response: %s", err)
		}

		var response productsExportResponse
		if err := json.Unmarshal(b, &response); err != nil {
			return fmt.Errorf("Cannot parse products response: %s", err)
		}

		for _, edge := range response.Data.Products.Edges {
			n := edge.Node
			product := Product{
				ID:          n.LegacyResourceId,
				Title:       n.Title,
				ProductType: n.ProductType,
				Handle:      n.Handle,
			}

			for _, vEdge := range n.Variants.Edges {
				v := vEdge.Node
				product.Variants = append(product.Variants, Variant{
					ID:      v.LegacyResourceId,
					Title:   v.Title,
					SKU:     v.SKU,
					Barcode: v.Barcode,
				})
			}

			if err := fn(product); err != nil {
				return err
			}
		}

		if !response.Data.Products.PageInfo.HasNextPage {
			break
		}

		vars["after"] = response.Data.Products.PageInfo.EndCursor
	}

	return nil
}

const productsQuery = `
query($first: Int!, $query: String) {
  products(first: $first, query: $query) {
    edges {
      node {
        legacyResourceId
        title
        descriptionHtml
        vendor
        productType
        handle
        createdAt
        updatedAt
        publishedAt
        tags
        status
        templateSuffix
        options {
          name
          position
          values
        }
        variants(first: 250) {
          edges {
            node {
              legacyResourceId
              title
              sku
              barcode
              price
              compareAtPrice
              position
              inventoryQuantity
            }
          }
        }
      }
    }
  }
}
`

type Product struct {
	ID             int64           `json:"id,omitempty"`
	Title          string          `json:"title,omitempty"`
	BodyHTML        string          `json:"body_html,omitempty"`
	Vendor         string          `json:"vendor,omitempty"`
	ProductType    string          `json:"product_type,omitempty"`
	Handle         string          `json:"handle,omitempty"`
	CreatedAt      string          `json:"created_at,omitempty"`
	UpdatedAt      string          `json:"updated_at,omitempty"`
	PublishedAt    string          `json:"published_at,omitempty"`
	Tags           string          `json:"tags,omitempty"`
	Status         string          `json:"status,omitempty"`
	Options        []ProductOption `json:"options,omitempty"`
	Variants       []Variant       `json:"variants,omitempty"`
	TemplateSuffix string          `json:"template_suffix,omitempty"`
}

type ProductOption struct {
	Name     string   `json:"name,omitempty"`
	Position int      `json:"position,omitempty"`
	Values   []string `json:"values,omitempty"`
}

type Variant struct {
	ID                int64  `json:"id,omitempty"`
	Title             string `json:"title,omitempty"`
	SKU               string `json:"sku,omitempty"`
	Barcode           string `json:"barcode,omitempty"`
	Price             string `json:"price,omitempty"`
	CompareAtPrice    string `json:"compare_at_price,omitempty"`
	Position          int    `json:"position,omitempty"`
	InventoryQuantity int    `json:"inventory_quantity,omitempty"`
}

// JSON structs for parsing GraphQL response

type variantJSON struct {
	LegacyResourceId  int64  `json:"legacyResourceId,string"`
	Title             string `json:"title"`
	SKU               string `json:"sku"`
	Barcode           string `json:"barcode"`
	Price             string `json:"price"`
	CompareAtPrice    string `json:"compareAtPrice"`
	Position          int    `json:"position"`
	InventoryQuantity int    `json:"inventoryQuantity"`
}

type productJSON struct {
	LegacyResourceId int64    `json:"legacyResourceId,string"`
	Title            string   `json:"title"`
	DescriptionHTML  string   `json:"descriptionHtml"`
	Vendor           string   `json:"vendor"`
	ProductType      string   `json:"productType"`
	Handle           string   `json:"handle"`
	CreatedAt        string   `json:"createdAt"`
	UpdatedAt        string   `json:"updatedAt"`
	PublishedAt      string   `json:"publishedAt"`
	Tags             []string `json:"tags"`
	Status           string   `json:"status"`
	TemplateSuffix   string   `json:"templateSuffix"`
	Options          []struct {
		Name     string   `json:"name"`
		Position int      `json:"position"`
		Values   []string `json:"values"`
	} `json:"options"`
	Variants struct {
		Edges []struct {
			Node variantJSON `json:"node"`
		} `json:"edges"`
	} `json:"variants"`
}

type productsResponse struct {
	Data struct {
		Products struct {
			Edges []struct {
				Node productJSON `json:"node"`
			} `json:"edges"`
		} `json:"products"`
	} `json:"data"`
}

func buildQuery(ids []int64, status string) (string, int) {
	if len(ids) > 0 {
		parts := make([]string, len(ids))
		for i, id := range ids {
			parts[i] = fmt.Sprintf("id:%d", id)
		}
		return strings.Join(parts, " OR "), len(ids)
	}

	if len(status) > 0 {
		return "status:" + status, 0
	}

	return "", 0
}

const productsInventoryQuery = `
query($first: Int!, $after: String) {
  products(first: $first, after: $after) {
    pageInfo {
      hasNextPage
      endCursor
    }
    edges {
      node {
        legacyResourceId
        title
        variants(first: 250) {
          edges {
            node {
              legacyResourceId
              title
              sku
              barcode
              inventoryItem {
                inventoryLevels(first: 20) {
                  edges {
                    node {
                      location {
                        name
                      }
                      quantities(names: ["available", "on_hand"]) {
                        name
                        quantity
                      }
                    }
                  }
                }
              }
            }
          }
        }
      }
    }
  }
}
`

type InventoryLevel struct {
	Location  string
	Available int
	OnHand    int
}

type VariantInventory struct {
	VariantID       int64
	VariantTitle    string
	SKU             string
	Barcode         string
	InventoryLevels []InventoryLevel
}

type ProductInventory struct {
	ProductID    int64
	ProductTitle string
	Variants     []VariantInventory
}

type productsInventoryResponse struct {
	Data struct {
		Products struct {
			PageInfo struct {
				HasNextPage bool   `json:"hasNextPage"`
				EndCursor   string `json:"endCursor"`
			} `json:"pageInfo"`
			Edges []struct {
				Node struct {
					LegacyResourceId int64  `json:"legacyResourceId,string"`
					Title            string `json:"title"`
					Variants         struct {
						Edges []struct {
							Node struct {
								LegacyResourceId int64  `json:"legacyResourceId,string"`
								Title            string `json:"title"`
								SKU              string `json:"sku"`
								Barcode          string `json:"barcode"`
								InventoryItem    struct {
									InventoryLevels struct {
										Edges []struct {
											Node struct {
												Location struct {
													Name string `json:"name"`
												} `json:"location"`
												Quantities []struct {
													Name     string `json:"name"`
													Quantity int    `json:"quantity"`
												} `json:"quantities"`
											} `json:"node"`
										} `json:"edges"`
									} `json:"inventoryLevels"`
								} `json:"inventoryItem"`
							} `json:"node"`
						} `json:"edges"`
					} `json:"variants"`
				} `json:"node"`
			} `json:"edges"`
		} `json:"products"`
	} `json:"data"`
}

func FetchAllInventory(shop, token string, fn func(ProductInventory) error, options map[string]interface{}) error {
	client := gqlclient.NewClient(shop, token, options)

	vars := map[string]interface{}{"first": 10}

	for {
		data, err := client.Execute(productsInventoryQuery, vars)
		if err != nil {
			return fmt.Errorf("Cannot fetch products: %s", err)
		}

		b, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("Cannot re-encode products response: %s", err)
		}

		var response productsInventoryResponse
		if err := json.Unmarshal(b, &response); err != nil {
			return fmt.Errorf("Cannot parse products response: %s", err)
		}

		for _, edge := range response.Data.Products.Edges {
			n := edge.Node
			pi := ProductInventory{
				ProductID:    n.LegacyResourceId,
				ProductTitle: n.Title,
			}

			for _, vEdge := range n.Variants.Edges {
				v := vEdge.Node
				vi := VariantInventory{
					VariantID:    v.LegacyResourceId,
					VariantTitle: v.Title,
					SKU:          v.SKU,
					Barcode:      v.Barcode,
				}

				for _, levelEdge := range v.InventoryItem.InventoryLevels.Edges {
					level := levelEdge.Node
					il := InventoryLevel{Location: level.Location.Name}
					for _, q := range level.Quantities {
						switch q.Name {
						case "available":
							il.Available = q.Quantity
						case "on_hand":
							il.OnHand = q.Quantity
						}
					}
					vi.InventoryLevels = append(vi.InventoryLevels, il)
				}

				pi.Variants = append(pi.Variants, vi)
			}

			if err := fn(pi); err != nil {
				return err
			}
		}

		if !response.Data.Products.PageInfo.HasNextPage {
			break
		}

		vars["after"] = response.Data.Products.PageInfo.EndCursor
	}

	return nil
}

const variantsInventoryQuery = `
query($first: Int!, $after: String, $query: String!) {
  productVariants(first: $first, after: $after, query: $query) {
    pageInfo {
      hasNextPage
      endCursor
    }
    edges {
      node {
        legacyResourceId
        title
        sku
        barcode
        product {
          legacyResourceId
          title
        }
        inventoryItem {
          inventoryLevels(first: 20) {
            edges {
              node {
                location {
                  name
                }
                quantities(names: ["available", "on_hand"]) {
                  name
                  quantity
                }
              }
            }
          }
        }
      }
    }
  }
}
`

type variantsInventoryResponse struct {
	Data struct {
		ProductVariants struct {
			PageInfo struct {
				HasNextPage bool   `json:"hasNextPage"`
				EndCursor   string `json:"endCursor"`
			} `json:"pageInfo"`
			Edges []struct {
				Node struct {
					LegacyResourceId int64  `json:"legacyResourceId,string"`
					Title            string `json:"title"`
					SKU              string `json:"sku"`
					Barcode          string `json:"barcode"`
					Product          struct {
						LegacyResourceId int64  `json:"legacyResourceId,string"`
						Title            string `json:"title"`
					} `json:"product"`
					InventoryItem struct {
						InventoryLevels struct {
							Edges []struct {
								Node struct {
									Location struct {
										Name string `json:"name"`
									} `json:"location"`
									Quantities []struct {
										Name     string `json:"name"`
										Quantity int    `json:"quantity"`
									} `json:"quantities"`
								} `json:"node"`
							} `json:"edges"`
						} `json:"inventoryLevels"`
					} `json:"inventoryItem"`
				} `json:"node"`
			} `json:"edges"`
		} `json:"productVariants"`
	} `json:"data"`
}

func FetchInventoryByIdentifiers(shop, token, identifyBy string, identifiers []string, fn func(ProductInventory) error, options map[string]interface{}) error {
	client := gqlclient.NewClient(shop, token, options)

	parts := make([]string, len(identifiers))
	for i, id := range identifiers {
		parts[i] = identifyBy + ":\"" + id + "\""
	}

	vars := map[string]interface{}{
		"first": 50,
		"query": strings.Join(parts, " OR "),
	}

	for {
		data, err := client.Execute(variantsInventoryQuery, vars)
		if err != nil {
			return fmt.Errorf("Cannot fetch variants: %s", err)
		}

		b, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("Cannot re-encode variants response: %s", err)
		}

		var response variantsInventoryResponse
		if err := json.Unmarshal(b, &response); err != nil {
			return fmt.Errorf("Cannot parse variants response: %s", err)
		}

		for _, edge := range response.Data.ProductVariants.Edges {
			v := edge.Node
			vi := VariantInventory{
				VariantID:    v.LegacyResourceId,
				VariantTitle: v.Title,
				SKU:          v.SKU,
				Barcode:      v.Barcode,
			}

			for _, levelEdge := range v.InventoryItem.InventoryLevels.Edges {
				level := levelEdge.Node
				il := InventoryLevel{Location: level.Location.Name}
				for _, q := range level.Quantities {
					switch q.Name {
					case "available":
						il.Available = q.Quantity
					case "on_hand":
						il.OnHand = q.Quantity
					}
				}
				vi.InventoryLevels = append(vi.InventoryLevels, il)
			}

			pi := ProductInventory{
				ProductID:    v.Product.LegacyResourceId,
				ProductTitle: v.Product.Title,
				Variants:     []VariantInventory{vi},
			}

			if err := fn(pi); err != nil {
				return err
			}
		}

		if !response.Data.ProductVariants.PageInfo.HasNextPage {
			break
		}

		vars["after"] = response.Data.ProductVariants.PageInfo.EndCursor
	}

	return nil
}

const locationsQuery = `
{
  locations(first: 250, includeLegacy: false, includeInactive: false) {
    edges {
      node {
        id
        name
      }
    }
  }
}
`

type locationsResponse struct {
	Data struct {
		Locations struct {
			Edges []struct {
				Node struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"node"`
			} `json:"edges"`
		} `json:"locations"`
	} `json:"data"`
}

// FetchLocations returns a map of location name to GID for all active locations.
func FetchLocations(shop, token string, options map[string]interface{}) (map[string]string, error) {
	client := gqlclient.NewClient(shop, token, options)

	data, err := client.Execute(locationsQuery)
	if err != nil {
		return nil, fmt.Errorf("Cannot fetch locations: %s", err)
	}

	b, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("Cannot re-encode locations response: %s", err)
	}

	var response locationsResponse
	if err := json.Unmarshal(b, &response); err != nil {
		return nil, fmt.Errorf("Cannot parse locations response: %s", err)
	}

	locations := make(map[string]string)
	for _, edge := range response.Data.Locations.Edges {
		locations[edge.Node.Name] = edge.Node.ID
	}

	return locations, nil
}

func FetchProducts(shop, token string, ids []int64, status string, limit int, options map[string]interface{}) ([]Product, error) {
	client := gqlclient.NewClient(shop, token, options)

	query, first := buildQuery(ids, status)
	if first == 0 {
		first = limit
	}

	vars := map[string]interface{}{"first": first}
	if len(query) > 0 {
		vars["query"] = query
	}

	data, err := client.Execute(productsQuery, vars)
	if err != nil {
		return nil, fmt.Errorf("Cannot list products: %s", err)
	}

	b, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("Cannot re-encode products response: %s", err)
	}

	var response productsResponse
	if err := json.Unmarshal(b, &response); err != nil {
		return nil, fmt.Errorf("Cannot parse products response: %s", err)
	}

	var result []Product
	for _, edge := range response.Data.Products.Edges {
		n := edge.Node

		product := Product{
			ID:             n.LegacyResourceId,
			Title:          n.Title,
			BodyHTML:        n.DescriptionHTML,
			Vendor:         n.Vendor,
			ProductType:    n.ProductType,
			Handle:         n.Handle,
			CreatedAt:      n.CreatedAt,
			UpdatedAt:      n.UpdatedAt,
			PublishedAt:    n.PublishedAt,
			Tags:           strings.Join(n.Tags, ", "),
			Status:         n.Status,
			TemplateSuffix: n.TemplateSuffix,
		}

		for _, opt := range n.Options {
			product.Options = append(product.Options, ProductOption{
				Name:     opt.Name,
				Position: opt.Position,
				Values:   opt.Values,
			})
		}

		for _, vEdge := range n.Variants.Edges {
			v := vEdge.Node
			product.Variants = append(product.Variants, Variant{
				ID:                v.LegacyResourceId,
				Title:             v.Title,
				SKU:               v.SKU,
				Barcode:           v.Barcode,
				Price:             v.Price,
				CompareAtPrice:    v.CompareAtPrice,
				Position:          v.Position,
				InventoryQuantity: v.InventoryQuantity,
			})
		}

		result = append(result, product)
	}

	return result, nil
}
