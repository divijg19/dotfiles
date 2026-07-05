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
-- 📋 CONTEXT
-- ================================
local function ctx()
  return {
    file = vim.fn.expand("%:p"),
    filetype = vim.bo.filetype,
    cwd = vim.fn.expand("%:p:h"),
    pid = vim.uv.os_getpid(),
  }
end

-- ================================
-- 🗂️ LANGUAGE ADAPTERS
-- ================================
local languages = {}

languages.c = {
  run = function(c)
    local outfile = "/tmp/nvim_run_" .. c.pid
    return {
      phases = {
        { argv = { "gcc", c.file, "-o", outfile } },
        { argv = { outfile } },
      },
    }
  end,
  build = function(c)
    local outfile = c.file:match("^(.+)%.[^.]*$") or c.file
    return {
      phases = {
        { argv = { "gcc", c.file, "-o", outfile } },
      },
    }
  end,
}

languages.go = {
  run = function(c)
    return {
      phases = {
        { argv = { "go", "run", c.file } },
      },
    }
  end,
  build = function(c)
    return {
      phases = {
        { argv = { "go", "build" } },
      },
    }
  end,
  test = function(c)
    return {
      phases = {
        { argv = { "go", "test", "./..." } },
      },
    }
  end,
}

languages.lua = {
  run = function(c)
    return {
      phases = {
        { argv = { "lua", c.file } },
      },
    }
  end,
}

-- ================================
-- 🏃 INTERACTIVE EXECUTOR
-- ================================
---@param action { phases: { argv: string[] }[] }
---@param opts? { cwd?: string }
---@param callback? fun(success: boolean, code?: integer)
local function execute(action, opts, callback)
  opts = opts or {}
  local cwd = opts.cwd or vim.uv.cwd()

  vim.cmd("botright 12split")
  local term_buf = vim.api.nvim_get_current_buf()

  local phases = action.phases
  local phase_idx = 1

  local function next_phase()
    if phase_idx > #phases then
      if callback then
        vim.schedule(function()
          callback(true)
        end)
      end
      return
    end

    local phase = phases[phase_idx]
    phase_idx = phase_idx + 1

    vim.fn.jobstart(phase.argv, {
      term = true,
      cwd = cwd,
      on_exit = function(_, code)
        vim.schedule(function()
          if code == 0 then
            vim.bo[term_buf].modified = false
            next_phase()
          else
            if callback then
              callback(false, code)
            end
          end
        end)
      end,
    })
  end

  next_phase()

  vim.keymap.set("t", "<Esc>", [[<C-\><C-n>:close<CR>]], { buf = term_buf })
  vim.keymap.set("t", "q", [[<C-\><C-n>:close<CR>]], { buf = term_buf })
  vim.cmd("startinsert")
end

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
  vim.keymap.set("n", "<Esc>", close, { buf = buf, nowait = true })
  vim.keymap.set("n", "q", close, { buf = buf, nowait = true })
  vim.keymap.set("n", "<CR>", close, { buf = buf, nowait = true })
  vim.keymap.set("n", "<Space>", close, { buf = buf, nowait = true })

  -- optional: auto-focus top
  vim.cmd("normal! gg")

  return buf
end

-- ================================
-- ⚡ ASYNC RUNNER
-- ================================
local function run_job(cmd, label)
  local function on_exit(obj)
    local lines = {}
    local function add(text)
      if text and text ~= "" then
        for line in text:gmatch("[^\n]+") do
          table.insert(lines, line)
        end
      end
    end
    add(obj.stdout)
    add(obj.stderr)

    last_output = {
      lines = (#lines > 0) and lines or { "[No output]" },
      mode = "job",
    }

    if obj.code == 0 then
      vim.notify("✅ " .. label .. " succeeded", vim.log.levels.INFO)
      if #lines > 0 then
        open_output_buf(lines)
      end
    else
      vim.notify("❌ " .. label .. " failed (code " .. obj.code .. ")", vim.log.levels.ERROR)
      open_output_buf(lines)
    end
  end

  vim.notify("▶ " .. label, vim.log.levels.INFO)
  vim.system(cmd, { text = true }, on_exit)
end

-- ================================
-- 🔑 KEYMAPS
-- ================================

-- ⚡ Run
keymap("n", "<leader>cx", function()
  vim.cmd("write")
  local c = ctx()
  local lang = languages[c.filetype]
  local action = lang and lang.run and lang.run(c)

  if action then
    last_output = { lines = nil, mode = "term" }
    vim.notify("▶ Run (" .. c.filetype .. ")", vim.log.levels.INFO)
    execute(action, { cwd = c.cwd }, function(success, code)
      if success then
        vim.notify("✔ Run finished", vim.log.levels.INFO)
      else
        vim.notify("❌ Run failed (code " .. code .. ")", vim.log.levels.ERROR)
      end
    end)
    return
  end

  vim.notify("Unsupported filetype", vim.log.levels.WARN)
end, { desc = "Run (Terminal)" })

-- 🧱 Compile
keymap("n", "<leader>cc", function()
  vim.cmd("write")
  local c = ctx()
  local lang = languages[c.filetype]
  local action = lang and lang.build and lang.build(c)

  if action then
    run_job(action.phases[1].argv, "Build (" .. c.filetype .. ")")
    return
  end

  vim.notify("Unsupported filetype", vim.log.levels.WARN)
end, { desc = "Compile" })

-- 🔁 Recall
keymap("n", "<leader>cr", function()
  if not last_output.lines then
    vim.notify("No previous output", vim.log.levels.WARN)
    return
  end

  vim.notify("↺ Recall", vim.log.levels.INFO)
  open_output_buf(last_output.lines)
end, { desc = "Recall" })

-- 🧪 Tests
keymap("n", "<leader>ct", function()
  local c = ctx()
  local lang = languages[c.filetype]
  local action = lang and lang.test and lang.test(c)

  if action then
    run_job(action.phases[1].argv, "Test (" .. c.filetype .. ")")
    return
  end

  vim.notify("No tests configured", vim.log.levels.INFO)
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

-- ================================
-- 🔧 LINT
-- ================================
keymap("n", "<leader>cl", function()
  if vim.bo.filetype == "go" then
    require("lint").try_lint({ "golangcilint" })
  else
    require("lint").try_lint()
  end
end, { desc = "Lint" })

-- 📋 Show line diagnostics (makes diagnostics discoverable)
keymap("n", "<leader>cd", function() vim.diagnostic.open_float() end, { desc = "Line Diagnostics" })

-- 📦 Mason Management
-- <leader>cm is provided by LazyVim natively. Removed local override.

