data "observe_oid" "customer" {
  id   = "123403122673"
  type = "customer"
}


# Applying a customer-wide soft limit
# Query Governor - Customer Level Throttled
# Target must be a customer
resource "observe_layered_setting_record" "base_tenant_credit_limit_throttled" {
  workspace     = data.observe_workspace.default.oid
  name          = "Global Credit Limit THROTTLED"
  setting       = "QueryGovernor.throttledLimitCreditsPerDay"
  value_float64 = 100.0
  target        = data.observe_oid.customer.oid
}

# Applying hard and soft limits for all users
# Query Governor - User Level - All Users - Throttled
# Target can be a customer or user
# when you target a customer, all users inherit this limit, unless a user is specifically targeted
resource "observe_layered_setting_record" "all_users_credit_limit_soft" {
  workspace     = data.observe_workspace.default.oid
  name          = "All Users Query Limit THROTTLED"
  setting       = "QueryGovernor.userThrottledLimitCreditsPerDay"
  value_float64 = 50.0
  target        = data.observe_oid.customer.oid
}

# Query Governor - User Level - All Users - Hard
# Target can be a customer or user
# when you target a customer, all users inherit this limit, unless a user is specifically targeted
resource "observe_layered_setting_record" "all_users_credit_limit_hard" {
  workspace     = data.observe_workspace.default.oid
  name          = "All Users Query Limit HARD"
  setting       = "QueryGovernor.userCreditsPerDay"
  value_float64 = 80.0
  target        = data.observe_oid.customer.oid
}


# Applying hard and soft limits to specific users
# These override the all users settings above for specific users
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
  target        = data.observe_user.carl_chumplin.oid
}

# Query Governor - User Level - User 1 - Hard
resource "observe_layered_setting_record" "base_admin_credit_limit_hard" {
  workspace     = data.observe_workspace.default.oid
  name          = "User 1 Query Limit HARD"
  setting       = "QueryGovernor.userCreditsPerDay"
  value_float64 = 10.0
  target        = data.observe_user.carl_chumplin.oid
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

# There are also global limits that are evaluated last.
# meaning if the user generic or user specific limits above are not applied
# these will kick in - think of them as the total general limit
# Transform Governor - Customer Level Hard Limit
# Target must be a customer

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
