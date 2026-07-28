if status is-interactive
    starship init fish | source
    atuin init fish | source

    ## Delay shell integrations until the first fish_prompt
    function __zoxide_init --on-event fish_prompt
        functions -e __zoxide_init

        zoxide init fish | source
        fzf --fish | source
    end
end

set -gx LESS -R

# ----- aliases -----

alias gt "git tag"
alias gs "git status"
alias gl "git log --oneline --graph --decorate"
alias gp "git push"
alias gpu "git pull"
alias gpf "git push --force-with-lease"

alias got "go test ./..."
alias gob "go build ./..."

alias upg "sudo dnf upgrade -y"
alias upgr "sudo dnf upgrade --refresh -y"
alias upgc "sudo dnf upgrade --refresh --assumeno"
alias dsync "sudo dnf distro-sync --refresh -y && sudo dnf autoremove -y"
alias dnfclean "sudo dnf clean all && sudo rm -rf /var/cache/dnf && sudo rm -rf /var/cache/libdnf5"
alias hist "sudo dnf history list"
alias histi "sudo dnf history info"
alias dnfundo "sudo dnf history undo"

alias cls clear
alias ls "eza --group-directories-first"

# ----- local functions -----

if test -d ~/.config/fish/functions/local
    for file in ~/.config/fish/functions/local/*.fish
        test -f $file; and source $file
    end
end
