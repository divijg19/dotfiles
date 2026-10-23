function gn --description 'Initialize a profile-aware GitHub repository'
    argparse h/help -- $argv
    or return 2

    if set -q _flag_help
        printf 'Usage: gn <directory>\n\n'
        printf 'Initialize a Git repository with its profile-aware GitHub SSH remote.\n\n'
        printf 'Examples:\n'
        printf '  gn my-project\n'
        printf '  gn ~/Projects/my-project\n'
        printf '  gn .\n'
        return 0
    end

    if test (count $argv) -ne 1
        printf '%sUsage:%s gn <directory>\n' \
            (set_color yellow --bold) (set_color normal) >&2
        printf "Run 'gn --help' for help.\n" >&2
        return 2
    end

    set -l target (path resolve -- "$argv[1]")
    or return 1

    set -l repo (path basename -- "$target")
    set -l reset (set_color normal)
    set -l muted (set_color brblack)
    set -l green (set_color green --bold)
    set -l red (set_color red --bold)
    set -l cyan (set_color cyan --bold)
    set -l yellow (set_color yellow --bold)

    if not string match -rq '^[A-Za-z0-9_.-]+$' -- "$repo"
        printf '%s✗%s Invalid repository name: %s\n' \
            "$red" "$reset" "$repo" >&2
        return 2
    end

    if test -e "$target/.git"
        printf '%s✗%s Git repository already exists:\n  %s\n' \
            "$red" "$reset" "$target" >&2
        return 1
    end

    if not mkdir -p -- "$target"
        printf '%s✗%s Could not create directory:\n  %s\n' \
            "$red" "$reset" "$target" >&2
        return 1
    end

    printf '\n%s gn %s\n' "$cyan" "$reset"
    printf '%s────────────────────────────────────────%s\n' \
        "$muted" "$reset"
    printf '%s  Initializing%s %s\n\n' "$muted" "$reset" "$repo"

    if not git init --quiet "$target"
        printf '%s✗%s Git initialization failed.\n' \
            "$red" "$reset" >&2
        return 1
    end

    set -l profile (git -C "$target" config --get gid.profile)
    set -l owner (git -C "$target" config --get gid.owner)
    set -l host (git -C "$target" config --get gid.sshHost)

    if test -z "$profile"; or test -z "$owner"; or test -z "$host"
        printf '%s✗%s Could not resolve GitHub profile.\n' \
            "$red" "$reset" >&2
        printf '%s  Required: gid.profile, gid.owner, gid.sshHost%s\n' \
            "$muted" "$reset" >&2
        printf '%s  Repository initialized at %s%s\n' \
            "$muted" "$target" "$reset" >&2
        return 1
    end

    if not string match -rq '^[A-Za-z0-9-]+$' -- "$owner"; or not string match -rq '^[A-Za-z0-9._-]+$' -- "$host"
        printf '%s✗%s Invalid GitHub owner or SSH host.\n' \
            "$red" "$reset" >&2
        return 1
    end

    set -l remote "git@$host:$owner/$repo.git"

    if not git -C "$target" remote add origin "$remote"
        printf '%s✗%s Could not configure origin.\n' \
            "$red" "$reset" >&2
        printf '%s  Repository initialized at %s%s\n' \
            "$muted" "$target" "$reset" >&2
        return 1
    end

    if not cd "$target"
        printf '%s✗%s Repository created, but could not enter it.\n' \
            "$red" "$reset" >&2
        return 1
    end

    printf '%s✓%s Git repository initialized\n' "$green" "$reset"
    printf '%s✓%s SSH remote configured\n' "$green" "$reset"

    printf '\n%s Repository ready %s\n' "$green" "$reset"
    printf '%s  Profile %s  %s%s\n' "$muted" "$reset" "$cyan" "$profile"
    printf '%s  Path    %s  %s\n' "$muted" "$reset" (pwd)
    printf '%s  Branch  %s  %s\n' \
        "$muted" "$reset" (git branch --show-current)
    printf '%s  Remote  %s  %s%s\n\n' \
        "$muted" "$reset" "$yellow" "$remote"
end
