# Provider Infrastructure

This directory is a Terraform module that defines backing infrastructure for the provider, for use in testing and releasing the provider.

## Usage

This module is not applied automatically. After merging changes, you should `terraform apply` locally.

### Requirements

* Set `GITHUB_TOKEN` to a GitHub access token with at least the `repo` scope

## Contents

### Testing

* GitHub Actions secrets and variables with credentials that can be used to execute acceptance tests against a live instance of the Observe API
