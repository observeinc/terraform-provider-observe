---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "observe_resource_grants Resource - terraform-provider-observe"
subcategory: ""
description: |-
  NOTE: This feature is still under development. It is not meant for customer use yet.
  Authoritative. Manages the complete set of grants for a given resource.
---
# observe_resource_grants

NOTE: This feature is still under development. It is not meant for customer use yet.

Authoritative. Manages the complete set of grants for a given resource.
## Example Usage
```terraform
data "observe_workspace" "default" {
  name = "Default"
}

data "observe_rbac_group" "example" {
  name = "engineering"
}

data "observe_rbac_group" "everyone" {
  name = "Everyone"
}

data "observe_dataset" "example" {
  workspace = data.observe_workspace.default.oid
  name      = "Engineering Logs"
}

// Allow group engineering to edit and Everyone to view dataset Engineering Logs.
// Ensures there are no other statements targeting this dataset,
// so no one else (except admins) can view or edit it.
resource "observe_resource_grants" "example" {
  oid = data.observe_dataset.example.oid
  grant {
    subject = data.observe_rbac_group.example.oid
    role    = "dataset_editor"
  }
  grant {
    subject = data.observe_rbac_group.everyone.oid
    role    = "dataset_viewer"
  }
}
```
<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `oid` (String) The OID of the resource to manage grants for.

### Optional

- `grant` (Block Set) (see [below for nested schema](#nestedblock--grant))

### Read-Only

- `id` (String) The ID of this resource.

<a id="nestedblock--grant"></a>
### Nested Schema for `grant`

Required:

- `role` (String) The role to grant.
- `subject` (String) OID of the subject. Must be a user or a group.

Read-Only:

- `oid` (String)
## Import
Import is supported using the following syntax:
```shell
terraform import observe_resource_grants.example o:::dataset:41000007
```