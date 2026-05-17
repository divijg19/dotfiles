function fe
    set file (command fd | fzf)
    if test -n "$file"
        nvim "$file"
    end
end
