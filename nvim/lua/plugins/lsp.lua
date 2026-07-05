return {
  {
    "neovim/nvim-lspconfig",
    opts = {
      codelens = { enabled = false },
      servers = {
        ["*"] = {
          keys = {
            { "<leader>cr", false },
            { "<leader>cR", false },
            { "<leader>co", false },
            { "<leader>cc", false },
            { "<leader>cC", false },
            { "<leader>cA", false },
          },
        },
        lua_ls = {
          settings = {
            Lua = {
              runtime = {
                version = "LuaJIT",
              },
              diagnostics = {
                globals = { "love" },
              },
            },
          },
        },
      },
    },
  },
}
