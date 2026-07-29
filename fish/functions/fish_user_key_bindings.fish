function fish_user_key_bindings
    fish_vi_key_bindings

    bind -M insert ctrl-backspace backward-kill-word
    bind ctrl-backspace backward-kill-word
end
