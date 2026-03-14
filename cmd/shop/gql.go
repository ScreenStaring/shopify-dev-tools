package shop

import (
	"encoding/json"
	"fmt"

	"github.com/ScreenStaring/shopify-dev-tools/gql"
)

const shopQuery = `
{
  shop {
    id
    name
    shopOwnerName
    email
    contactEmail
    createdAt
    updatedAt
    myshopifyDomain
    primaryDomain {
      host
    }
    plan {
      displayName
      partnerDevelopment
      shopifyPlus
    }
    currencyCode
    weightUnit
    billingAddress {
      address1
      address2
      city
      province
      provinceCode
      country
      countryCodeV2
      zip
      phone
      latitude
      longitude
    }
    ianaTimezone
    timezoneAbbreviation
    timezoneOffset
    currencyFormats {
      moneyFormat
      moneyWithCurrencyFormat
      moneyInEmailsFormat
      moneyWithCurrencyInEmailsFormat
    }
    taxesIncluded
    taxShipping
    setupRequired
  }
}
`

const accessScopesQuery = `
{
  currentAppInstallation {
    accessScopes {
      handle
    }
  }
}
`

type ShopInfo struct {
	ID                              string
	Name                            string
	ShopOwner                       string
	Email                           string
	CustomerEmail                   string
	CreatedAt                       string
	UpdatedAt                       string
	Address1                        string
	Address2                        string
	City                            string
	Country                         string
	CountryCode                     string
	CountryName                     string
	Currency                        string
	Domain                          string
	Latitude                        float64
	Longitude                       float64
	Phone                           string
	Province                        string
	ProvinceCode                    string
	Zip                             string
	MoneyFormat                     string
	MoneyWithCurrencyFormat         string
	WeightUnit                      string
	MyshopifyDomain                 string
	PlanDisplayName                 string
	Timezone                        string
	IanaTimezone                    string
	TaxShipping                     bool
	TaxesIncluded                   bool
	SetupRequired                   bool
	MoneyInEmailsFormat             string
	MoneyWithCurrencyInEmailsFormat string
}

type shopJSON struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	ShopOwnerName string `json:"shopOwnerName"`
	Email         string `json:"email"`
	ContactEmail  string `json:"contactEmail"`
	CreatedAt     string `json:"createdAt"`
	UpdatedAt     string `json:"updatedAt"`
	MyshopifyDomain string `json:"myshopifyDomain"`
	PrimaryDomain   struct {
		Host string `json:"host"`
	} `json:"primaryDomain"`
	Plan struct {
		DisplayName string `json:"displayName"`
	} `json:"plan"`
	CurrencyCode   string `json:"currencyCode"`
	WeightUnit     string `json:"weightUnit"`
	BillingAddress struct {
		Address1      string  `json:"address1"`
		Address2      string  `json:"address2"`
		City          string  `json:"city"`
		Province      string  `json:"province"`
		ProvinceCode  string  `json:"provinceCode"`
		Country       string  `json:"country"`
		CountryCodeV2 string  `json:"countryCodeV2"`
		Zip           string  `json:"zip"`
		Phone         string  `json:"phone"`
		Latitude      float64 `json:"latitude"`
		Longitude     float64 `json:"longitude"`
	} `json:"billingAddress"`
	IanaTimezone         string `json:"ianaTimezone"`
	TimezoneAbbreviation string `json:"timezoneAbbreviation"`
	TimezoneOffset       string `json:"timezoneOffset"`
	CurrencyFormats      struct {
		MoneyFormat                     string `json:"moneyFormat"`
		MoneyWithCurrencyFormat         string `json:"moneyWithCurrencyFormat"`
		MoneyInEmailsFormat             string `json:"moneyInEmailsFormat"`
		MoneyWithCurrencyInEmailsFormat string `json:"moneyWithCurrencyInEmailsFormat"`
	} `json:"currencyFormats"`
	TaxesIncluded bool `json:"taxesIncluded"`
	TaxShipping   bool `json:"taxShipping"`
	SetupRequired bool `json:"setupRequired"`
}

