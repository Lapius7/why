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
