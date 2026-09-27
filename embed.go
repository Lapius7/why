// Package why は組み込みルールとシェル用フックを埋め込む。
package why

import "embed"

//go:embed rules/*.yaml
var Rules embed.FS

//go:embed shell/*
var Shell embed.FS
