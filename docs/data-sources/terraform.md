---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "observe_terraform Data Source - terraform-provider-observe"
subcategory: ""
description: |-
  Generates Terraform configuration for a given resource in Observe. Datasets, monitors, and dashboards are supported.
---

# observe_terraform (Data Source)

Generates Terraform configuration for a given resource in Observe. Datasets, monitors, and dashboards are supported.

## Example Usage

```terraform
data "observe_workspace" "default" {
  name = "Default"
}

data "observe_dataset" "kubernetes_container" {
  workspace = data.observe_workspace.default.oid
  name      = "kubernetes/Container"
}

data "observe_terraform" "Example" {
  target = data.observe_dataset.kubernetes_container.oid
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `target` (String) The OID of the target object, for which Terraform configuration will be generated. This can be a dataset, monitor, or dashboard.

### Read-Only

- `data_source` (String) Terraform data_source representation of the specified Observe resource.
- `id` (String) The ID of this resource.
- `import_id` (String) Observe ID that can be used with terraform import to bring the Observe resource under terraform management.
- `import_name` (String) Name of the specified resource  that can be used with terraform import to bring the Observe resource under terraform management.
- `resource` (String) Terraform resource representation of the specified Observe resource.
