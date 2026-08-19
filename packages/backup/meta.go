package backup

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type metaFile struct {
	Driver string `json:"driver"`
}

func metaPath(backupFile string) string {
	return backupFile + ".json"
}

func writeMeta(backupFile, driver string) error {
	raw, err := json.Marshal(metaFile{Driver: driver})
	if err != nil {
		return err
	}
	return os.WriteFile(metaPath(backupFile), raw, 0o644)
}

func readMeta(backupFile string) string {
	raw, err := os.ReadFile(metaPath(backupFile))
	if err != nil {
		// also try without double extension quirks
		base := strings.TrimSuffix(backupFile, filepath.Ext(backupFile))
		raw, err = os.ReadFile(base + ".json")
		if err != nil {
			return ""
		}
	}
	var m metaFile
	if json.Unmarshal(raw, &m) != nil {
		return ""
	}
	return m.Driver
}
