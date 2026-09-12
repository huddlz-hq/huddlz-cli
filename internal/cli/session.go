package cli

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

var errNoSession = errors.New("not logged in to this server; run 'huddlz auth login --email <email>'")

func sessionPath(server *url.URL) (string, error) {
	root, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("could not locate user configuration directory")
	}
	key := sha256.Sum256([]byte(server.String()))
	return filepath.Join(root, "huddlz", "sessions", fmt.Sprintf("%x", key)), nil
}

func saveSession(server *url.URL, token string) error {
	path, err := sessionPath(server)
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("could not create session directory")
	}
	file, err := os.CreateTemp(dir, ".session-")
	if err != nil {
		return fmt.Errorf("could not create session file")
	}
	defer os.Remove(file.Name())
	_, writeErr := file.WriteString(token)
	closeErr := file.Close()
	if writeErr != nil || closeErr != nil {
		return fmt.Errorf("could not write session file")
	}
	if err := os.Rename(file.Name(), path); err != nil {
		return fmt.Errorf("could not replace session file")
	}
	return nil
}

func loadSession(server *url.URL) (string, error) {
	path, err := sessionPath(server)
	if err != nil {
		return "", err
	}
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return "", errNoSession
	}
	if err != nil || !info.Mode().IsRegular() || (runtime.GOOS != "windows" && info.Mode().Perm()&0077 != 0) || info.Size() > 64<<10 {
		return "", fmt.Errorf("session file must be a user-only regular file")
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 || strings.ContainsAny(string(data), "\r\n\t ") {
		return "", fmt.Errorf("could not read a valid saved session; log in again")
	}
	return string(data), nil
}

func removeSession(server *url.URL) error {
	path, err := sessionPath(server)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("could not remove saved session")
	}
	return nil
}
