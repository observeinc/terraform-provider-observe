# GitHub Actions Secrets

This directory contains **encrypted** secrets used by GitHub Actions. They are encrypted via GitHub using the public key for this repository and can only be decrypted by GitHub Actions via the `secrets` context.

Any file added to this directory (other than this readme) should contain an **already-encrypted** secret that will be created by Terraform in GitHub Actions, using the filename as the secret name.

The `actions/` contains Actions secrets while `dependabot/` contains [Dependabot secrets](https://docs.github.com/en/code-security/dependabot/working-with-dependabot/automating-dependabot-with-github-actions#accessing-secrets). These use different encryption keys, so encrypted secrets for Actions cannot be used by Dependabot jobs, and vice versa.

## Generating a Secret

Using the [GitHub CLI](https://cli.github.com):

```sh
SECRET_NAME=MY_SECRET
APP=actions
gh secret set "$SECRET_NAME" --app "$APP" --no-store > "$APP/$SECRET_NAME"
```

For Dependabot secrets, change `APP` to `dependabot`.

You can optionally pipe a value in, e.g., using the macOS clipboard:

```sh
pbpaste | gh secret set ...
```
