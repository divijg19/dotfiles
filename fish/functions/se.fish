function se
    set match (rg --line-number --no-heading --color=always "" | fzf)
    if test -n "$match"
        set parts (string split ":" $match)
        set file $parts[1]
        set line $parts[2]
        nvim "$file" +$line
    end
end
