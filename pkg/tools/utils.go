package tools

import (
	"os"
	"path/filepath"
	"strings"
)

func AppName() string {
	app := strings.Split(filepath.Base(os.Args[0]), ".")
	if len(app) > 0 {
		return app[0]
	} else {
		return filepath.Base(os.Args[0])
	}
}