type shopResponse struct {
	Data struct {
		Shop shopJSON `json:"shop"`
	} `json:"data"`
}

type accessScopesResponse struct {
	Data struct {
		CurrentAppInstallation struct {
			AccessScopes []struct {
				Handle string `json:"handle"`
			} `json:"accessScopes"`
		} `json:"currentAppInstallation"`
	} `json:"data"`
}

func findShop(shop, token string) (*ShopInfo, error) {
	client := gql.NewClient(shop, token)

	data, err := client.Execute(shopQuery)
	if err != nil {
		return nil, fmt.Errorf("Cannot get shop info: %s", err)
	}

	b, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("Cannot re-encode shop response: %s", err)
	}

	var response shopResponse
	if err := json.Unmarshal(b, &response); err != nil {
		return nil, fmt.Errorf("Cannot parse shop response: %s", err)
	}

	s := response.Data.Shop
	timezone := s.TimezoneAbbreviation
	if s.TimezoneOffset != "" {
		timezone = fmt.Sprintf("(GMT%s) %s", s.TimezoneOffset, s.TimezoneAbbreviation)
	}

	return &ShopInfo{
		ID:                              s.ID,
		Name:                            s.Name,
		ShopOwner:                       s.ShopOwnerName,
		Email:                           s.Email,
		CustomerEmail:                   s.ContactEmail,
		CreatedAt:                       s.CreatedAt,
		UpdatedAt:                       s.UpdatedAt,
		Address1:                        s.BillingAddress.Address1,
		Address2:                        s.BillingAddress.Address2,
		City:                            s.BillingAddress.City,
		Country:                         s.BillingAddress.CountryCodeV2,
		CountryCode:                     s.BillingAddress.CountryCodeV2,
		CountryName:                     s.BillingAddress.Country,
		Currency:                        s.CurrencyCode,
		Domain:                          s.PrimaryDomain.Host,
		Latitude:                        s.BillingAddress.Latitude,
		Longitude:                       s.BillingAddress.Longitude,
		Phone:                           s.BillingAddress.Phone,
		Province:                        s.BillingAddress.Province,
		ProvinceCode:                    s.BillingAddress.ProvinceCode,
		Zip:                             s.BillingAddress.Zip,
		MoneyFormat:                     s.CurrencyFormats.MoneyFormat,
		MoneyWithCurrencyFormat:         s.CurrencyFormats.MoneyWithCurrencyFormat,
		WeightUnit:                      s.WeightUnit,
		MyshopifyDomain:                 s.MyshopifyDomain,
		PlanDisplayName:                 s.Plan.DisplayName,
		Timezone:                        timezone,
		IanaTimezone:                    s.IanaTimezone,
		TaxShipping:                     s.TaxShipping,
		TaxesIncluded:                   s.TaxesIncluded,
		SetupRequired:                   s.SetupRequired,
		MoneyInEmailsFormat:             s.CurrencyFormats.MoneyInEmailsFormat,
		MoneyWithCurrencyInEmailsFormat: s.CurrencyFormats.MoneyWithCurrencyInEmailsFormat,
	}, nil
}

func findAccessScopes(shop, token string) ([]string, error) {
	client := gql.NewClient(shop, token)

	data, err := client.Execute(accessScopesQuery)
	if err != nil {
		return nil, fmt.Errorf("Cannot get access scopes: %s", err)
	}

	b, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("Cannot re-encode access scopes response: %s", err)
	}

	var response accessScopesResponse
	if err := json.Unmarshal(b, &response); err != nil {
		return nil, fmt.Errorf("Cannot parse access scopes response: %s", err)
	}

	var scopes []string
	for _, s := range response.Data.CurrentAppInstallation.AccessScopes {
		scopes = append(scopes, s.Handle)
	}

	return scopes, nil
}
