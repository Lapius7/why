// Package exitcode は終了コードとシグナルの意味を返す。
package exitcode

import "fmt"

var signals = map[int][2]string{
	1:  {"SIGHUP", "端末が閉じられた・SSH が切断された"},
	2:  {"SIGINT", "Ctrl+C で中断された"},
	3:  {"SIGQUIT", "Ctrl+\\ で中断された"},
	4:  {"SIGILL", "不正な CPU 命令（CPU 非対応のバイナリなど）"},
	5:  {"SIGTRAP", "デバッガのブレークポイント・トラップ"},
	6:  {"SIGABRT", "プログラム自身が abort した（assert 失敗・内部エラー・メモリ破壊）"},
	7:  {"SIGBUS", "バスエラー（mmap 中のファイルが縮んだ・メモリ整列違反）"},
	8:  {"SIGFPE", "算術エラー（整数のゼロ除算など）"},
	9:  {"SIGKILL", "強制終了された（多くはメモリ不足による OOM Killer、または kill -9）"},
	10: {"SIGUSR1", "ユーザー定義シグナル 1"},
	11: {"SIGSEGV", "セグメンテーション違反（不正なメモリアクセス）"},
	12: {"SIGUSR2", "ユーザー定義シグナル 2"},
	13: {"SIGPIPE", "出力先のパイプが閉じられた（| head などの後ろなら正常）"},
	14: {"SIGALRM", "タイマー（alarm）による終了"},
	15: {"SIGTERM", "終了要求を受けた（kill・docker stop・systemctl stop など）"},
	24: {"SIGXCPU", "CPU 時間の上限（ulimit -t）を超えた"},
	25: {"SIGXFSZ", "ファイルサイズの上限（ulimit -f）を超えた"},
}

// Describe は終了コードの一般的な意味を返す。
func Describe(code int) string {
	switch {
	case code == 0:
		return "成功"
	case code == 1:
		return "一般的なエラー（詳細はコマンドの出力を参照）"
	case code == 2:
		return "使い方の誤り（引数・オプション・構文）であることが多い"
	case code == 124:
		return "タイムアウト（timeout コマンドによる打ち切り）"
	case code == 126:
		return "コマンドは見つかったが実行できない（権限・形式）"
	case code == 127:
		return "コマンドが見つからない"
	case code == 128:
		return "致命的エラー（git の fatal など）、または exit に不正な値が渡された"
	case code > 128 && code <= 128+64:
		n := code - 128
		if s, ok := signals[n]; ok {
			return fmt.Sprintf("シグナル %d (%s): %s", n, s[0], s[1])
		}
		return fmt.Sprintf("シグナル %d で終了した", n)
	case code == 255:
		return "範囲外の終了コード（exit -1、ssh の接続失敗など）"
	case code < 0 || code > 255:
		return "範囲外の終了コード"
	}
	return "プログラム固有のエラーコード"
}
