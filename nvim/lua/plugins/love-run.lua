local love_buf = nil

return {
  {
    "folke/snacks.nvim",
    keys = {
      {
        "<leader>r",
        function()
          vim.cmd("wa")

          local root = vim.fn.expand("%:p:h")
          while root ~= "/" do
            if vim.fn.filereadable(root .. "/main.lua") == 1 then
              break
            end
            root = vim.fn.fnamemodify(root, ":h")
          end
          if root == "/" then
            vim.notify("No main.lua found", vim.log.levels.ERROR)
            return
          end

          if love_buf and vim.api.nvim_buf_is_valid(love_buf) then
            for _, win in ipairs(vim.api.nvim_list_wins()) do
              if vim.api.nvim_win_get_buf(win) == love_buf then
                vim.api.nvim_win_close(win, true)
                break
              end
            end
            pcall(vim.api.nvim_buf_delete, love_buf, { force = true })
            love_buf = nil
          end

          local win = Snacks.win({
            position = "float",
            border = "rounded",
            width = 0.8,
            height = 0.8,
            wo = { winbar = "LOVE" },
          })

          love_buf = win.buf

          local ok = vim.api.nvim_buf_call(love_buf, function()
            return vim.fn.termopen({ "love", root }, {
              cwd = root,
              on_exit = vim.schedule_wrap(function()
                vim.notify("♥ LOVE exited", vim.log.levels.INFO)
              end),
            })
          end)

          if ok == -1 then
            win:close()
            love_buf = nil
            vim.notify("Failed to launch LOVE", vim.log.levels.ERROR)
            return
          end

          vim.cmd("startinsert")
        end,
        desc = "Run LOVE",
      },
    },
  },
}
