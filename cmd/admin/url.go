package admin

import (
	"fmt"
	"net/url"
)

// In the general case we'd have a type for each URL and each function
// would return a new instance of that type which will have
// methods specific to the URL the type represents:
//
// type ProductURL struct {
// 	// Variant ...
// 	func Variant(id) *VariantURL {

// 	}
// }

type Admin struct {
	endpoint string
}

const endpoint = "https://%s.myshopify.com/admin"
const products = "/products"
const product = products + "/%d"
const orders = "/orders"
const order = orders + "/%d"
const themes = "/themes"
const theme = themes + "/%d"

func NewAdminURL(shop string) *Admin {
	return &Admin{fmt.Sprintf(endpoint, shop)}
}

func (a *Admin) buildURL(path string, q map[string]string) string {
	s := a.endpoint+path

	if len(q) > 0 {
		qs := url.Values{}

		for k, v := range(q) {
			qs.Set(k, v)
		}

		s += "?"+qs.Encode()
	}

	return s
}

func (a *Admin) Order(id int64, q map[string]string) string {
	return a.buildURL(fmt.Sprintf(order, id), q)
}

func (a *Admin) Orders(q map[string]string) string {
	return a.buildURL(orders, q)
}

func (a *Admin) Product(id int64, q map[string]string) string {
	return a.buildURL(fmt.Sprintf(product, id), q)
}

func (a *Admin) Products(q map[string]string) string {
	return a.buildURL(products, q)
}

func (a *Admin) Theme(id int64, q map[string]string) string {
	return a.buildURL(fmt.Sprintf(theme, id), q)
}

func (a *Admin) Themes(q map[string]string) string {
	return a.buildURL(themes, q)
}
