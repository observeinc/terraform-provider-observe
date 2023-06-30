# GitHub Actions Secrets

This directory contains **encrypted** secrets used by GitHub Actions. They are encrypted via GitHub using the public key for this repository and can only be decrypted by GitHub Actions via the `secrets` context.

Any file added to this directory (other than this readme) should contain an **already-encrypted** secret that will be created by Terraform in GitHub Actions, using the filename as the secret name.

## Generating a Secret

Using the [GitHub CLI](https://cli.github.com):

```sh
SECRET_NAME=MY_SECRET
gh secret set $SECRET_NAME --no-store > $SECRET_NAME
```

You can optionally pipe a value in, e.g., using the macOS clipboard:

```sh
pbpaste | gh secret set ...
```
