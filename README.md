# Shopify Development Tools

Command-line program to assist with the development and/or maintenance of Shopify apps and stores.

## Installation

Download the version for your platform on the [releases page](https://github.com/ScreenStaring/shopify_dev_tools/releases).
Windows, macOS/OS X, and GNU/Linux are supported.

## Usage

    NAME:
       sdt - Shopify Development Tools

    USAGE:
       sdt command [command options] [arguments...]

    VERSION:
       0.0.5

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

### Credentials

You'll need access to the Shopify store you want to execute commands against. Also see [Environment Variables](#environment-variables).

#### Access Token

If the store has your app installed you can use the credentials generated when the shop installed your app:
```
sdt COMMAND --shop shopname --access-token value
```

In this scenario you will likely need to execute the command against many shops, and having to lookup the token every
time you need it can become annoying. To simplify this process you can [specify an Access Token Command](#access-token-command).

#### Key & Password

If you have access to the store via the Shopify Admin you can authenticate by
[generating private app API credentials](https://shopify.dev/tutorials/generate-api-credentials). Once obtained they can be specified as follows:
```
sdt COMMAND --shop shopname --api-key thekey --api-password thepassword
```

#### Access Token Command

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

#### Environment Variables

You can use the following environment variables to set credentials:

- `SHOPIFY_SHOP`
- `SHOPIFY_ACCESS_TOKEN` or `SHOPIFY_API_TOKEN`
- `SHOPIFY_API_PASSWORD`
- `SHOPIFY_API_KEY`

Other environment variables:

- `SHOPIFY_PRODUCT_FIELDS` - default fields for the `products` command's `--fields` flag

### Commands

#### Metafields

Metafield utilities

    NAME:
       sdt metafield - Metafield utilities

    USAGE:
       sdt metafield command [command options] [arguments...]

    COMMANDS:
       customer, c                 List metafields for the given customer
       product, products, prod, p  List metafields for the given product
       shop, s                     List metafields for the given shop
       storefront, sf              Storefront API utilities
       variant, var, v             List metafields for the given variant
       help, h                     Shows a list of commands or help for one command

    OPTIONS:
       --help, -h  show help (default: false)

#### Charges

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

#### Orders

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

#### Products

Do things with products

    NAME:
       sdt products - Do things with products

    USAGE:
       sdt products command [command options] [arguments...]

    COMMANDS:
       ls, l    List some of a shop's products or the products given by the specified IDs
       help, h  Shows a list of commands or help for one command

    OPTIONS:
       --help, -h  show help (default: false)

#### GraphQL

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
       --help, -h                     show help (default: false)

#### ScriptTags

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

#### Shop

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

#### Shopify Admin

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


#### Themes

    NAME:
       sdt themes cp - Copy files to a theme

    USAGE:
       sdt themes cp [command options] themeid source [...] destination

    OPTIONS:
       --verbose             Output Shopify API request/response (default: false)
       --shop value          Shopify domain or shop name to perform command against [$SHOPIFY_SHOP]
       --api-password value  Shopify API password [$SHOPIFY_API_PASSWORD]
       --access-token value  Shopify access token for shop [$SHOPIFY_ACCESS_TOKEN, $SHOPIFY_API_TOKEN]
       --api-key value       Shopify API key to for shop [$SHOPIFY_API_KEY]
       --help, -h            show help (default: false)

Currently `source` can only be a local file

#### Webhooks

Webhooks utilities

    NAME:
       sdt webhook - Webhook utilities

    USAGE:
       sdt webhook command [command options] [arguments...]

    COMMANDS:
       create, c
       delete, del, rm, d
       ls
       help, h             Shows a list of commands or help for one command

    OPTIONS:
       --shop value          Shopify domain or shop name to perform command against [$SHOPIFY_SHOP]
       --api-password value  Shopify API password [$SHOPIFY_API_PASSWORD]
       --access-token value  Shopify access token for shop [$SHOPIFY_ACCESS_TOKEN, $SHOPIFY_API_TOKEN]
       --api-key value       Shopify API key to for shop [$SHOPIFY_API_KEY]
       --help, -h            show help (default: false)

## See Also

- [Shopify ID Export](https://github.com/ScreenStaring/shopify_id_export/) - Dump Shopify product and variant IDs —along with other identifiers— to a CSV or JSON file.

## License

Released under the MIT License: http://www.opensource.org/licenses/MIT

---

Made by [ScreenStaring](http://screenstaring.com)
