return {
  "stevearc/oil.nvim",
  cmd = "Oil",
  keys = {
    {
      "-",
      "<CMD>Oil<CR>",
      desc = "Filesystem Edit",
    },
  },
  opts = {
    default_file_explorer = true,
    float = {
      border = "rounded",
    },
  },
}
