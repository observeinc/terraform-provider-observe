# Contributing to the Observe Provider

## Development

The Observe provider is developed with [Go](https://go.dev) and requires a minimium version specified in [`go.mod`](go.mod).

### Testing

To run the acceptance tests, you must set the following environment variables to valid values for an Observe customer instance:

* `OBSERVE_CUSTOMER`
* `OBSERVE_USER_EMAIL`
* `OBSERVE_USER_PASSWORD`
* `OBSERVE_DOMAIN`

Then:

```sh
make testacc
```

## Pull Requests

To submit a change for consideration, [open a pull request](https://github.com/observeinc/terraform-provider-observe/compare). When updating your pull request, push a new commit. Do not rewrite history and force-push if you can avoid it, since this makes it impossible for reviewers to see what changed since their last review.

Only the "squash" merge strategy is enabled for this repository, so every pull request will become a single commit upon merge, with the pull request title as the commit message.

## Releasing

To trigger a new release, run the [release](https://github.com/observeinc/terraform-provider-observe/actions/workflows/release.yml) workflow, either from the UI, or the CLI:

```sh
gh workflow run .github/workflows/release.yml \
  --field tag=v0.0.0 \
  --field publish=true
```

You must set the `tag` input when calling this workflow and select the appropriate next semver tag. 

If the `publish` input is not set, release artifacts will be created for the job and planned S3 uploads will be logged but not copied. When `publish` is enabled, a GitHub Release will be created and the release artifacts will be uploaded to the Observe Terraform Registry via S3.

### Prerelease

**⚠️ Caution:** Triggering the release job on a reference other than a tag applied to a commit on the default branch will release a version of the provider using code that has not yet been tested and merged. Take caution to include a [prerelease suffix](https://semver.org/#spec-item-9) to ensure that users do not download these versions by default.

The release workflow also supports the `workflow_dispatch` event for manually triggering a release on a specific reference. Using the [UI](https://github.com/observeinc/terraform-provider-observe/actions/workflows/release.yml) or CLI:

```sh
gh workflow run .github/workflows/release.yaml \
  --ref my-branch-name \
  --field tag=v0.0.1-rc1 \
  --field publish=true
```

You must set the `tag` input when calling this workflow and select an appropriate prerelease tag. 
