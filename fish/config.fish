if status is-interactive
    fish_vi_key_bindings

    starship init fish | source
    atuin init fish | source

    ## Delay shell integrations until the first fish_prompt
    function __lazy_init --on-event fish_prompt
        functions -e __lazy_init

        zoxide init fish | source
        fzf --fish | source
    end
end

set -gx LESS -R

# ----- local functions -----

if test -d ~/.config/fish/functions/local
    for file in ~/.config/fish/functions/local/*.fish
        test -f $file; and source $file
    end
end
