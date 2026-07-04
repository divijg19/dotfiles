-- LSP servers managed outside Mason.
-- These attach from PATH rather than through mason-lspconfig.
-- Everything not listed here remains managed by mason-lspconfig.

local M = {}

M.external_servers = {
  gopls = { mason = false },
  clangd = { mason = false },
}

M.excluded_servers = {
  marksman = true,
  dockerls = true,
  docker_compose_language_service = true,
  jsonls = true,
  yamlls = true,
  taplo = true,
  neocmake = true,
  nil_ls = true,
  bashls = true,
}

return M
