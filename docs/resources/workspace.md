---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "observe_workspace Resource - terraform-provider-observe"
subcategory: ""
description: |-
  Most objects you can create on Observe - such as datasets, monitors
  and dashboards - live within workspaces. An Observe workspace provides
  isolation between usecases within an Observe account. You can use workspaces to
  segregate between different environments, teams or tenants.
  All Observe accounts have a Default workspace. This workspace cannot be deleted.
---
# observe_workspace

Most objects you can create on Observe - such as datasets, monitors
and dashboards - live within workspaces. An Observe workspace provides
isolation between usecases within an Observe account. You can use workspaces to
segregate between different environments, teams or tenants. 

All Observe accounts have a `Default` workspace. This workspace cannot be deleted.
## Example Usage
```terraform
resource "observe_workspace" "example" {
  name = "Example"
}
```
<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String) Workspace name. Must be unique for customer account.

### Read-Only

- `id` (String) Resource ID for this object.
- `oid` (String) OID (Observe ID) for this object. This is the canonical identifier that
should be used when referring to this object in terraform manifests.
## Import
Import is supported using the following syntax:
```shell
terraform import observe_workspace.example 4100001
```
