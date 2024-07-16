package tools

import (
	"fmt"
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

func GetStatusExt(original, newExt string) string {
	baseName := filepath.Base(original)
	dirName := filepath.Dir(original)
	ext := filepath.Ext(baseName)
	stem := baseName[:len(baseName)-len(ext)]
	statusFile := filepath.Join(dirName, stem+newExt)
	return statusFile
}

func GenerateUniqueFilename(original string, newExt ...string) (string, error) {
	baseName := filepath.Base(original)
	dirName := filepath.Dir(original)
	ext := filepath.Ext(baseName)
	stem := baseName[:len(baseName)-len(ext)]

	// Try the original filename first.
	if _, err := os.Stat(original); os.IsNotExist(err) {
		return original, nil
	}

	if len(newExt) > 0 {
		statusFile := filepath.Join(dirName, stem+newExt[0])
		if _, err := os.Stat(statusFile); os.IsNotExist(err) {
			return original, nil
		}
	}
	// Append numbers until we find a non-existing filename.
	for i := 1; ; i++ {
		newBaseName := fmt.Sprintf("%s_%d%s", stem, i, ext)
		newPath := filepath.Join(dirName, newBaseName)
		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			return newPath, nil
		}
	}
}

func AddUncovered(uncovered map[int64]int64, start, length, N int64) {
	for length > 0 {
		if length > N {
			uncovered[start] = start + N
			start += N
			length -= N
		} else {
			uncovered[start] = start + length
			break
		}
	}
}

func FindUncoveredPositions(total int64, boundaries []int64, limit int64) map[int64]int64 {
	uncovered := make(map[int64]int64)
	if len(boundaries) < 1 {
		AddUncovered(uncovered, 0, total, limit)
		return uncovered
	}

	lastEnd := int64(0) // 总长度的起始位置

	for i := 0; i < len(boundaries); i += 2 {
		// 当前未覆盖区间的起始位置
		if i > 0 {
			// last end
			lastEnd = boundaries[i-1]
		}
		start := boundaries[i]

		// 检查是否有未覆盖区间
		if lastEnd < start {
			AddUncovered(uncovered, lastEnd, start-lastEnd, limit)
		}
	}

	if boundaries[len(boundaries)-1] < total {
		AddUncovered(uncovered, boundaries[len(boundaries)-1], total-boundaries[len(boundaries)-1], limit)
	}

	return uncovered
}
