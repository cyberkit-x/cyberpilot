package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

type Paths struct {
	ConfigDir    string
	DataDir      string
	ConfigFile   string
	DatabaseFile string
	ArtifactsDir string
	RuntimeDir   string
	LockFile     string
	Endpoint     string
}

func ResolvePaths() (Paths, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return Paths{}, fmt.Errorf("resolve user config directory: %w", err)
	}
	dataDir, err := userDataDir()
	if err != nil {
		return Paths{}, err
	}
	configDir = filepath.Join(configDir, "cyberpilot")
	dataDir = filepath.Join(dataDir, "cyberpilot")
	runtimeDir := filepath.Join(dataDir, "run")
	endpoint := filepath.Join(runtimeDir, "cyberpilot.sock")
	if runtime.GOOS == "windows" {
		endpoint = `\\.\pipe\cyberpilot-` + currentUserKey()
	}
	return Paths{
		ConfigDir: configDir, DataDir: dataDir,
		ConfigFile:   filepath.Join(configDir, "config.yaml"),
		DatabaseFile: filepath.Join(dataDir, "cyberpilot.db"),
		ArtifactsDir: filepath.Join(dataDir, "artifacts"),
		RuntimeDir:   runtimeDir, LockFile: filepath.Join(runtimeDir, "daemon.lock"), Endpoint: endpoint,
	}, nil
}

func Ensure(paths Paths) error {
	for _, dir := range []string{paths.ConfigDir, paths.DataDir, paths.ArtifactsDir, paths.RuntimeDir} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("create %s: %w", dir, err)
		}
		if runtime.GOOS != "windows" {
			if err := os.Chmod(dir, 0o700); err != nil {
				return fmt.Errorf("protect %s: %w", dir, err)
			}
		}
	}
	return nil
}

func userDataDir() (string, error) {
	if runtime.GOOS == "windows" {
		if value := os.Getenv("LOCALAPPDATA"); value != "" {
			return value, nil
		}
	}
	if value := os.Getenv("XDG_DATA_HOME"); value != "" && runtime.GOOS != "darwin" {
		return value, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user data directory: %w", err)
	}
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support"), nil
	case "windows":
		return filepath.Join(home, "AppData", "Local"), nil
	default:
		return filepath.Join(home, ".local", "share"), nil
	}
}

func currentUserKey() string {
	if value := os.Getenv("USERNAME"); value != "" {
		return sanitizeKey(value)
	}
	if value := os.Getenv("USER"); value != "" {
		return sanitizeKey(value)
	}
	return "user"
}

func sanitizeKey(value string) string {
	result := make([]byte, 0, len(value))
	for i := range len(value) {
		c := value[i]
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_' {
			result = append(result, c)
		}
	}
	if len(result) == 0 {
		return "user"
	}
	return string(result)
}
