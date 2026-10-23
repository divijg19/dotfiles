# Internal presentation helpers.

function __gid_heading
    set_color --bold cyan
    echo
    echo "  $argv"
    set_color normal
end

function __gid_field
    set -l label $argv[1]
    set -l value (string join " " -- $argv[2..-1])

    set_color --dim
    printf "  %-13s" "$label"
    set_color normal
    printf "%s\n" "$value"
end

function __gid_result
    set -l kind $argv[1]
    set -l message (string join " " -- $argv[2..-1])

    switch $kind
        case PASS
            set_color --bold green
        case WARN
            set_color --bold yellow
        case FAIL
            set_color --bold red
        case '*'
            set_color normal
    end

    printf "  %-5s" "$kind"
    set_color normal
    printf " %s\n" "$message"
end

function __gid_rule
    set_color --dim
    echo "  "(string repeat -n 42 "─")
    set_color normal
end

function gid --description "Inspect Git identity, signing, and SSH configuration"
    set -l mode info

    if test (count $argv) -gt 1
        echo "Usage: gid [check|ssh|remotes|help]"
        return 2
    end

    if test (count $argv) -eq 1
        set mode $argv[1]
    end

    if not contains -- $mode info check ssh remotes help
        echo "Usage: gid [check|ssh|remotes|help]"
        return 2
    end

    if test "$mode" = help
        set_color --bold
        echo gid
        set_color normal
        echo "  Git identity, signing, and SSH diagnostics"
        echo
        echo "  gid             Inspect effective configuration"
        echo "  gid check       Validate identity, signing, and remotes"
        echo "  gid ssh         Test SSH authentication"
        echo "  gid remotes     Inspect remote URLs"
        echo "  gid help        Show this help"
        return 0
    end

    if not git rev-parse --is-inside-work-tree >/dev/null 2>&1
        __gid_result FAIL "Not inside a Git working tree."
        return 1
    end

    set -l root (git rev-parse --show-toplevel)
    set -l branch (git branch --show-current)
    set -l profile (git config --get gid.profile 2>/dev/null)
    set -l ssh_host (git config --get gid.sshHost 2>/dev/null)
    set -l name (git config --get user.name 2>/dev/null)
    set -l email (git config --get user.email 2>/dev/null)
    set -l signingkey (git config --path --get user.signingkey 2>/dev/null)
    set -l format (git config --get gpg.format 2>/dev/null)
    set -l signcommits (git config --bool --get commit.gpgsign 2>/dev/null)

    # Default command: show effective configuration.
    if test "$mode" = info
        set_color --bold
        echo
        echo "  GID"
        set_color normal
        set_color --dim
        echo "  Git environment"
        set_color normal

        __gid_field Root "$root"
        __gid_field Branch (test -n "$branch"; and echo "$branch"; or echo detached)
        __gid_field Profile (test -n "$profile"; and echo "$profile"; or echo unmapped)
        __gid_field "SSH alias" (test -n "$ssh_host"; and echo "$ssh_host"; or echo unset)

        __gid_heading Identity
        __gid_field Name (test -n "$name"; and echo "$name"; or echo unset)
        __gid_field Email (test -n "$email"; and echo "$email"; or echo unset)

        __gid_heading Signing
        __gid_field Key (test -n "$signingkey"; and echo "$signingkey"; or echo unset)
        __gid_field Format (test -n "$format"; and echo "$format"; or echo unset)
        __gid_field Auto-sign (test -n "$signcommits"; and echo "$signcommits"; or echo unset)

        __gid_heading Configuration origins
        for key in gid.profile gid.sshHost user.name user.email user.signingkey gpg.format commit.gpgsign
            set -l origin (git config --show-origin --show-scope --get "$key" 2>/dev/null)

            if test -n "$origin"
                set_color --dim
                printf "  %-20s" "$key"
                set_color normal
                printf "%s\n" "$origin"
            else
                set_color --dim
                printf "  %-20s" "$key"
                set_color normal
                echo "Not configured"
            end
        end

        __gid_heading Remotes
        set -l remotes (git remote)

        if test (count $remotes) -eq 0
            __gid_result WARN "No remotes configured."
        else
            for remote in $remotes
                set_color --bold
                echo "  $remote"
                set_color normal

                for url in (git remote get-url --all "$remote" 2>/dev/null)
                    __gid_field Fetch "$url"
                end

                for url in (git remote get-url --push --all "$remote" 2>/dev/null)
                    __gid_field Push "$url"
                end
            end
        end

        if test -z "$profile"
            __gid_result WARN "gid.profile is not configured."
        end

        if test -z "$ssh_host"
            __gid_result WARN "gid.sshHost is not configured."
        end

        echo
        return 0
    end

    # Inspect remotes independently of profile validation.
    if test "$mode" = remotes
        __gid_heading "Remote URLs"
        __gid_field Repository "$root"

        set -l remotes (git remote)
        set -l failed 0

        if test (count $remotes) -eq 0
            __gid_result WARN "No remotes configured."
            return 0
        end

        for remote in $remotes
            echo
            set_color --bold
            echo "  $remote"
            set_color normal

            for direction in fetch push
                set -l urls

                if test "$direction" = fetch
                    set urls (git remote get-url --all "$remote" 2>/dev/null)
                else
                    set urls (git remote get-url --push --all "$remote" 2>/dev/null)
                end

                for url in $urls
                    __gid_field "$direction" "$url"

                    if string match -rq 'github\.com|github-' -- "$url"
                        if test -z "$ssh_host"
                            __gid_result WARN "Cannot validate alias; gid.sshHost is unset."
                        else if not string match -q "git@$ssh_host:*" -- "$url"; and not string match -q "ssh://git@$ssh_host/*" -- "$url"
                            __gid_result FAIL "$remote $direction does not use '$ssh_host'."
                            set failed 1
                        end
                    end
                end
            end
        end

        echo
        if test $failed -eq 0
            __gid_result PASS "Remote inspection complete."
            return 0
        end

        __gid_result FAIL "Remote mismatches found."
        return 1
    end

    # SSH connectivity requires an explicit local profile mapping.
    if test "$mode" = ssh
        if test -z "$profile"; or test -z "$ssh_host"
            __gid_result FAIL "Configure gid.profile and gid.sshHost first."
            return 1
        end

        __gid_heading "SSH authentication"
        __gid_field Profile "$profile"
        __gid_field Host "$ssh_host"
        echo

        set -l output (command ssh -T -o ConnectTimeout=8 "git@$ssh_host" 2>&1)
        set -l ssh_status $status

        if test -n "$output"
            printf '%s\n' $output
        end

        if string match -rq "^Hi .*! You've successfully authenticated" -- "$output"
            __gid_result PASS "GitHub SSH authentication succeeded."
            return 0
        end

        __gid_result FAIL "Authentication not confirmed (exit $ssh_status)."
        return 1
    end

    # Validate configuration without changing it.
    __gid_heading "Configuration check"
    __gid_field Profile (test -n "$profile"; and echo "$profile"; or echo unmapped)

    set -l failed 0

    if test -z "$profile"
        __gid_result FAIL "gid.profile is not configured."
        set failed 1
    end

    if test -z "$ssh_host"
        __gid_result FAIL "gid.sshHost is not configured."
        set failed 1
    end

    if test -z "$name"
        __gid_result FAIL "user.name is unset."
        set failed 1
    else
        __gid_result PASS "Author name is configured."
    end

    if test -z "$email"
        __gid_result FAIL "user.email is unset."
        set failed 1
    else
        __gid_result PASS "Author email is configured."
    end

    if test "$format" != ssh
        __gid_result FAIL "gpg.format must be ssh."
        set failed 1
    else
        __gid_result PASS "SSH signing format is enabled."
    end

    if test "$signcommits" != true
        __gid_result FAIL "commit.gpgsign must be true."
        set failed 1
    else
        __gid_result PASS "Automatic commit signing is enabled."
    end

    if test -z "$signingkey"
        __gid_result FAIL "user.signingkey is unset."
        set failed 1
    else if not test -f "$signingkey"
        __gid_result FAIL "Signing key file does not exist: $signingkey"
        set failed 1
    else
        __gid_result PASS "Signing key file exists."
    end

    set -l remotes (git remote)

    if test (count $remotes) -eq 0
        __gid_result WARN "No remotes configured."
    else if test -n "$ssh_host"
        for remote in $remotes
            for direction in fetch push
                set -l urls

                if test "$direction" = fetch
                    set urls (git remote get-url --all "$remote" 2>/dev/null)
                else
                    set urls (git remote get-url --push --all "$remote" 2>/dev/null)
                end

                for url in $urls
                    if string match -rq 'github\.com|github-' -- "$url"
                        if not string match -q "git@$ssh_host:*" -- "$url"; and not string match -q "ssh://git@$ssh_host/*" -- "$url"
                            __gid_result FAIL "$remote $direction does not use '$ssh_host': $url"
                            set failed 1
                        end
                    end
                end
            end
        end
    end

    echo
    if test $failed -ne 0
        __gid_result FAIL "Configuration check failed."
        return 1
    end

    __gid_result PASS "Configuration check passed."
end
