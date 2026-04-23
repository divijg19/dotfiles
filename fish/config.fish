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
alias dnfclean "sudo dnf clean all"
alias hist "sudo dnf history list"
alias histi "sudo dnf history info"
alias dnfundo "sudo dnf history undo"

alias cls clear
alias ls "eza --group-directories-first"
alias grep rg
alias find fd

# ----- functions -----

function ll
    eza -lh --group-directories-first $argv
end

function tree
    eza --tree --icons $argv
end

function ga
    git add $argv
end

function gc
    git commit $argv
end

function gom
    go mod tidy
end

function gof
    go fmt ./...
end

function mkcd
    mkdir -p $argv && cd $argv
end

function cat
    if isatty stdout
        bat -P $argv
    else
        command cat $argv
    end
end

function f
    command fd | fzf
end

function fe
    set file (command fd | fzf)
    if test -n "$file"
        nvim "$file"
    end
end

function se
    set match (rg --line-number --no-heading --color=always "" | fzf)
    if test -n "$match"
        set parts (string split ":" $match)
        set file $parts[1]
        set line $parts[2]
        nvim "$file" +$line
    end
end
