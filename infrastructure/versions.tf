terraform {
  required_providers {
    github = {
      source  = "integrations/github"
      version = ">= 5"
    }

    aws = {
      source  = "hashicorp/aws"
      version = ">= 5"
    }
  }
}
