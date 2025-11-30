variables { pattern = "^[a-z0-9_]+$" }
outputs   { pattern = "^[a-z0-9_]+$" }
modules   { pattern = "^[a-z0-9_]+$" }
resources { pattern = "^[a-z0-9_]+$" }

block_spacing {
  min_blank_lines = 2
  allow_compact   = ["variable", "output"]
}
