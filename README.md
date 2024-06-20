# Observe Terraform Provider

## Requirements

-	[Terraform](https://www.terraform.io/downloads.html) 1.x
-	[Go](https://golang.org/doc/install) 1.18 (to build the provider plugin)

## Building The Provider

```sh
git clone git@github.com:observeinc/terraform-provider-observe.git
```

When it comes to building you have two options:

#### `go install`

If you don't mind installing the development version of the provider globally, you can use `go install` in the provider directory which will build and link the binary into your `$GOBIN` directory.

```sh
go install
```

#### `go build`

If you would rather install the provider locally and not impact the stable version you already have installed, you can use the `~/.terraformrc` file to tell Terraform where your provider is. You do this by building the provider using Go.

```sh
go build -o terraform-provider-observe
```

And then update your `~/.terraformrc` file to point at the location
you've built it.

```
provider_installation {
dev_overrides {
  "observeinc/observe" = "path/to/terraform-provider-observe/terraform-provider-observe"
}
# For all other providers, install them directly from their origin provider
# registries as normal. If you omit this, Terraform will _only_ use
# the dev_overrides block, and so no other providers will be available.
direct {}
}
```

A caveat with this approach is that you will need to run `terraform init` whenever the provider is rebuilt. You'll also need to remember to comment it/remove it when it's not in use to avoid tripping yourself up.

## Developing the Provider

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (version 1.18+ is *required*). 

In order to run the unit tests the provider, you can simply run `make test`.

```sh
make test
```

In order to run the full suite of acceptance tests, run `make testacc`.

*Note:* Acceptance tests create real resources, and often cost money to run.

```sh
make testacc
```

To print debug logs during acceptance tests, including all HTTP requests and responses, set `TF_LOG=debug`:

```sh
TF_LOG=debug make testacc
```

## Managing Dependencies

Terraform providers use [Go modules][go modules] to manage the dependencies. To add or update a dependency, you would run the following (`v1.2.3` of `foo` is a new package we want to add):

```
go get foo@v1.2.3
go mod tidy
go mod vendor
```
