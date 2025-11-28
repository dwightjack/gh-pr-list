# gh-pr-list

A GitHub CLI extension to quickly list all your open pull requests.

## Why?

Keeping track of your open pull requests across multiple repositories can be time-consuming. This extension provides a quick way to view all your open PRs in one place, helping you stay organized and ensure nothing falls through the cracks.

## Installation

```sh
gh extension install dwightjack/gh-pr-list
```

## Usage

List all your open pull requests:

```sh
gh pr-list
```

### Options

Filter PRs by organization:

```sh
gh pr-list --org myorg
```

Output as markdown (useful for copy-pasting into documents):

```sh
gh pr-list --markdown
```

Output as JSON (useful for scripting):

```sh
gh pr-list --json
```

### Examples

```sh
# List all your open PRs
gh pr-list

# List PRs only from your organization
gh pr-list --org acme-corp

# Export as markdown for your daily standup notes
gh pr-list --markdown >> standup-notes.md

# Use JSON output with jq to count total PRs
gh pr-list --json | jq 'length'
```

## Alias

You can create shorter aliases using `gh alias set`:

```sh
gh alias set prl 'pr-list'
gh prl  # Now you can use the shorter command
```

## License

[MIT](LICENSE)
