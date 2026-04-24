return {
  -- LOVE2D type definitions
  {
    "LuaCATS/love2d",
    lazy = true,
  },

  -- Inject definitions into LuaLS cleanly
  {
    "folke/lazydev.nvim",
    ft = "lua",
    opts = {
      library = {
        { path = "love2d/love", words = { "love" } },
      },
    },
  },
}
