package features_test

import (
	"os"
	"path/filepath"
)

func testEnvironment(server, home string) []string {
	return []string{"HUDDLZ_URL=" + server, "HOME=" + home, "USERPROFILE=" + home, "XDG_CONFIG_HOME=" + filepath.Join(home, "config"), "APPDATA=" + filepath.Join(home, "appdata"), "PATH=" + os.Getenv("PATH")}
}
