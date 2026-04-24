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

      local function find_love_root()
        local path = vim.fn.expand("%:p:h")

        while path ~= "/" do
          if vim.fn.filereadable(path .. "/main.lua") == 1 then
            return path
          end
          path = vim.fn.fnamemodify(path, ":h")
        end

        return nil
      end

      local love_term = nil

      local function run_love()
        vim.cmd("wa")

        local root = find_love_root()
        if not root then
          vim.notify("No main.lua found", vim.log.levels.ERROR)
          return
        end

        if love_term then
          love_term:close()
          love_term = nil
        end

        love_term = Terminal:new({
          cmd = "love " .. root,
          hidden = true,
          direction = "float",
          close_on_exit = false,
        })

        love_term:toggle()
      end

      vim.keymap.set("n", "<leader>r", run_love, { desc = "Run LOVE" })
    end,
  },
}
