---
subcategory: ""
page_title: "Manage Layered Settings"
description: |-
    Manage Layered Settings
---

## Manage Layered Settings

This page lists examples of how to manage different Observe layered setting records within Terraform. More examples will be added over time.

### Layered Setting Definitions
Below are the layered setting definitions, including the required and optional values.
```json

    {
      "default_value": "397",
      "possible_target_scopes": ["Customer", "Datastream"],
      "setting": "DataRetention.periodDays",
      "type": "int64",
      "writable_by": ["admin"]
    }

    {
      "default_value": "null(int64)",
      "possible_target_scopes": ["Customer", "App", "Dataset"],
      "setting": "Dataset.freshnessDesired",
      "type": "int64",
      "writable_by": ["writer", "admin"]
    }

    {
      "default_value": "null(int64)",
      "possible_target_scopes": ["Customer", "App", "Monitor"],
      "setting": "Monitor.freshnessGoal",
      "type": "int64",
      "writable_by": ["writer", "admin"]
    }

    {
      "default_value": "null(timestamp)",
      "possible_target_scopes": ["User"],
      "setting": "QueryGovernor.bypassUntil",
      "type": "timestamp",
      "writable_by": ["reader", "writer", "admin"]
    }

    {
      "default_value": "0.0",
      "possible_target_scopes": ["Customer"],
      "setting": "QueryGovernor.creditsPerDay",
      "type": "float64",
      "writable_by": ["admin"]
    }

    {
      "default_value": "0.0",
      "possible_target_scopes": ["Customer", "User"],
      "setting": "QueryGovernor.userCreditsPerDay",
      "type": "float64",
      "writable_by": ["admin"]
    }

    {
      "default_value": "0.0",
      "possible_target_scopes": ["Customer", "User"],
      "setting": "QueryGovernor.userThrottledLimitCreditsPerDay",
      "type": "float64",
      "writable_by": ["admin"]
    }

    {
      "default_value": "0.0",
      "possible_target_scopes": ["Customer"],
      "setting": "TransformGovernor.creditsPerDay",
      "type": "float64",
      "writable_by": ["admin"]
    }
```

### Configure Data Retention Settings

```terraform
# layered setting for a datastream retention for a specific stream
# targets a datastream OID
resource "observe_layered_setting_record" "datastream_retention_my_first" {
  workspace   = data.observe_workspace.default.oid
  name        = "Layered Setting For Retention 30 days my first datastream"
  setting     = "DataRetention.periodDays"
  value_int64 = 30
  target      = resource.observe_datastream.my_first_datastream.oid
}
```

### Configure Query Governor Settings

```terraform
data "observe_oid" "customer" {
  id   = "124203122673"
  type = "customer"
}


# Applying a customer-wide soft limit
# Query Governor - Customer Level Throttled
# Target must be a customer OID
resource "observe_layered_setting_record" "base_tenant_credit_limit_throttled" {
  workspace     = data.observe_workspace.default.oid
  name          = "New Global Credit Limit THROTTLED"
  setting       = "QueryGovernor.throttledLimitCreditsPerDay"
  value_float64 = 100.0
  target        = data.observe_oid.customer.oid
}

# Applying hard and soft limits for all users
# Query Governor - User Level - All Users - Throttled
# Target can be a customer OID or user OID
# when you target a customer OID, it just means that all
# users inherit this limit, unless they are targeted specifically
# as a user OID
resource "observe_layered_setting_record" "all_users_credit_limit_soft" {
  workspace     = data.observe_workspace.default.oid
  name          = "All Users Query Limit THROTTLED"
  setting       = "QueryGovernor.userThrottledLimitCreditsPerDay"
  value_float64 = 50.0
  target        = data.observe_oid.customer.oid
}

# Query Governor - User Level - All Users - Hard
# Target can be a customer OID or user OID
# when you target a customer OID, it just means that all
# users inherit this limit, unless they are targeted specifically
# as a user OID
resource "observe_layered_setting_record" "all_users_credit_limit_hard" {
  workspace     = data.observe_workspace.default.oid
  name          = "All Users Query Limit HARD"
  setting       = "QueryGovernor.userCreditsPerDay"
  value_float64 = 80.0
  target        = data.observe_oid.customer.oid
}


# Applying hard and soft limits to specific users
# These override the all users settings above
# for whatever users they are set for
# User 1 Lookup
data "observe_user" "carl_chumplin" {
  email = "carlTerraformChumplin@observeinc.com"
}

# Query Governor - User Level - User 1 - Throttled
resource "observe_layered_setting_record" "base_admin_credit_limit" {
  workspace     = data.observe_workspace.default.oid
  name          = "User 1 Query Limit THROTTLED"
  setting       = "QueryGovernor.userThrottledLimitCreditsPerDay"
  value_float64 = 5.0
  target        = data.observe_user.kyle_champlin.oid
}

# Query Governor - User Level - User 1 - Hard
resource "observe_layered_setting_record" "base_admin_credit_limit_hard" {
  workspace     = data.observe_workspace.default.oid
  name          = "User 1 Query Limit HARD"
  setting       = "QueryGovernor.userCreditsPerDay"
  value_float64 = 10.0
  target        = data.observe_user.kyle_champlin.oid
}

# User 2 Lookup
data "observe_user" "carl_credit" {
  email = "carlCreditLimits@observeinc.com"
}


resource "observe_layered_setting_record" "base_admin_credit_limit_throttled" {
  workspace     = data.observe_workspace.default.oid
  name          = "User 2 Query Limit throttled"
  setting       = "QueryGovernor.userThrottledLimitCreditsPerDay"
  value_float64 = 10.0
  target        = data.observe_user.carl_credit.oid
}


# Query Governor - User Level - User 2 - Throttled
resource "observe_layered_setting_record" "base_reader_credit_limit" {
  workspace     = data.observe_workspace.default.oid
  name          = "User 2 Credit Limit HARD"
  setting       = "QueryGovernor.userCreditsPerDay"
  value_float64 = 20.0
  target        = data.observe_user.carl_credit.oid
}

# There are also global limits, that are evaluated last
# meaning if the User generic or User specific limits above are not applied
# these will kick in - think of them at the total general limit
# Transforms Governor - Customer Level Hard Limit
# Target must be a customer OID

resource "observe_layered_setting_record" "base_tenant_credit_limit_transforms" {
  workspace     = data.observe_workspace.default.oid
  name          = "New Global Credit Limit HARD Transforms"
  setting       = "TransformGovernor.creditsPerDay"
  value_float64 = 200.0
  target        = data.observe_oid.customer.oid
}

resource "observe_layered_setting_record" "base_tenant_credit_limit_query" {
  workspace     = data.observe_workspace.default.oid
  name          = "New Global Credit Limit HARD query"
  setting       = "QueryGovernor.creditsPerDay"
  value_float64 = 200.0
  target        = data.observe_oid.customer.oid
}
```
