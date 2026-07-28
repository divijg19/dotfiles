function gcl
    if test (count $argv) -ne 2
        echo "Usage: gcl <owner> <repo>"
        return 1
    end
    git clone https://github.com/$argv[1]/$argv[2].git
end
