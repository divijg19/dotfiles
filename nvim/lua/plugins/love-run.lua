return {
  {
    "akinsho/toggleterm.nvim",
    opts = {
      direction = "float",
      float_opts = {
        border = "rounded",
      },
    },
    config = function(_, opts)
      require("toggleterm").setup(opts)

      local Terminal = require("toggleterm.terminal").Terminal

      local love = Terminal:new({
        cmd = "love .",
        hidden = true,
        direction = "float",
      })

      function _LOVE_RUN()
        love:toggle()
      end

      vim.keymap.set("n", "<leader>r", _LOVE_RUN, { desc = "Run LOVE" })
    end,
  },
}
