# tfsuit

## About
Terraform customizable tool for defining your standards, styles, and naming conventions

## Install
```sh
wget https://github.com/josdagaro/tfsuit/releases/download/vx.y.z/tfsuit
mv tfsuit /usr/local/bin
chmod a+x /usr/local/bin/tfsuit
# ...
```

## Use
```sh
tfsuit --dir="/my/project/path" --config-json-path="/my/project/path/tfsuit.json" -f --docs-link="foobar.com"
```

### GitHub Actions
```yml
jobs:
  tfsuit:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - name: Run tfsuit
        id: tfsuit
        uses: josdagaro/tfsuit@v0.5.0
        with:
          dir: "."
          config_json_path: tfsuit.json
          fail_on_not_compliant: "true"
```

### Configuration file
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
    "line_breaks": {
      "blank_lines_above": 1,
      "blank_lines_below": 1
    }
  }
}
```
