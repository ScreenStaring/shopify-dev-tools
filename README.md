# Shopify Development Tools

Command-line program to assist with the development and/or maintenance of Shopify apps and stores.

## Installation

Download the version for your platform on the [releases page](https://github.com/ScreenStaring/shopify_dev_tools/releases).
Windows, macOS/OS X, and GNU/Linux are supported.

## Usage

The CLI interface uses the executable `sdt`:

    NAME:
       sdt - Shopify Development Tools

    USAGE:
       sdt command [command options] [arguments...]

    VERSION:
       0.0.10

    COMMANDS:
       admin, a                     Open admin pages
       charges, c, ch               Do things with charges
       metafield, m, meta           Metafield utilities
       orders, o                    Information about orders
       products, p                  Do things with products
       graphql, gql                 Run a GraphQL query against the Admin API
       shop, s                      Information about the given shop
       scripttags                   ScriptTag utilities
       themes, theme, t             Theme utilities
       webhook, webhooks, hooks, w  Webhook utilities
       help, h                      Shows a list of commands or help for one command

    GLOBAL OPTIONS:
       --help, -h     show help (default: false)
       --version, -v  print the version (default: false)

## Credentials

You'll need access to the Shopify store you want to execute commands against. Also see [Environment Variables](#environment-variables).

### Access Token

If the store has your app installed you can use the credentials generated when the shop installed your app:
```
sdt COMMAND --shop shopname --access-token value
```

In this scenario you will likely need to execute the command against many shops, and having to lookup the token every
time you need it can become annoying. To simplify this process you can [specify an Access Token Command](#access-token-command).

### Key & Password

If you have access to the store via the Shopify Admin you can authenticate by
[generating private app API credentials](https://shopify.dev/tutorials/generate-api-credentials). Once obtained they can be specified as follows:
```
sdt COMMAND --shop shopname --api-key thekey --api-password thepassword
```

### Access Token Command

Instead of specifying an access token per store you can provide a custom command that can lookup the token for the given `shop`.
For example:

```
sdt COMMAND --shop shopname --access-token '<shopify-access-token.sh'
```

Note that `--access-token`'s argument begins with a `<`. This tells Shopify Development Tools to treat the remaining argument
as a command, execute it, and use the first line of its output as the shop's access token.

The access token command will be passed the shop's name, as given on the command-line.

For example, if your app used Rails `shopify-access-token.sh` may contain the following:
```sh
#!/bin/bash

shop=$1
ssh example.com 'cd /app && RAILS_ENV=production bundle exec rails r "print Shop.find_by!(:shopify_domain => ARGV[0]).token" "$shop"'
```

Furthermore, you can use the [`SHOPIFY_ACCESS_TOKEN` environment variable](#environment-variables) to reduce the required options to
just `shop`:

```
export SHOPIFY_ACCESS_TOKEN='<shopify-access-token.sh'
# ...
sdt COMMAND --shop shopname
```

### Environment Variables

You can use the following environment variables to set credentials:

- `SHOPIFY_SHOP`
- `SHOPIFY_ACCESS_TOKEN` or `SHOPIFY_API_TOKEN`
- `SHOPIFY_API_PASSWORD`
- `SHOPIFY_API_KEY`

Other environment variables:

- `SHOPIFY_PRODUCT_FIELDS` - default fields for the `products` command's `--fields` flag

## Commands

Functionality can depend the GraphQL Admin API version. By default requests do not specify an API version.
If you need a specific version specify it with the `--api-version` option.

### Metafields

Metafield utilities

    NAME:
       sdt metafield - Metafield utilities

    USAGE:
       sdt metafield command [command options] [arguments...]

    COMMANDS:
       definitions, def            Metafield definition utilities
       delete, d                   Delete one or more metafields
       customer, c                 List metafields for the given customer
       product, products, prod, p  List metafields for the given product
       shop, s                     List metafields for the given shop
       storefront, sf              Storefront API utilities
       variant, var, v             List metafields for the given variant
       help, h                     Shows a list of commands or help for one command

    OPTIONS:
       --help, -h  show help (default: false)

#### Deleting Metafields in Bulk

You can specify multiple metafields to delete on the command-line:

```
sdt metafields delete [GID@namespace.key [GID@namespace.key ...]]
```

Or via stdin, with 1 ID per line:

```
sdt metafields delete < list-of-ids.txt
```

### Charges

Do things with charges

    NAME:
       sdt charges - Do things with charges

    USAGE:
       sdt charges command [command options] [arguments...]

    COMMANDS:
       ls, l      List the shop's charges or the charges given by the specified IDs
       create, c  Create a one-time charge (application charge)
       help, h    Shows a list of commands or help for one command

    OPTIONS:
       --help, -h  show help (default: false)

### Orders

Information about orders

    NAME:
       sdt orders - Information about orders

    USAGE:
       sdt orders command [command options] [arguments...]

    COMMANDS:
       useragent, ua  Info about the web browser used to place the order
       ls             List the shop's orders or the orders given by the specified IDs
       help, h        Shows a list of commands or help for one command

    OPTIONS:
       --help, -h  show help (default: false)

### Products

Do things with products

    NAME:
       sdt products - Do things with products

    USAGE:
       sdt products command [command options] [arguments...]

    COMMANDS:
       ls, l      List some of a shop's products or the products given by the specified IDs
       import, i  Import products synchronously from a Shopify CSV file
       export, e  Export product information: identifiers, inventory counts, etc...
       bulk, b    Import prodcuts from a Shopify CSV file using the Bulk API
       help, h    Shows a list of commands or help for one command

    OPTIONS:
       --help, -h  show help (default: false)

#### Bulk Importing

You can bulk import products and their inventories in a single spreadsheet. The format is a combination of Shopify's
[product CSV format](https://help.shopify.com/en/manual/products/import-export/using-csv) and its [inventory CSV format](https://help.shopify.com/en/manual/products/inventory/setup/inventory-csv).
Only one of available and on hand inventory counts can be set at a time. Note that setting "current"/"new" not supported, and these columns do not have parenthesis.

Shopify Development Tools has 2 commands for importing products:

1. `bulk` - import products using the [Shopify Admin GraphQL Bulk API](https://shopify.dev/docs/api/usage/bulk-operations/imports)
1. `import` - synchronously import

Both operations perform an upsert, i.e., the product is created it if does not exist and updated if it does.
Use the `-i`/`--identify-by` option to specify the identifier.

A good use of `import` over `bulk` is to seed your store for automated tests.

#### Asynchronously Using the Bulk API

1. `sdt products bulk import` with the appropriate arguments. This will return an ID you can use to check the bulk operation's status
1. `sdt products bulk status ID` to check the status
1. If you'd like to cancel: `sdt products bulk cancel ID`


#### Exporting Product Identifiers

Dump Shopify product and variant IDs —along with other identifiers— to a CSV or JSON file.

##### CSV

```sh
# assuming env authentication
sdt products export ids --shop YOUR_SHOP
```

This will output `YOUR_SHOP.csv`

##### JSON

```sh
# assuming env authentication
sdt products export ids --shop YOUR_SHOP -j
```

This will output `YOUR_SHOP.json` with the products as a JSON array.

If you're cross-referencing IDs it may be useful to set the root property for the JSON object output for each product/variant.

This will output each object with the variant's SKU as the root:

```
sdt products export ids --shop YOUR_SHOP -j -r sku
```

Valid properties for the `-r`/`--json-root` option are: `product_id`, `product_title`, `barcode`, `handle`, `variant_id`, `sku`.

#### Deleting Products in Bulk

You can specify multiple product IDs to delete on the command-line:

```
sdt products delete [ID [ID ...]]
```

Or via stdin, with 1 ID per line:

```
sdt products delete < list-of-ids.txt
```

### GraphQL

Run a GraphQL query against the Admin API

    NAME:
       sdt graphql - Run a GraphQL query against the Admin API

    USAGE:
       sdt graphql [command options] [query-file.graphql]

    DESCRIPTION:
       If query-file.graphql is not given query is read from stdin

    OPTIONS:
       --verbose                      Output Shopify API request/response (default: false)
       --shop value                   Shopify domain or shop name to perform command against [$SHOPIFY_SHOP]
       --api-password value           Shopify API password [$SHOPIFY_API_PASSWORD]
       --access-token value           Shopify access token for shop [$SHOPIFY_ACCESS_TOKEN, $SHOPIFY_API_TOKEN]
       --api-key value                Shopify API key to for shop [$SHOPIFY_API_KEY]
       --api-version value, -a value  API version to use; default is a versionless call
       --variable value, -v value     GraphQL variable in the format name=value; can be specified multiple times
       --extras, -x                   Include extension information in the response (default: false)
       --help, -h                     show help (default: false)

### ScriptTags

ScriptTag utilities

    NAME:
       sdt scripttags - ScriptTag utilities

    USAGE:
       sdt scripttags command [command options] [arguments...]

    COMMANDS:
       delete, del, rm, d  Delete the given ScriptTag
       list, ls            List scripttags for the given shop
       help, h             Shows a list of commands or help for one command

    OPTIONS:
       --help, -h  show help (default: false)

### Shop

Information about the given shop

    NAME:
       sdt shop - Information about the given shop

    USAGE:
       sdt shop command [command options] [arguments...]

    COMMANDS:
       access, a  List access scopes granted to the shop's token
       info, i    Information about the shop
       help, h    Shows a list of commands or help for one command

    OPTIONS:
       --help, -h  show help (default: false)

### Shopify Admin

Open admin pages

    NAME:
       sdt admin - Open admin pages

    USAGE:
       sdt admin command [command options] [arguments...]

    COMMANDS:
       order, orders, o            Open the given order ID for editing; if no ID given open the orders page
       product, products, prod, p  Open the given product ID for editing; if no ID given open the products page
       theme, t                    Open the currently published theme or given theme ID for editing
       themes                      Open themes section of the admin (not for editing)
       help, h                     Shows a list of commands or help for one command

    OPTIONS:
       --help, -h  show help (default: false)


### Themes

    NAME:
       sdt themes - Theme utilities

    USAGE:
       sdt themes command [command options] [arguments...]

    COMMANDS:
       ls        List the shop's themes
       cp, copy  Copy files to a theme
       help, h   Shows a list of commands or help for one command

    OPTIONS:
       --help, -h  show help (default: false)

Currently `source` can only be a local file

### Webhooks

Webhooks utilities

    NAME:
       sdt webhook - Webhook utilities

    USAGE:
       sdt webhook command [command options] [arguments...]

    COMMANDS:
       create, c           Create a webhook for the given shop
       delete, del, rm, d  Delete the given webhook
       update, u           Update the given webhook
       ls                  List the shop's webhooks
       help, h             Shows a list of commands or help for one command

    OPTIONS:
       --help, -h  show help (default: false)

## See Also

- [`ShopifyAPI::GraphQL::Request`](https://github.com/ScreenStaring/shopify_api-graphql-request) - Ruby gem to Simplify GraphQL queries and mutations for Shopify Admin API. Built-in pagination, retry, error handling, and more!
- [`ShopifyAPI::GraphQL::Bulk`](https://github.com/ScreenStaring/shopify_api-graphql-bulk) - Ruby gem to bulk import data with the Shopify GraphQL Admin Bulk API

## License

Released under the MIT License: http://www.opensource.org/licenses/MIT

---

Made by [ScreenStaring](http://screenstaring.com)
