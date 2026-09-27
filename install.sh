#!/bin/sh
# why のインストーラ
#
#   curl -fsSL https://raw.githubusercontent.com/Lapius7/why/main/install.sh | sh
#                 GitHub Release からビルド済みバイナリを取得（Go 不要）
#   ./install.sh  リポジトリ内から実行し Go があれば、ソースをビルドしてインストール
#
# オプション:
#   --no-hook     シェルの設定ファイルにフックを追記しない
#   --uninstall   バイナリとフックを削除する
# 環境変数:
#   WHY_BIN       インストール先（既定: ~/.local/bin）
#   WHY_VERSION   取得するバージョン（例: v0.1.1。既定: 最新）
set -eu

MODULE=github.com/Lapius7/why
REPO=Lapius7/why
HOOK=1
MODE=install

for a in "$@"; do
	case $a in
	--no-hook) HOOK=0 ;;
	--uninstall) MODE=uninstall ;;
	-h | --help)
		sed -n '2,13p' "$0" 2>/dev/null | sed 's/^# \{0,1\}//'
		exit 0
		;;
	*)
		echo "install.sh: 不明な引数 $a" >&2
		exit 2
		;;
	esac
done

info() { printf '\033[1;32m==>\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m警告:\033[0m %s\n' "$*" >&2; }
die() {
	printf '\033[1;31mエラー:\033[0m %s\n' "$*" >&2
	exit 1
}

# 使っているシェルと、その設定ファイル・フック行（フック行は展開せずそのまま書き込む）
# shellcheck disable=SC2016
detect_shell() {
	case ${SHELL##*/} in
	zsh)
		RC=${ZDOTDIR:-$HOME}/.zshrc
		LINE='command -v why >/dev/null && eval "$(why init zsh)"'
		;;
	bash)
		RC=$HOME/.bashrc
		LINE='command -v why >/dev/null && eval "$(why init bash)"'
		;;
	fish)
		RC=${XDG_CONFIG_HOME:-$HOME/.config}/fish/config.fish
		LINE='command -q why; and why init fish | source'
		;;
	*)
		RC=
		LINE=
		;;
	esac
}

bin_dir() {
	echo "${WHY_BIN:-$HOME/.local/bin}"
}

# GitHub Release から OS/CPU に合うバイナリを取得して $1 に置く
download() {
	case $(uname -s) in
	Linux) os=linux ;;
	Darwin) os=darwin ;;
	*) die "未対応の OS です: $(uname -s)（Linux / macOS のみ）" ;;
	esac
	case $(uname -m) in
	x86_64 | amd64) arch=x64 ;;
	aarch64 | arm64) arch=arm64 ;;
	*) die "未対応の CPU です: $(uname -m)（x86_64 / arm64 のみ）" ;;
	esac
	if [ -n "${WHY_VERSION:-}" ]; then
		url=https://github.com/$REPO/releases/download/$WHY_VERSION/why-$os-$arch.tar.gz
	else
		url=https://github.com/$REPO/releases/latest/download/why-$os-$arch.tar.gz
	fi
	info "ダウンロードします: $url"
	tmp=$(mktemp -d)
	if command -v curl >/dev/null 2>&1; then
		curl -fsSL "$url" -o "$tmp/why.tar.gz" || die "ダウンロードに失敗しました"
	elif command -v wget >/dev/null 2>&1; then
		wget -qO "$tmp/why.tar.gz" "$url" || die "ダウンロードに失敗しました"
	else
		die "curl か wget が必要です"
	fi
	tar -xzf "$tmp/why.tar.gz" -C "$tmp" why
	chmod +x "$tmp/why"
	mv -f "$tmp/why" "$1/why"
	rm -rf "$tmp"
}

uninstall() {
	dir=$(bin_dir)
	if [ -f "$dir/why" ]; then
		rm -f "$dir/why"
		info "削除しました: $dir/why"
	fi
	detect_shell
	if [ -n "$RC" ] && [ -f "$RC" ] && grep -q 'why init' "$RC"; then
		# フック行とその直前のコメント行を消す
		tmp=$(mktemp)
		grep -v -e 'why init' -e '^# why: 失敗したコマンドを記録' "$RC" >"$tmp"
		cat "$tmp" >"$RC"
		rm -f "$tmp"
		info "フックを削除しました: $RC"
	fi
	info "記録ファイルを消すなら: rm -rf ${XDG_STATE_HOME:-$HOME/.local/state}/why"
}

install() {
	dir=$(bin_dir)
	mkdir -p "$dir"
	src=$(cd "$(dirname "$0")" 2>/dev/null && pwd || true)
	if [ -n "$src" ] && [ -f "$src/go.mod" ] && grep -q "^module $MODULE\$" "$src/go.mod" &&
		command -v go >/dev/null 2>&1; then
		info "ソースからビルドします: $src"
		(cd "$src" && GOBIN=$dir go install ./cmd/why)
	else
		download "$dir"
	fi
	info "インストールしました: $dir/why ($("$dir/why" -v))"

	case ":$PATH:" in
	*":$dir:"*) ;;
	*) warn "$dir が PATH に入っていません。設定ファイルに export PATH=\"\$PATH:$dir\" を追記してください" ;;
	esac

	[ "$HOOK" = 1 ] || return 0
	detect_shell
	if [ -z "$RC" ]; then
		warn "未対応のシェルです（${SHELL:-不明}）。why init zsh|bash|fish を参考に手動で設定してください"
		return 0
	fi
	if [ -f "$RC" ] && grep -q 'why init' "$RC"; then
		info "フックは設定済みです: $RC"
	else
		mkdir -p "$(dirname "$RC")"
		printf '\n# why: 失敗したコマンドを記録（why で原因と対処法を表示）\n%s\n' "$LINE" >>"$RC"
		info "フックを追記しました: $RC"
	fi
	info "新しいシェルを開くか exec ${SHELL##*/} で有効になります。試すには: gti status → why"
}

"$MODE"
