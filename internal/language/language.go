// Package language turns an editor mode or a file name into a short language id.
package language

import (
	"path/filepath"
	"strings"
)

// Resolve prefers the editor's language name. Empty and generic modes fall
// back to the file extension.
func Resolve(editorLang, path string) string {
	lang := strings.ToLower(strings.TrimSpace(editorLang))
	lang = strings.TrimSuffix(lang, "-mode")
	lang = strings.TrimSuffix(lang, "-ts")
	switch lang {
	case "", "fundamental", "text", "conf", "normal", "special":
		return fromExt(path)
	default:
		return lang
	}
}

func fromExt(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == "" {
		base := strings.ToLower(filepath.Base(path))
		switch base {
		case "makefile", "gnumakefile":
			return "make"
		case "dockerfile":
			return "docker"
		default:
			return "unknown"
		}
	}
	if lang, ok := byExt[ext]; ok {
		return lang
	}
	return strings.TrimPrefix(ext, ".")
}

var byExt = map[string]string{
	".go":       "go",
	".py":       "python",
	".pyi":      "python",
	".rs":       "rust",
	".c":        "c",
	".h":        "c",
	".cc":       "cpp",
	".cpp":      "cpp",
	".hpp":      "cpp",
	".hh":       "cpp",
	".java":     "java",
	".js":       "javascript",
	".jsx":      "javascript",
	".mjs":      "javascript",
	".ts":       "typescript",
	".tsx":      "typescript",
	".rb":       "ruby",
	".php":      "php",
	".lua":      "lua",
	".vim":      "vim",
	".el":       "emacs-lisp",
	".clj":      "clojure",
	".hs":       "haskell",
	".ml":       "ocaml",
	".sh":       "shell",
	".bash":     "shell",
	".zsh":      "shell",
	".fish":     "fish",
	".md":       "markdown",
	".markdown": "markdown",
	".org":      "org",
	".txt":      "text",
	".json":     "json",
	".yaml":     "yaml",
	".yml":      "yaml",
	".toml":     "toml",
	".xml":      "xml",
	".html":     "html",
	".htm":      "html",
	".css":      "css",
	".scss":     "scss",
	".sql":      "sql",
	".mod":      "go",
	".sum":      "go",
	".swift":    "swift",
	".kt":       "kotlin",
	".kts":      "kotlin",
	".r":        "r",
	".dart":     "dart",
	".ex":       "elixir",
	".exs":      "elixir",
	".erl":      "erlang",
	".zig":      "zig",
	".nim":      "nim",
	".pl":       "perl",
	".pm":       "perl",
}
