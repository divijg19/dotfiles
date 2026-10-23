function gcl --description 'Clone a GitHub repository using configured SSH or HTTPS'
    argparse --stop-nonopt \
        h/help \
        'H/host=' \
        https \
        -- $argv
    or return 2

    if set -q _flag_help
        echo "Usage: gcl [OPTIONS] OWNER REPO [CLONE_OPTIONS]"
        echo ""
        echo "Options:"
        echo "  -H, --host HOST  SSH host alias to use"
        echo "      --https      Clone over HTTPS"
        echo "  -h, --help       Show this help"
        echo ""
        echo "Examples:"
        echo "  gcl OWNER REPO"
        echo "  gcl --host github-work OWNER REPO"
        echo "  gcl --https OWNER REPO"
        echo "  gcl OWNER REPO --depth 1"
        return 0
    end

    if test (count $argv) -lt 2
        echo "Usage: gcl [OPTIONS] OWNER REPO [CLONE_OPTIONS]" >&2
        return 2
    end

    set -l owner $argv[1]
    set -l repo $argv[2]
    set -l clone_options $argv[3..-1]

    if not string match -rq '^[A-Za-z0-9][A-Za-z0-9-]*$' -- $owner
        echo "gcl: invalid GitHub owner: $owner" >&2
        return 2
    end

    if not string match -rq '^[A-Za-z0-9_.-]+$' -- $repo
        echo "gcl: invalid repository name: $repo" >&2
        return 2
    end

    set -l url

    if set -q _flag_https
        set url "https://github.com/$owner/$repo.git"
    else
        set -l host

        if set -q _flag_host
            set host $_flag_host
        else
            set host (git config --get gid.sshHost 2>/dev/null)
        end

        if test -z "$host"
            echo "gcl: no SSH host configured." >&2
            echo "Set gid.sshHost or pass --host HOST." >&2
            return 2
        end

        if not string match -rq '^[A-Za-z0-9._-]+$' -- $host
            echo "gcl: invalid SSH host alias: $host" >&2
            return 2
        end

        set url "git@$host:$owner/$repo.git"
    end

    git clone $clone_options $url
end
