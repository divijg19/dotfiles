function se
    set -l match (rg --line-number --no-heading --color=always "" | fzf)
    if test -n "$match"
        set -l parts (string split ":" $match)
        set -l file $parts[1]
        set -l line $parts[2]
        nvim "$file" +$line
    end
end
