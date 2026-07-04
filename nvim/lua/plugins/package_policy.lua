-- Central tool ownership policy.
-- Adapts package ownership decisions into LazyVim plugin overrides.
-- Filtering runs after LazyVim's opts merge, so every extra's
-- ensure_installed contributions are visible.

local policy = require("policy.packages")
local lsp_policy = require("policy.lsp")

return {
  -- Layer 1: Mason ensure_installed — filter out externally managed tools
  {
    "mason-org/mason.nvim",
    opts = function(_, opts)
      opts.ensure_installed = vim.tbl_filter(function(tool)
        return not vim.tbl_contains(policy.external, tool)
      end, opts.ensure_installed or {})
    end,
  },
  -- Layer 2: LSP servers — bypass Mason for external; exclude unwanted servers
  {
    "neovim/nvim-lspconfig",
    opts = function(_, opts)
      -- 1. Exclude blocked servers
      for server in pairs(opts.servers) do
        if lsp_policy.excluded_servers[server] then
          opts.servers[server] = nil
        end
      end

      -- 2. Inject external servers
      for server, server_opts in pairs(lsp_policy.external_servers) do
        if opts.servers[server] == nil then
          opts.servers[server] = server_opts
        elseif type(opts.servers[server]) == "table" then
          opts.servers[server] = vim.tbl_deep_extend("force", opts.servers[server], server_opts)
        end
      end
    end,
  },
}
