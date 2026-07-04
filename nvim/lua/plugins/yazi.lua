return {
  "mikavilpas/yazi.nvim",
  version = "*",
  event = "VeryLazy",
  dependencies = { { "nvim-lua/plenary.nvim", lazy = true } },
  keys = {
    {
      "<leader>y",
      "<cmd>Yazi<cr>",
      desc = "Yazi (cwd)",
    },
    {
      "<leader>Y",
      function()
        local root = LazyVim.root.git() or LazyVim.root()
        require("yazi").yazi(nil, root)
      end,
      desc = "Yazi (project root)",
    },
  },
  opts = {
    open_for_directories = false,
    floating_window_scaling_factor = 0.9,
    yazi_floating_window_border = "rounded",
    highlight_hovered_buffers_in_same_directory = true,
    change_neovim_cwd_on_close = false,
    integrations = {
      grep_in_directory = "snacks.picker",
      grep_in_selected_files = "snacks.picker",
      bufdelete_implementation = "bundled-snacks",
    },
  },
}
