# why

失敗したコマンドの**原因と対処法を日本語で表示する** CLI。オフラインで動作し、AI は使わない。

```
$ git push
fatal: not a git repository (or any of the parent directories): .git
$ why
✗ git push  (終了コード 128)
  ~/work · たった今
  [128] 致命的エラー（git の fatal など）、または exit に不正な値が渡された

原因  今いるディレクトリは git リポジトリではない
対処  → リポジトリのディレクトリに移動する（pwd で現在地を確認）
      → 新しく始めるなら: git init
```

## インストール

Go が必要。バイナリの配置と、シェルの設定ファイルへのフック追記を一度に行う。

```sh
curl -fsSL https://raw.githubusercontent.com/Lapius7/why/main/install.sh | sh
```

リポジトリを clone 済みなら `./install.sh`（または `just install`）。オプションは `--no-hook`（フックを追記しない）と `--uninstall`（削除）。

Go だけで入れる場合は `go install github.com/Lapius7/why/cmd/why@latest` を実行し、フックは次を手動で追記する（失敗したコマンドを記録するだけで、成功時は何もしない）。

| シェル | 追記する内容 |
|-|-|
| zsh (`~/.zshrc`) | `eval "$(why init zsh)"` |
| bash (`~/.bashrc`) | `eval "$(why init bash)"` |
| fish (`~/.config/fish/config.fish`) | `why init fish \| source` |

## 使い方

| コマンド | 内容 |
|-|-|
| `why` | 直前に失敗したコマンドを説明 |
| `why -r` | 直前のコマンドを確認後に再実行し、出力も含めて説明 |
| `why -- <cmd>` | コマンドを実行し、失敗したら説明（終了コードは引き継ぐ） |
| `<cmd> 2>&1 \| why` | パイプで渡した出力を説明 |
| `why 137` | 終了コードの意味を説明 |
| `why rules` | 読み込まれているルールの一覧 |

### エラー出力の取得

シェルは stderr を保存しないため、フックで分かるのは「コマンド・終了コード・場所」だけ。出力は次の方法で補う。

- **tmux 内**: 画面の履歴から直前のコマンドの出力を自動で取り込む
- **`why -r`**: 再実行して取り込む。`rm` / `git push` / `sudo` / リダイレクトなど副作用がありそうなものは拒否する（`--force` で強制）
- **パイプ / `why --`**: 最初から取り込む

再実行時はメッセージを英語（`LC_MESSAGES=C.UTF-8`）にしてルールと照合しやすくする。主なルールは日本語メッセージにも対応している。

## ルール

組み込みルールは `rules/*.yaml`。`~/.config/why/rules/*.yaml` に置くと追加でき、同じ `id` なら組み込みを上書きする。

```yaml
- id: my-port
  prog: 'node|npm'            # プログラム名（正規表現・完全一致）
  cmd: 'run dev'              # コマンドライン全体（正規表現）
  code: [1]                   # 終了コード
  output: 'EADDRINUSE.*?:(?P<port>\d+)'   # 出力（正規表現、(?m) 付き）
  example: "Error: listen EADDRINUSE: address already in use :::3000"
  priority: 1                 # 表示順の補正（任意）
  cause: "ポート {{port}} が使用中"
  fix:
    - "ss -ltnp 'sport = :{{port}}'"
```

- 指定した条件がすべて一致したときに表示される。スコアは output=2、その他=各1 に priority を足したもので、上位 3 件を表示する
- `output` の名前付きグループ、および `{{prog}}` `{{cmd}}` `{{code}}` を `cause` / `fix` で使える
- 値が空の変数を含む `fix` 行は表示しない
- `output` を持つ組み込みルールには `example` が必須で、`go test` で一致を検証する

## 記録ファイル

`${XDG_STATE_HOME:-~/.local/state}/why/last-<シェルのPID>`。内容は終了コード・cwd・コマンドの 3 つ。7 日より古いものは自動で削除する。
