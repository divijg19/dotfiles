function gid
    echo
    echo "Path     : "(pwd)

    if git rev-parse --is-inside-work-tree >/dev/null 2>&1
        echo "Author   : "(git config user.name)
        echo "Email    : "(git config user.email)
        echo "Remote   : "(git remote get-url origin 2>/dev/null)
        echo "Branch   : "(git branch --show-current)
    else
        echo "Not inside a git repository."
    end
end
