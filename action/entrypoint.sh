#!/usr/bin/env bash
set -euo pipefail

debug_flag=
[[ "${1:-}" == "true" ]] && debug_flag="--debug"

tfsuit $debug_flag \
  --format "${4:-pretty}" \
  --config "${3:-tfsuit.hcl}" \
  ${5:+--fail} \
  "${2:-.}"
