#!/bin/bash

github::set_output() {
  local name
  local message
  name="$1"
  message="$2"

  if [ "$set_github_actions_outputs" == 1 ]; then
    echo "::set-output name=$name::$message"
  fi
}
