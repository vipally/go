package vendor

import (
	"path/filepath"
	"runtime"
)

func GetPackagePath() string {
	depth := 0
	if _, __file, _, __ok := runtime.Caller(depth); __ok {
		thisFilePath := filepath.Dir(__file)
		return thisFilePath
	}
	return ""
}
