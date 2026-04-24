return {
  {
    "neovim/nvim-lspconfig",
    opts = {
      servers = {
        lua_ls = {
          settings = {
            Lua = {
              runtime = {
                version = "LuaJIT",
              },
              diagnostics = {
                globals = { "love" },
              },
              workspace = {
                checkThirdParty = false,
              },
              hint = {
                enable = true,
              },
            },
          },
        },
      },
    },
  },
}
