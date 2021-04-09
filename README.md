# tfsuit
Terraform customizable linter for defining your standards, styles, and naming conventions

Common match patterns:
```sh
# For variables
echo "my vars..." | grep -oE 'variable\s+[a-z0-9_]+_(virginia|ohio|california|oregon)\b'
# For not matching variables
echo "my vars..." | grep -oE 'variable\s+[a-z0-9_]+_(virginia|ohio|california|oregon)\b' | grep -oE 'variable\s+[a-z0-9_]+'
```
