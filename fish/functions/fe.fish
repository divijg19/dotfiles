function fe
    set -l file (command fd | fzf)
    if test -n "$file"
        nvim "$file"
    end
end
