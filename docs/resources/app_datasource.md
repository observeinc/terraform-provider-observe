---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "observe_app_datasource Resource - terraform-provider-observe"
subcategory: ""
description: |-
  An AppDataSource is a datasource associted with a specific app, which form the conceptual data entrypoints for the Observe resources provided by the app. Currently mainly used internally to setup new apps when the user requests an installation.
---
# observe_app_datasource

An AppDataSource is a datasource associted with a specific app, which form the conceptual data entrypoints for the Observe resources provided by the app. Currently mainly used internally to setup new apps when the user requests an installation.

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `app` (String) OID of the app to attach the datasource to
- `instructions` (String) Instructions on how to connect the app to the data producer usding the datsource.
- `name` (String) Name of the AppDataSource.
- `source_url` (String) Terraform SourceUrl for the datasource.
- `variables` (Map of String) Input options to use while creating the datasource.

### Optional

- `description` (String) AppDataSource description.

### Read-Only

- `id` (String) The ID of this resource.
- `oid` (String) OID (Observe ID) for this object. This is the canonical identifier that
should be used when referring to this object in terraform manifests.

