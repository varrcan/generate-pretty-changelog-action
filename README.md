[![License](https://img.shields.io/github/license/varrcan/generate-pretty-changelog-action.svg?style=flat-square)](LICENSE)
[![Last commit](https://img.shields.io/github/last-commit/varrcan/generate-pretty-changelog-action.svg?style=flat-square)](https://github.com/varrcan/generate-pretty-changelog-action/commits)
[![Latest tag](https://img.shields.io/github/tag/varrcan/generate-pretty-changelog-action.svg?style=flat-square)](https://github.com/varrcan/generate-pretty-changelog-action/releases)
[![Issues](https://img.shields.io/github/issues/varrcan/generate-pretty-changelog-action.svg?style=flat-square)](https://github.com/varrcan/generate-pretty-changelog-action/issues)
[![Pull requests](https://img.shields.io/github/issues-pr/varrcan/generate-pretty-changelog-action.svg?style=flat-square)](https://github.com/varrcan/generate-pretty-changelog-action/pulls)

# ✏️ Generate Pretty Changelog

Automatically generate changelog from your pull requests on GitHub.

This action also makes the changelog available to other actions as [output](#outputs).

## Example usage

```yaml
name: Changelog
on:
  push:
    tags:
      - "*"
jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v3
      - name: Generate release changelog
        uses: varrcan/generate-pretty-changelog-action@v1
      - name: Release
        uses: softprops/action-gh-release@v1
        with:
          body_path: CHANGELOG.md
```

## Inputs

| Name     | Description                                             | Required | Default                       |
|----------|---------------------------------------------------------|----------|-------------------------------|
| `use`    | Changelog generation implementation to use              | no       | `github`                      |
| `config` | Use custom config file                                  | no       | `changelog.yaml`              |
| `token`  | GitHub token (only required if github type is selected) | no       | `${{ secrets.GITHUB_TOKEN }}` |

## Outputs

| Name        | Description                       |
|-------------|-----------------------------------|
| `changelog` | Contents of generated change log. |

