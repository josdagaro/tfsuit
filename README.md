# tfsuit
Terraform customizable linter for defining your standards, styles, and naming conventions

## Install
```sh
# ...
```

## Use
```sh
tfsuit --dir "/my/project/path" --config-json-path "/my/project/path/tfsuit.json"
```

## GitHub Actions
```yml
jobs:
  # ...
```

## Configuration file
##### Common match patterns:
```sh
# For TF projects' variables
'variable\s+[a-z0-9_]+_(virginia|ohio|california|oregon)\b'
```
