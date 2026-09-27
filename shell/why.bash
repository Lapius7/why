# why: 失敗したコマンドを記録する。~/.bashrc に eval "$(why init bash)" を追記して使う。
__why_dir="${XDG_STATE_HOME:-$HOME/.local/state}/why"
[[ -d $__why_dir ]] || mkdir -p -- "$__why_dir"
__why_last=""

__why_prompt() {
  local code=$? h
  ((code != 0)) || return 0
  h=$(HISTTIMEFORMAT= builtin history 1)
  # 空 Enter では履歴が変わらないので同じコマンドを二重に記録しない
  [[ $h == "$__why_last" ]] && return $code
  __why_last=$h
  [[ $h =~ ^[[:space:]]*[0-9]+[*]?[[:space:]]+(.*)$ ]] || return $code
  local cmd=${BASH_REMATCH[1]}
  [[ $cmd == why || $cmd == "why "* ]] && return $code
  printf '%s\n%s\n%s\n' "$code" "$PWD" "$cmd" > "$__why_dir/last-$$"
  return $code
}

# $? を保ったまま既存の PROMPT_COMMAND より先に動かす
if [[ $(declare -p PROMPT_COMMAND 2>/dev/null) == "declare -a"* ]]; then
  [[ " ${PROMPT_COMMAND[*]} " == *" __why_prompt "* ]] || PROMPT_COMMAND=(__why_prompt "${PROMPT_COMMAND[@]}")
else
  [[ ${PROMPT_COMMAND:-} == *__why_prompt* ]] || PROMPT_COMMAND="__why_prompt${PROMPT_COMMAND:+;$PROMPT_COMMAND}"
fi
