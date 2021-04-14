# tfsuit

## About
Terraform customizable linter for defining your standards, styles, and naming conventions

## Install
```sh
# ...
```

## Use
```sh
tfsuit --dir="/my/project/path" --config-json-path="/my/project/path/tfsuit.json"
```

## GitHub Actions
```yml
jobs:
  # ...
```

## Configuration file
##### Common match patterns:
```json
// For TF projects' variables
{
  "vars": {
    "naming_conventions": {
      "match_pattern": "[a-z0-9_]+_(virginia|ohio|california|oregon)\\b",
      "exact": null,
      "ignore": {
        "match_pattern": null,
        "exact": [
          "route53_domain"
        ]
      }
    },
    "linter": {
      "num_of_blank_lines_above": 1,
      "num_of_blank_lines_below": 1,
      "use_quotes": false
    }
  }
}
```
