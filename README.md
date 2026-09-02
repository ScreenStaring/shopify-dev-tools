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
       sdt [global options] command [command options] [arguments...]

    VERSION:
       0.1.2

    COMMANDS:
       admin, a                     Open admin pages
       charges, c, ch               Do things with charges
       collections, col             Do things with collections
       draftorders, do              Information about draft orders
       inventory, inv               Do things with inventory
       locations, loc               Do things with locations
       metafield, m, meta           Metafield utilities
       metaobjects, mo              Metaobject utilities
       orders, o                    Information about orders
       products, p                  Do things with products
       graphql, gql                 Run a GraphQL query against the Admin API
       shop, s                      Information about the given shop
       customers, cust              Do things with customers
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
sdt COMMAND --api-key thekey --api-password thepassword
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
- `SDT_MAX_RETRY_ATTEMPTS` - default connection retry attempts; defaults to `10`
- `SDT_READONLY` - [Read-only mode](#read-only-mode)

## Commands

Functionality can depend the GraphQL Admin API version. By default requests do not specify an API version.
If you need a specific version specify it with the `--api-version` option.

### Metaobjects

    NAME:
       sdt metaobjects - Metaobject utilities

    USAGE:
       sdt metaobjects command [command options] [arguments...]

    COMMANDS:
       ls, l      List metaobjects of the given type
       export, x  Export metaobjects of the given type to CSV or JSONL
       def, d     Metaobject definition utilities
       help, h    Shows a list of commands or help for one command

    OPTIONS:
       --help, -h  show help (default: false)


#### Exporting Metaobject Values

Metaobject values can be exported to CSV or JSONL. By default they're exported to CSV. Use the `-j`/`--jsonl` option to export to JSON:

```
sdt metaobjects export TYPE
```

Where `TYPE` is the metaobject type you want to export values for.

The resulting export will be output to a file in the current directory as `SHOP-TYPE.csv` or `.json`:

Values can be filtered via the `-q`/`--query` option. For example, given the `users` type with a field of `country`, you can export
users with a country value of `"Mexico"`via:

```
sdt -q 'fields.country:Mexico' users
```

Note that the field must be configured as searchable in Shopify.

For more info see [Shopify's documentation](https://shopify.dev/docs/apps/build/metafields/query-using-metafields) on querying metafields.

### Metafields

    NAME:
       sdt metafield - Metafield utilities

    USAGE:
       sdt metafield command [command options] [arguments...]

    COMMANDS:
       definitions, def            Metafield definition utilities
       delete, d                   Delete one or more metafields
       app                         List metafields for the app installation associated with the credentials
       customer, c                 List metafields for the given customer
       draftorders, draftorder, do  List metafields for the draft orders matching the given IDs, 'name:VALUE' and/or 'sku:VALUE' arguments
       orders, order, o            List metafields for the orders matching the given IDs, 'name:VALUE' and/or 'sku:VALUE' arguments
       product, products, prod, p  List metafields for the products matching the given IDs and/or 'sku:VALUE' arguments
       shop, s                     List metafields for the given shop
       storefront, sf              Storefront API utilities
       variant, var, v             List metafields for the variants matching the given IDs and/or 'sku:VALUE' arguments
       help, h                     Shows a list of commands or help for one command

    OPTIONS:
       --help, -h  show help (default: false)

#### Exporting Metafields to JSON

Use the `-j`/`--jsonl` option to export the given metafields to JSONL. For example, to export variant metafields for products with the given IDs and SKUs:

```
sdt metafields products -j 123123 sku:LP-SMALL > metafields.jsonl
```

Note that SKUs must be prefixed with `sku:`

To export metafields for all products see [Exporting Product Metafields](#exporting-product-metafields).

#### Deleting Metafields in Bulk

You can specify multiple metafields to delete on the command-line:

```
sdt metafields delete [GID@namespace.key [GID@namespace.key ...]]
```

Or via stdin, with 1 ID per line:

```
sdt metafields delete < list-of-ids.txt
```

#### Creating Metafield Definitions in Bulk

Create metafield definitions from a CSV file:

```
sdt metafield definitions import FILE.csv
```

If `FILE` is not given the CSV is read from stdin. Supported columns:

| Column | Description |
| ------ | ----------- |
| `Owner Type` | Required. Owner type, e.g. `PRODUCT`, `SHOP`, `CUSTOMER` |
| `Name` | Required. Human-readable name |
| `Key` | Required. Unique identifier within the namespace |
| `Type` | Required. [Metafield type](https://shopify.dev/docs/apps/build/metafields/list-of-data-types), e.g. `single_line_text_field` |
| `Namespace` | Defaults to the app-reserved namespace if empty |
| `Description` | Optional |
| `Pin` | `true`/`false`, default `false` |
| `Access Admin` | [MetafieldAdminAccessInput](https://shopify.dev/docs/api/admin-graphql/latest/enums/MetafieldAdminAccessInput); empty means the API default |
| `Access Customer Account` | [MetafieldCustomerAccountAccessInput](https://shopify.dev/docs/api/admin-graphql/latest/enums/MetafieldCustomerAccountAccessInput) |
| `Access Storefront` | [MetafieldStorefrontAccessInput](https://shopify.dev/docs/api/admin-graphql/latest/enums/MetafieldStorefrontAccessInput) |
| `Capability Admin Filterable` / `Capability Smart Collection Condition` / `Capability Unique Values` | `true`/`false`, default `false`. `smart_collection_condition` cannot be combined with constraints |
| `Constraint Key` | Constraint subtype key, e.g. `category`; requires `Constraint Value` |
| `Constraint Value` | Constraint values (GIDs), `\|`-separated |
| `Validation Name` / `Validation Value` | Validations per the [validation specs](https://shopify.dev/docs/apps/build/metafields/list-of-validation-options). One validation per column pair. To specify multiple validations add multiple  `Validation Name` / `Validation Value` columns. |

Column names are matched case-insensitively. Each row is one definition.

### Charges

Do things with app and onetime charges

    NAME:
       sdt charges - Do things with charges

    USAGE:
       sdt charges command [command options] [arguments...]

    COMMANDS:
       ls, l      List the shop's charges or the charges given by the specified IDs (bare ids are one time charges unless -r given)
       create, c  Create a charge (one-time by default; use -i to create a recurring charge)
       cancel     Cancel recurring charges (app subscriptions) by ID
       help, h    Shows a list of commands or help for one command

    OPTIONS:
       --help, -h  show help (default: false)

### Collections

Do things with collections

    NAME:
       sdt collections - Do things with collections

    USAGE:
       sdt collections command [command options] [arguments...]

    COMMANDS:
       ls, l    List the shop's collections or a collection given by ID
       help, h  Shows a list of commands or help for one command

    OPTIONS:
       --help, -h  show help (default: false)


### Inventory

Do things with inventory and inventory items

    NAME:
       sdt inventory - Do things with inventory

    USAGE:
       sdt inventory command [command options] [arguments...]

    COMMANDS:
       items, i     Look up the variants and products for the given inventory item IDs
       export, x    Export inventory quantities by variant and location to a CSV file
       help, h      Shows a list of commands or help for one command

    OPTIONS:
       --help, -h  show help (default: false)

IDs can be given on the command-line or read from stdin, 1 per line. Use the `-j`/`--jsonl` option to output the results in JSONL format.

### Locations

Do things with locations

    NAME:
       sdt locations - Do things with locations

    USAGE:
       sdt locations command [command options] [arguments...]

    COMMANDS:
       ls, l    List the shop's locations or the locations given by the specified IDs
       help, h  Shows a list of commands or help for one command

    OPTIONS:
       --help, -h  show help (default: false)


### Customers

Do things with customers

    NAME:
       sdt customers - Do things with customers

    USAGE:
       sdt customers command [command options] [arguments...]

    COMMANDS:
       ls, l          List the shop's customers or a customer given by ID
       segments, seg  Customer segment commands
       help, h        Shows a list of commands or help for one command

    OPTIONS:
       --help, -h  show help (default: false)

### Orders

Information about orders

    NAME:
       sdt orders - Information about orders

    USAGE:
       sdt orders command [command options] [arguments...]

    COMMANDS:
       fulfillmentorders, fo  Fulfillment order commands for an order
       fulfillments, f        Fulfillment commands for an order
       attributes, attr       Do things with an order's attributes
       ls                     List the shop's orders or the orders matching the given IDs and/or 'sku:VALUE' arguments
       help, h                Shows a list of commands or help for one command

    OPTIONS:
       --help, -h  show help (default: false)

#### Listing Orders

`sdt orders ls` lists open orders by default. Use the `-s`/`--status` option to filter by status and the `--sort` option to change the sort order (lowercase GraphQL sort enum values are accepted).
The maximum number of orders returned is 250, set with `-l`/`--limit`.

Orders can be looked up by their IDs or by the SKU of an included line item:

```
sdt orders ls --shop YOUR_SHOP sku:ABC123
```

The `sku:` prefix is matched case-insensitively.

#### Marking a Shipment as Delivered

You can mark a shipment as delivered using the `orders fulfillments delivered` command.
This requires the ID of the fulfillment to mark as delivered.

##### 1. Find the ID of the fulfillment

```
sdt orders fulfillments ls --shop YOUR_SHOP ORDER_ID
```

Here `ORDER_ID` is the numeric order ID or the Shopify GID.

This will list all fulfillments for the order, including its ID.

##### 2. Mark the fulfillment as shipped

Once you have the fulfillment ID from step 1:

```
sdt orders fulfillments delivered --shop YOUR_SHOP FULFILLMENT_ID
```

The fulfillment is now marked as delivered. If you want to add a message and/or set the event's time:

```
sdt orders fulfillments delivered --shop YOUR_SHOP -d '2026-02-14T02:30' FULFILLMENT_ID 'Your message goes here'
```

### Draft Orders

Information about draft orders

    NAME:
       sdt draftorders - Information about draft orders

    USAGE:
       sdt draftorders command [command options] [arguments...]

    COMMANDS:
       ls, l    List the shop's draft orders or the draft orders matching the given IDs and/or 'sku:VALUE' arguments
       help, h  Shows a list of commands or help for one command

    OPTIONS:
       --help, -h  show help (default: false)

#### Listing Draft Orders

`sdt draftorders ls` accepts the same arguments as [`orders ls`](#listing-orders).

### Products

Do things with products

    NAME:
       sdt products - Do things with products

    USAGE:
       sdt products command [command options] [arguments...]

    COMMANDS:
       ls, l         List some of a shop's products or the products matching the given IDs and/or 'sku:VALUE' arguments
       delete, d     Delete products by ID
       import, i     Import products synchronously from a Shopify CSV file
       export, e, x  Export product data
       bulk, b       Import products from a Shopify CSV file using the Bulk API
       help, h       Shows a list of commands or help for one command

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

To output the results of the bulk import in JSON use the `-j`/`--json` option.

#### Metafields

Products and variant metafields can be imported in bulk by using the following columns:

- `Metafield Owner`: `Product` or `Variant`
- `Metafield Type`: [Shopify metafield type](https://shopify.dev/docs/apps/build/metafields/list-of-data-types)
- `Metafield Namespace`
- `Metafield Key`
- `Metafield Value`

Multiple metafields can be provided by include additional rows underneath the initial product's row.

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
sdt products export ids --shop YOUR_SHOP -j
```

This will output `YOUR_SHOP.json` with the products as a JSON array.

If you're cross-referencing IDs it may be useful to set the root property for the JSON object output for each product/variant.

This will output each object with the variant's SKU as the root:

```
sdt products export ids --shop YOUR_SHOP -j -r sku
```

Valid properties for the `-r`/`--json-root` option are: `product_id`, `product_title`, `barcode`, `handle`, `variant_id`, `sku`.

#### Exporting Inventory Quantities

`sdt products export inventory` exports inventory quantities by variant and location to a CSV file named `YOUR_SHOP-inventory.csv`.

Use the `-i`/`--identify-by` option to read identifiers from stdin and only export inventory for the matching variants. Valid values are `id`, `sku`, and `barcode`.

#### Exporting Product Metafields

`sdt products export metafields` exports product metafields to a CSV file named `YOUR_SHOP-metafields.csv`, using the same metafield columns as the import format:

```sh
sdt products export metafields --shop YOUR_SHOP
```

Use the `-n`/`--namespace` and `-k`/`--key` options to only export metafields with the given namespace and/or key.

Each row is a metafield with the product identifier columns followed by the metafields. To output the metafields as JSONL instead, use the `-j`/`--jsonl` option:

```sh
sdt products export metafields --shop YOUR_SHOP -j
```

For fine-grained exporting of product or other resource metafields [use the metafields command](#metafields).

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
       --verbose                   Output Shopify API request/response (default: false)
       --shop value                Shopify domain or shop name to perform command against [$SHOPIFY_SHOP]
       --api-password value        Shopify API password [$SHOPIFY_API_PASSWORD]
       --access-token value        Shopify access token for shop [$SHOPIFY_ACCESS_TOKEN, $SHOPIFY_API_TOKEN]
       --api-key value             Shopify API key to for shop [$SHOPIFY_API_KEY]
       --api-version value  API version to use; default is a versionless call
       --variable value, -v value  GraphQL variable in the format name=value; can be specified multiple times
       --extras, -x                Include extension information in the response (default: false)
       --help, -h                  show help (default: false)


The `-v`/`--variable` argument is used to provide GraphQL variables. To specify an array value use `[i1, i2, ... iN]` syntax. For example:

```
-v ids='["gid://shopify/Product/123", "gid://shopify/Product/456"]'
```

#### Read-Only Mode

To prevent Shopify Development Tools from executing GraphQL mutations set the `SDT_READONLY` environment variable to `"1"`.
This is useful when using that good ol' untrustworthy AI.


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
       settings, s                 Open the general settings page or settings sections
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

#### Filtering the Webhook List

`sdt webhook ls` supports filtering by topic and address via the `-t`/`--topic` and `-a`/`--address` options. Use the `-j`/`--jsonl` option to output the webhooks in JSONL format.

#### Avoiding Duplicate Webhooks

Shopify allows one to create multiple webhooks with the same topic and endpoint. You can avoid this by using the `webhook create`'s `-1` option. This will result in an error if the
webhook you're creating exists for the given topic and endpoint.

#### Deleting Webhooks

`webhook delete` accepts one or more webhook IDs and/or topics. Topics can be given in `resource/action` or `RESOURCE_ACTION` format:

```
sdt webhook delete ORDERS_CREATE 1234567 products/update 8901234
```

This deletes every webhook matching the given topics, plus the webhooks with the given IDs.

You can delete all of the shop's webhooks via the `--all` option:

```
sdt webhook delete --all
```

## See Also

- [`ShopifyAPI::GraphQL::Request`](https://github.com/ScreenStaring/shopify_api-graphql-request) - Ruby gem to Simplify GraphQL queries and mutations for Shopify Admin API. Built-in pagination, retry, error handling, and more!
- [`ShopifyAPI::GraphQL::Bulk`](https://github.com/ScreenStaring/shopify_api-graphql-bulk) - Ruby gem to bulk import data with the Shopify GraphQL Admin Bulk API

## License

Released under the MIT License: http://www.opensource.org/licenses/MIT

---

Made by [ScreenStaring](https://screenstaring.com)
