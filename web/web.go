package web

import "embed"

// TemplatesFS embeds all HTML templates into the Go binary.
//
//go:embed templates/*.html
var TemplatesFS embed.FS

// StaticFS embeds all static CSS, JS, and font assets into the Go binary.
//
//go:embed static/*
var StaticFS embed.FS
