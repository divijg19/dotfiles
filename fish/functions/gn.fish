function gn --description 'Initialize a profile-aware GitHub repository'
    argparse h/help -- $argv
    or return 2

    if set -q _flag_help
        printf 'Usage: gn <directory>\n\n'
        printf 'Initialize a Git repository and configure its GitHub SSH remote.\n\n'
        printf 'Arguments:\n'
        printf '  directory   Target directory; repository name defaults to its basename\n\n'
        printf 'Options:\n'
        printf '  -h, --help  Show this help\n\n'
        printf 'Examples:\n'
        printf '  gn my-project\n'
        printf '  gn ~/Projects/my-project\n'
        printf '  gn .\n'
        return 0
    end

    if test (count $argv) -ne 1
        printf 'Usage: gn <directory>\n' >&2
        printf "Run 'gn --help' for help.\n" >&2
        return 2
    end

    set -l target (path resolve -- "$argv[1]")
    set -l repo (path basename -- "$target")

    if not string match -rq '^[A-Za-z0-9_.-]+$' -- "$repo"
        printf '%s✗%s Invalid repository name: %s\n' \
            (set_color red) (set_color normal) "$repo" >&2
        return 2
    end

    if test -e "$target/.git"
        printf '%s✗%s A Git repository already exists at %s\n' \
            (set_color red) (set_color normal) "$target" >&2
        return 1
    end

    if not mkdir -p -- "$target"
        printf '%s✗%s Could not create the directory.\n' \
            (set_color red) (set_color normal) >&2
        return 1
    end

    # Resolve the conditional Git configuration for the target path.
    set -l ssh_host (git -C "$target" config --get gid.sshHost 2>/dev/null)
    set -l email (git -C "$target" config --get user.email 2>/dev/null)

    if test -z "$ssh_host"; or test -z "$email"
        printf '%s✗%s Could not resolve the GitHub identity.\n' \
            (set_color red) (set_color normal) >&2
        printf '  Check gid.sshHost and user.email in your Git configuration.\n' >&2
        return 1
    end

    if not string match -rq '^[A-Za-z0-9._-]+$' -- "$ssh_host"
        printf '%s✗%s Invalid SSH host alias.\n' \
            (set_color red) (set_color normal) >&2
        return 1
    end

    # Derive the owner from GitHub's noreply email convention.
    set -l identity (string match -r '^[0-9]+\+([A-Za-z0-9-]+)@users\.noreply\.github\.com$' -- "$email")

    if test (count $identity) -lt 2
        printf '%s✗%s Cannot derive the GitHub owner from user.email.\n' \
            (set_color red) (set_color normal) >&2
        return 1
    end

    set -l owner $identity[2]
    set -l remote "git@$ssh_host:$owner/$repo.git"

    printf '\n%s gn %s\n' (set_color --bold) (set_color normal)
    printf '%s────────────────────────────────────%s\n' \
        (set_color brblack) (set_color normal)

    if not git init --quiet "$target"
        printf '%s✗%s Git initialization failed.\n' \
            (set_color red) (set_color normal) >&2
        return 1
    end

    printf '  %s✓%s Git repository initialized\n' \
        (set_color green) (set_color normal)

    if not git -C "$target" remote add origin "$remote"
        printf '  %s✗%s Could not configure origin.\n' \
            (set_color red) (set_color normal) >&2
        return 1
    end

    printf '  %s✓%s SSH remote configured\n' \
        (set_color green) (set_color normal)

    if not cd "$target"
        printf '  %s✗%s Repository created, but could not enter it.\n' \
            (set_color red) (set_color normal) >&2
        return 1
    end

    printf '\n%s Repository ready %s\n' \
        (set_color green --bold) (set_color normal)
    printf '  %sPath%s    %s\n' \
        (set_color brblack) (set_color normal) (pwd)
    printf '  %sProfile%s %s\n' \
        (set_color brblack) (set_color normal) \
        (git config --get gid.profile)
    printf '  %sBranch%s  %s\n' \
        (set_color brblack) (set_color normal) (git branch --show-current)
    printf '  %sRemote%s  %s\n\n' \
        (set_color brblack) (set_color normal) "$remote"
end
