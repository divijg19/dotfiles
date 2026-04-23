local keymap = vim.keymap.set

-- ================================
-- 🧠 STATE
-- ================================
local last_output = {
  lines = nil,
  mode = nil, -- "term" | "job"
}
local main_buf = nil

-- ================================
-- 🪟 EPHEMERAL OUTPUT SPLIT
-- ================================
local function open_output_buf(lines)
  local cwd = vim.fn.expand("%:p:h")

  vim.cmd("botright 12split")
  vim.cmd("lcd " .. vim.fn.fnameescape(cwd))

  local buf = vim.api.nvim_get_current_buf()
  -- prepare content
  local content = vim.deepcopy(lines or {})
  if #content == 0 then
    table.insert(content, "[No output]")
  end

  table.insert(content, "")
  table.insert(content, "↵ Press any key to continue...")

  vim.api.nvim_buf_set_lines(buf, 0, -1, false, content)

  -- buffer config
  vim.bo[buf].buftype = "nofile"
  vim.bo[buf].bufhidden = "wipe"
  vim.bo[buf].swapfile = false
  vim.bo[buf].modifiable = false

  local function close()
    if vim.api.nvim_buf_is_valid(buf) then
      vim.api.nvim_buf_delete(buf, { force = true })
    end
  end

  -- 🔥 true “any key” feel (without blocking)
  vim.keymap.set("n", "<Esc>", close, { buffer = buf, nowait = true })
  vim.keymap.set("n", "q", close, { buffer = buf, nowait = true })
  vim.keymap.set("n", "<CR>", close, { buffer = buf, nowait = true })
  vim.keymap.set("n", "<Space>", close, { buffer = buf, nowait = true })

  -- optional: auto-focus top
  vim.cmd("normal! gg")

  return buf
end

