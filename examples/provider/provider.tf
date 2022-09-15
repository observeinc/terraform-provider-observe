terraform {
  required_providers {
    observe = {
      source  = "terraform.observeinc.com/observeinc/observe"
      version = "~> 0.11"
    }
  }
}

# Configure the observe provider
provider "observe" {}

# Look up existing workspace 
data "observe_workspace" "default" {
  name = "Default"
}
