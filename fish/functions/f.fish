function f
    set -l target (fd $argv | fzf --query="$argv")

    test -z "$target"; and return

    if test -d "$target"
        cd "$target"
    else
        nvim "$target"
    end
end
