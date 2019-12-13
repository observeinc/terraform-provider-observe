# First Example

## Setting up

You will need to build the terraform provider, and update `~/.terraformrc`:

```
providers {
  observe = "${GOPATH}/bin/terraform-provider-observe"
}
```


The Observe terraform provider reads the following environment variables:

| Env Var            | Description                                  |
| ------------------ | ---------------------------------------------|
| `OBSERVE_CUSTOMER` | Customer ID                                  |
| `OBSERVE_TOKEN`    | Token (no leading customer ID                |
| `OBSERVE_DOMAIN`   | Observe domain (default: `observeinc.com`)   |


For the purposes of this terraform script, you will also need to provide a
workspace ID. You can copy paste this value from the URL when navigating our
UI.

| Env Var               | Description                                    |
| --------------------- | ---------------------------------------------- |
| `TF_VAR_workspace_id` | Workspace ID within which to create transforms |


## Running terraform

To verify your plugin has been correctly detected, run init:

```
→ terraform init

Initializing the backend...

Initializing provider plugins...

Terraform has been successfully initialized!
```

You can then run `terraform plan` to do a dry-run and view the diff between the
local configuration and remote infrastructure. To apply changes, run `terraform
apply`.