-- ================================
-- ⚡ ASYNC RUNNER
-- ================================
local function run_job(cmd, label)
  local output = {}

  vim.notify("▶ " .. label, vim.log.levels.INFO)

  vim.fn.jobstart(cmd, {
    stdout_buffered = true,
    stderr_buffered = true,

    on_stdout = function(_, data)
      if not data then
        return
      end
      for _, line in ipairs(data) do
        if line ~= "" then
          table.insert(output, line)
        end
      end
    end,

    on_stderr = function(_, data)
      if not data then
        return
      end
      for _, line in ipairs(data) do
        if line ~= "" then
          table.insert(output, line)
        end
      end
    end,

    on_exit = function(_, code)
      vim.schedule(function()
        last_output = {
          lines = (#output > 0) and vim.deepcopy(output) or { "[No output]" },
          mode = "job",
        }

        if code == 0 then
          vim.notify("✅ " .. label .. " succeeded", vim.log.levels.INFO)

          -- ✅ ONLY show output if something exists
          if #output > 0 then
            open_output_buf(output)
          end
        else
          vim.notify("❌ " .. label .. " failed (code " .. code .. ")", vim.log.levels.ERROR)

          -- ❗ ALWAYS show errors
          open_output_buf(output)
        end
      end)
    end,
  })
end

-- ================================
-- 🖥️ TERMINAL RUNNER
-- ================================
local function run_in_terminal(cmd)
  last_output = { lines = nil, mode = "term" }
  local cwd = vim.fn.expand("%:p:h")

  vim.cmd("botright 12split")
  vim.cmd("lcd " .. vim.fn.fnameescape(cwd))

  vim.cmd("terminal bash -c " .. vim.fn.shellescape(cmd))
  vim.cmd("startinsert")

  -- scoped buffer
  local term_buf = vim.api.nvim_get_current_buf()

  -- close mappings
  vim.keymap.set("t", "<Esc>", [[<C-\><C-n>:close<CR>]], { buffer = term_buf })
  vim.keymap.set("t", "q", [[<C-\><C-n>:close<CR>]], { buffer = term_buf })

  -- notify on exit
  vim.api.nvim_create_autocmd("TermClose", {
    buffer = term_buf,
    once = true,
    callback = function()
      vim.notify("✔  Run finished", vim.log.levels.INFO)
    end,
  })
end

-- ================================
-- ⚙️ COMMAND RESOLUTION
-- ================================
local function run_cmd()
  local file = vim.fn.shellescape(vim.fn.expand("%:p"))

  if vim.bo.filetype == "c" then
    local outfile = "/tmp/nvim_run_" .. vim.fn.getpid()
    return "gcc " .. file .. " -o " .. outfile .. " && " .. outfile, "Run (C)"
  elseif vim.bo.filetype == "go" then
    return "go run " .. file, "Run (Go)"
  end
end

local function build_cmd()
  if vim.bo.filetype == "c" then
    local file = vim.fn.shellescape(vim.fn.expand("%:p"))
    local out = vim.fn.shellescape(vim.fn.expand("%:p:r"))
    return "gcc " .. file .. " -o " .. out, "Compile (C)"
  elseif vim.bo.filetype == "go" then
    return "go build", "Build (Go)"
  end
end

local function test_cmd()
  if vim.bo.filetype == "go" then
    return "go test ./...", "Tests (Go)"
  end
end

-- ================================
-- 🔑 KEYMAPS
-- ================================

-- ⚡ Run
keymap("n", "<leader>cx", function()
  vim.cmd("write")
  local cmd, label = run_cmd()

  if not cmd then
    vim.notify("Unsupported filetype", vim.log.levels.WARN)
    return
  end

  vim.notify("▶ " .. label, vim.log.levels.INFO)
  run_in_terminal(cmd)
end, { desc = "Run (Terminal)" })

-- 🧱 Compile
keymap("n", "<leader>cc", function()
  vim.cmd("write")
  local cmd, label = build_cmd()

  if not cmd then
    vim.notify("Unsupported filetype", vim.log.levels.WARN)
    return
  end

  run_job(cmd, label)
end, { desc = "Compile" })

-- 🔁 Recall
keymap("n", "<leader>cz", function()
  if not last_output.lines then
    vim.notify("No previous output", vim.log.levels.WARN)
    return
  end

  vim.notify("↺ Recall", vim.log.levels.INFO)
  open_output_buf(last_output.lines)
end, { desc = "Recall" })

-- 🧪 Tests
keymap("n", "<leader>t", function()
  local cmd, label = test_cmd()

  if not cmd then
    vim.notify("No tests configured", vim.log.levels.INFO)
    return
  end

  vim.notify("🧪 " .. label, vim.log.levels.INFO)
  run_job(cmd, label)
end, { desc = "Tests" })

-- ================================
-- 🖥️ MAIN TERMINAL (singleton, clean)
-- ================================
keymap({ "n", "t" }, "<C-/>", function()
  -- If terminal exists → close & wipe
  if main_buf and vim.api.nvim_buf_is_valid(main_buf) then
    for _, win in ipairs(vim.api.nvim_list_wins()) do
      if vim.api.nvim_win_get_buf(win) == main_buf then
        vim.api.nvim_win_close(win, true)
        vim.api.nvim_buf_delete(main_buf, { force = true })
        main_buf = nil
        return
      end
    end
  end

  -- Resolve cwd from current file
  local cwd = vim.fn.expand("%:p:h")

  -- Create fresh terminal
  vim.cmd("botright 12split")

  -- Set window-local cwd BEFORE terminal
  vim.cmd("lcd " .. vim.fn.fnameescape(cwd))
  vim.cmd("terminal")

  main_buf = vim.api.nvim_get_current_buf()

  -- Enter terminal mode immediately
  vim.cmd("startinsert")
end, { desc = "Main Terminal" })

-- ================================
-- 🚀 EXTERNAL TERMINAL
-- ================================
keymap("n", "<C-`>", function()
  local cwd = vim.fn.expand("%:p:h")
  vim.fn.jobstart({ "ghostty" }, { cwd = cwd, detach = true })
end, { desc = "Ghostty (cwd)" })
