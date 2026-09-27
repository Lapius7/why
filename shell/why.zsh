# why: 失敗したコマンドを記録する。~/.zshrc に eval "$(why init zsh)" を追記して使う。
typeset -g __why_cmd=""
typeset -g __why_dir="${XDG_STATE_HOME:-$HOME/.local/state}/why"
[[ -d $__why_dir ]] || mkdir -p -- "$__why_dir"

__why_preexec() { __why_cmd=$1 }

__why_precmd() {
  local code=$?
  [[ -n $__why_cmd ]] || return 0
  local cmd=$__why_cmd
  __why_cmd=""
  (( code != 0 )) || return 0
  [[ $cmd == why || $cmd == "why "* ]] && return 0
  print -r -- "$code"$'\n'"$PWD"$'\n'"$cmd" >| "$__why_dir/last-$$"
}

autoload -Uz add-zsh-hook
add-zsh-hook preexec __why_preexec
# $? を他のフックに変えられる前に読むため先頭に入れる
if (( ${precmd_functions[(I)__why_precmd]:-0} == 0 )); then
  precmd_functions=(__why_precmd $precmd_functions)
fi
