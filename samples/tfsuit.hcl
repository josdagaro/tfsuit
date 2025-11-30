variables {
  pattern = "^[a-z0-9_]+$"
}

outputs {
  pattern = "^[a-z0-9_]+$"
}

modules {
  pattern = "^[a-z0-9_]+$"
  require_provider = true
}

resources {
  pattern = "^[a-z0-9_]+$"
  require_provider = true
}

data {
  pattern = "^[a-z0-9_]+$"
}

files {
  pattern = "^[a-z0-9_]+\\.tf$"
}
