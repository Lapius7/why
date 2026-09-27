# why: 失敗したコマンドを記録する。~/.config/fish/config.fish に why init fish | source を追記して使う。
if set -q XDG_STATE_HOME
    set -g __why_dir $XDG_STATE_HOME/why
else
    set -g __why_dir $HOME/.local/state/why
end
test -d $__why_dir; or mkdir -p $__why_dir

function __why_postexec --on-event fish_postexec
    set -l code $status
    test $code -ne 0; or return
    test -n "$argv[1]"; or return
    string match -qr '^why(\s|$)' -- $argv[1]; and return
    printf '%s\n%s\n%s\n' $code $PWD $argv[1] >$__why_dir/last-$fish_pid
end
