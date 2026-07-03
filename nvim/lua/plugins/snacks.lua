return {
  "snacks.nvim",
  opts = {
    explorer = { replace_netrw = false },
  },
  keys = {
    {
      "<leader>E",
      function()
        Snacks.explorer.reveal()
      end,
      desc = "Explorer Reveal (Snacks)",
    },
  },
}
