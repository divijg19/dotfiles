function gn
    if test (count $argv) -ne 1
        echo "Usage: gn <directory>"
        return 1
    end

    mkdir -p $argv
    and cd $argv
    and git init
end
