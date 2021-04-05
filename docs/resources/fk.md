---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "observe_fk Resource - terraform-provider-observe"
subcategory: ""
description: |-
  
---
# observe_fk



<!-- schema generated by tfplugindocs -->
## Schema

### Required

- **fields** (List of String) Array of field mappings that provides a link between source and target datasets. A mapping between a `source_field` and a `target_field` is represented using a colon separated "<source_field>:<target_field>" format. If the source and target field share the same name, only "<source_field>".
- **source** (String) OID of source dataset.
- **target** (String) OID of target dataset.
- **workspace** (String) OID of workspace link is contained in.

### Optional

- **id** (String) The ID of this resource.
- **label** (String) Label describing link.
