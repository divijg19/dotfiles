if status is-interactive
    starship init fish | source

    function __zoxide_init --on-event fish_prompt
        functions -e __zoxide_init
        zoxide init fish | source
    end

    function __fzf_init --on-event fish_prompt
        functions -e __fzf_init
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
alias dnfclean "sudo dnf clean all && sudo rm -rf /var/cache/dnf"
alias hist "sudo dnf history list"
alias histi "sudo dnf history info"
alias dnfundo "sudo dnf history undo"

alias cls clear
alias ls "eza --group-directories-first"
