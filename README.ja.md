<div align="center">

<p align="center"><a href="README.md">English</a> | <b>日本語</b></p>

<h1 align="center">why</h1>

<h3 align="center">失敗したコマンドの原因と対処法を、日本語で表示します。</h3>

<p align="center">直前に失敗したコマンドの理由を説明する CLI です。オフラインで動き、AI は使いません。</p>

<p align="center">
  <a href="https://www.npmjs.com/package/@lapius/why"><img alt="npm" src="https://img.shields.io/npm/v/@lapius/why?style=flat-square&color=cb3837&logo=npm"></a>
  <a href="https://github.com/Lapius7/why/actions/workflows/release.yml"><img alt="Release workflow" src="https://img.shields.io/github/actions/workflow/status/Lapius7/why/release.yml?label=release&style=flat-square&logo=githubactions&logoColor=white"></a>
  <a href="https://go.dev"><img alt="Go" src="https://img.shields.io/badge/lang-Go-00add8?logo=go&logoColor=white&style=flat-square"></a>
  <a href="LICENSE"><img alt="License" src="https://img.shields.io/github/license/Lapius7/why?style=flat-square&color=97ca00"></a>
  <a href="https://github.com/Lapius7/why/stargazers"><img alt="Stars" src="https://img.shields.io/github/stars/Lapius7/why?style=flat-square&color=e3b341&logo=github"></a>
  <a href="https://github.com/Lapius7/why/commits/main"><img alt="Last commit" src="https://img.shields.io/github/last-commit/Lapius7/why?style=flat-square&color=3b82f6"></a>
</p>

<p align="center"><a href="#最短セットアップ">最短セットアップ</a> · <a href="#使い方">使い方</a> · <a href="#ルール">ルール</a> · <a href="#開発">開発</a></p>

</div>

---

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

## 特長

- **オフラインで、AI は使わない**: 終了コード、コマンド、出力を YAML のルールと照合します。
- **自分のルールを足せる**: `~/.config/why/rules/*.yaml` に置きます。同じ `id` なら組み込みのルールを上書きします。
- **シェルのフック**: zsh、bash、fish に対応します。フックは失敗したコマンドを記録するだけです。
- **出力の取り込み**: tmux の履歴、`why -r`、パイプのどれかで出力を取ります。
- **組み込みルール**: build、docker、generic、git、node、python、system のエラーを扱います。

## 最短セットアップ

### pnpm（推奨）

```sh
pnpm add -g @lapius/why
```

> [!NOTE]
> このパッケージは pnpm 前提です。npm と pnpm の両方に入れると同じコマンドが二重になるので、`pnpm add -g` だけを使ってください。

Linux / macOS（x64・arm64）用のビルド済みバイナリが入ります。Go は要りません。入れたあと、下の表のフックをシェルの設定ファイルに追記します。初回の起動で自分自身をネイティブバイナリに置き換えるので、2 回目からは node を通りません。

### インストーラ（npm も Go も不要）

GitHub Release からビルド済みバイナリを `~/.local/bin` に取得し、シェルの設定ファイルへのフック追記まで行います。

```sh
curl -fsSL https://raw.githubusercontent.com/Lapius7/why/main/install.sh | sh
```

オプションは `--no-hook`（フックを追記しない）と `--uninstall`（削除）です。インストール先は `WHY_BIN`、版は `WHY_VERSION=v0.1.1` で指定します。

### ソースから

`go install github.com/Lapius7/why/cmd/why@latest`。clone 済みなら、`./install.sh`（または `just install`）でソースからビルドし、フックも追記します。

### フック

| シェル | 追記する内容 |
|---|---|
| zsh (`~/.zshrc`) | `eval "$(why init zsh)"` |
| bash (`~/.bashrc`) | `eval "$(why init bash)"` |
| fish (`~/.config/fish/config.fish`) | `why init fish \| source` |

## 使い方

| コマンド | 内容 |
|---|---|
| `why` | 直前に失敗したコマンドを説明 |
| `why -r` | 直前のコマンドを確認後に再実行し、出力も含めて説明 |
| `why -- <cmd>` | コマンドを実行し、失敗したら説明（終了コードは引き継ぐ） |
| `<cmd> 2>&1 \| why` | パイプで渡した出力を説明 |
| `why 137` | 終了コードの意味を説明 |
| `why rules` | 読み込まれているルールの一覧 |

### エラー出力の取得

シェルは stderr を保存しないので、フックで分かるのは「コマンド・終了コード・場所」だけです。出力は次のどれかで取ります。

- **tmux 内**: 画面の履歴から、直前のコマンドの出力を取り込みます。
- **`why -r`**: 再実行して取り込みます。`rm`、`git push`、`sudo`、リダイレクトなど副作用がありそうなものは拒否します（`--force` で強制）。
- **パイプ / `why --`**: 最初から取り込みます。

再実行ではメッセージを英語（`LC_MESSAGES=C.UTF-8`）にして、ルールと照合しやすくします。主なルールは日本語のメッセージにも対応しています。

## ルール

組み込みのルールは `rules/*.yaml` にあります。`~/.config/why/rules/*.yaml` に置くと追加でき、同じ `id` なら組み込みを上書きします。

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

- 指定した条件がすべて一致したときに表示します。スコアは `output` が 2、ほかは各 1 に `priority` を足したもので、上位 3 件を表示します。
- `output` の名前付きグループと、`{{prog}}` `{{cmd}}` `{{code}}` を `cause` / `fix` で使えます。
- 値が空の変数を含む `fix` 行は表示しません。
- `output` を持つ組み込みルールには `example` が必須で、`go test` で一致を検証します。

### 記録ファイル

`${XDG_STATE_HOME:-~/.local/state}/why/last-<シェルのPID>` に、終了コード・cwd・コマンドの 3 つを記録します。7 日より古いものは自動で削除します。

## 安全のために

> [!NOTE]
> 未検証: npm とインストーラの経路、シェルのフック、tmux からの取り込み。このソースからビルドしたバイナリに、パイプで出力を渡して 1 回実行し、上の原因と対処が出ることは確かめました。ビルド済みパッケージは Windows 向けではありません。

## 開発

```sh
go test ./...
just npm-pack   # dist/npm/ に .tgz を作る
```

## リリース

`v*` タグを push すると、GitHub Actions がバイナリをビルドし、npm（`@lapius/why` と OS/CPU 別の `@lapius/why-<os>-<cpu>`）へ公開して、GitHub Release に添付します。

```sh
git tag v0.1.0 && git push origin v0.1.0
```

## ライセンス

[MIT](LICENSE)
