-- Tools managed outside Mason.
-- Used to filter ensure_installed lists.
--
-- The OS and language ecosystems own executable installation.
-- LazyVim owns only editor configuration. Mason is reserved for tools whose primary consumer is Neovim.
-- See: lua/policy/README.md

local M = {}

M.external = {
  "shfmt",
  "shellcheck",
  "hadolint",
  "goimports",
  "gofumpt",
  "golangci-lint",
  "delve",
  "gomodifytags",
  "impl",
  "stylua",
  "cmakelang",
  "cmakelint",
  "markdownlint-cli2",
  "markdown-toc",
}

return M
