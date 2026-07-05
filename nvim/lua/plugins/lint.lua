return {
  {
    "mfussenegger/nvim-lint",
    opts = {
      -- Runtime policy:
      -- Disable automatic heavyweight linters inherited from LazyVim extras.
      -- Project-wide analysis (e.g. golangci-lint) is intentionally explicit
      -- via <leader>cl rather than automatic during editing.
      linters_by_ft = {
        go = {},
        nix = {},
      },
    },
  },
}
