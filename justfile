# why のタスク

default:
    @just --list

# ビルドして bin/why に出力
build:
    go build -o bin/why ./cmd/why

# テスト（ルールの example 検証を含む）
test:
    go vet ./...
    go test ./...

# インストール（シェルフックの追記も行う）
install: test
    ./install.sh

# アンインストール
uninstall:
    ./install.sh --uninstall

# ルール一覧
rules: build
    ./bin/why rules

# npm パッケージを dist/npm/ に生成して .tgz を作る（例: just npm-pack 0.1.0）
npm-pack version="0.0.0-dev":
    node npm/build.mjs {{version}} --pack

# npm に Trusted Publisher を設定（初回のみ。普通のターミナルで実行）
npm-trust:
    ~/sh/npm-trust.sh Lapius7/why release.yml @lapius/why @lapius/why-linux-x64 @lapius/why-linux-arm64 @lapius/why-darwin-x64 @lapius/why-darwin-arm64
