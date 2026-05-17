function cat
    if isatty stdout
        bat -P $argv
    else
        command cat $argv
    end
end
