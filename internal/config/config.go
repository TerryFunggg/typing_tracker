// Package config reads the local typing-tracker settings.
package config

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	DefaultPort = 57147
	DefaultBind = "127.0.0.1"
	DefaultGap  = 120
)

// Config is the on-disk configuration. Bind must be a loopback address.
type Config struct {
	Port       int
	Bind       string
	GapSeconds int
	Ignore     []string
	Dir        string
	DataDir    string
}

// Load reads the config directory. When create is true and files are missing,
// it writes config.toml and a token. Existing files are left as they are.
func Load(create bool) (Config, error) {
	dir, err := Dir()
	if err != nil {
		return Config{}, err
	}
	data, err := DataDir()
	if err != nil {
		return Config{}, err
	}
	cfg := Config{
		Port:       DefaultPort,
		Bind:       DefaultBind,
		GapSeconds: DefaultGap,
		Dir:        dir,
		DataDir:    data,
	}
	path := filepath.Join(dir, "config.toml")
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		if !create {
			return cfg, nil
		}
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return Config{}, err
		}
		if err := os.WriteFile(path, []byte(defaultFile()), 0o600); err != nil {
			return Config{}, err
		}
	} else if err != nil {
		return Config{}, err
	} else {
		parsed, err := Parse(string(raw))
		if err != nil {
			return Config{}, fmt.Errorf("config %s: %w", path, err)
		}
		cfg.Port = parsed.Port
		cfg.Bind = parsed.Bind
		cfg.GapSeconds = parsed.GapSeconds
		cfg.Ignore = parsed.Ignore
	}
	if cfg.Port < 1 || cfg.Port > 65535 {
		return Config{}, fmt.Errorf("config port %d is outside 1-65535", cfg.Port)
	}
	if cfg.GapSeconds < 1 {
		return Config{}, fmt.Errorf("config gap_seconds must be at least 1")
	}
	if create {
		if err := ensureToken(dir); err != nil {
			return Config{}, err
		}
	}
	return cfg, nil
}

// Token reads the bearer token. The file is created by tt serve.
func Token(dir string) (string, error) {
	raw, err := os.ReadFile(filepath.Join(dir, "token"))
	if err != nil {
		return "", err
	}
	token := strings.TrimSpace(string(raw))
	if token == "" {
		return "", fmt.Errorf("token file is empty")
	}
	return token, nil
}

// DBPath is the SQLite file inside the data directory.
func (c Config) DBPath() string {
	return filepath.Join(c.DataDir, "tracker.db")
}

// Parse reads the small TOML subset this program writes.
func Parse(raw string) (Config, error) {
	cfg := Config{
		Port:       DefaultPort,
		Bind:       DefaultBind,
		GapSeconds: DefaultGap,
	}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return Config{}, fmt.Errorf("expected key = value, got %q", line)
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		switch key {
		case "port":
			n, err := strconv.Atoi(value)
			if err != nil {
				return Config{}, fmt.Errorf("port: %w", err)
			}
			cfg.Port = n
		case "bind":
			cfg.Bind = strings.Trim(value, `"`)
		case "gap_seconds":
			n, err := strconv.Atoi(value)
			if err != nil {
				return Config{}, fmt.Errorf("gap_seconds: %w", err)
			}
			cfg.GapSeconds = n
		case "ignore":
			var paths []string
			if err := json.Unmarshal([]byte(value), &paths); err != nil {
				return Config{}, fmt.Errorf("ignore: %w", err)
			}
			for _, p := range paths {
				if p == "" || !filepath.IsAbs(p) {
					return Config{}, fmt.Errorf("ignore entry %q must be an absolute path", p)
				}
				cfg.Ignore = append(cfg.Ignore, filepath.Clean(p))
			}
		default:
			return Config{}, fmt.Errorf("unknown setting %q", key)
		}
	}
	return cfg, nil
}

// Ignored reports whether path is the prefix itself or a file inside it.
func Ignored(path string, prefixes []string) bool {
	clean := filepath.Clean(path)
	for _, prefix := range prefixes {
		prefix = filepath.Clean(prefix)
		if clean == prefix {
			return true
		}
		sep := string(filepath.Separator)
		if strings.HasPrefix(clean, prefix+sep) {
			return true
		}
	}
	return false
}

func Dir() (string, error) {
	return xdg("XDG_CONFIG_HOME", ".config")
}

func DataDir() (string, error) {
	return xdg("XDG_DATA_HOME", ".local/share")
}

func xdg(env, fallback string) (string, error) {
	base := os.Getenv(env)
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, fallback)
	}
	return filepath.Join(base, "typing-tracker"), nil
}

func ensureToken(dir string) error {
	path := filepath.Join(dir, "token")
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(hex.EncodeToString(buf)+"\n"), 0o600)
}

func defaultFile() string {
	return `# typing-tracker stores coding time on this machine only.
# tt serve creates this file. Restart tt serve after you edit it.

port = 57147
bind = "127.0.0.1"

# A gap longer than this, in seconds, ends a coding session.
gap_seconds = 120

# Absolute directory prefixes to drop. One line, JSON strings.
# ignore = ["/home/you/notes"]
ignore = []
`
}
