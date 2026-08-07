if status is-interactive
    starship init fish | source
    atuin init fish --disable-up-arrow | source

    ## Delay shell integrations until the first fish_prompt
    function __lazy_init --on-event fish_prompt
        functions -e __lazy_init

        zoxide init fish | source
        fzf --fish | FZF_CTRL_R_COMMAND= source
    end
end

set -gx LESS -R

# ----- local functions -----

if test -d ~/.config/fish/functions/local
    for file in ~/.config/fish/functions/local/*.fish
        test -f $file; and source $file
    end
end

# bun
set --export BUN_INSTALL "$HOME/.bun"
set --export PATH $BUN_INSTALL/bin $PATH
