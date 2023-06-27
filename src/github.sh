#!/bin/bash

github::set_output() {
  local name
  local message
  name="$1"
  message="$2"

  if [ "$set_github_actions_outputs" -eq 1 ]; then
    echo "$name=$message" >> "$GITHUB_OUTPUT"
  fi
}
