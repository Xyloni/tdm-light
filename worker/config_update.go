package worker

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func updateEnvSetting(filePath, key, value string) error {
	value = strings.TrimSpace(value)
	if value == "" || strings.ContainsAny(value, "\r\n") {
		return fmt.Errorf("название игры пустое или содержит перевод строки")
	}

	data, err := os.ReadFile(filePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("не удалось прочитать config.env: %w", err)
	}
	mode := os.FileMode(0o600)
	if info, statErr := os.Stat(filePath); statErr == nil {
		mode = info.Mode().Perm()
	}

	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	if len(data) == 0 {
		lines = nil
	}
	updated := false
	for i, line := range lines {
		bom := ""
		if strings.HasPrefix(line, "\uFEFF") {
			bom = "\uFEFF"
			line = strings.TrimPrefix(line, bom)
		}
		name, _, ok := strings.Cut(line, "=")
		if ok && strings.TrimSpace(name) == key {
			lines[i] = bom + key + "=" + value
			updated = true
		}
	}
	if !updated {
		lines = append(lines, key+"="+value)
	}
	updatedData := []byte(strings.Join(lines, "\n") + "\n")

	dir := filepath.Dir(filePath)
	tmp, err := os.CreateTemp(dir, ".config-env-*.tmp")
	if err != nil {
		return fmt.Errorf("не удалось создать временный config.env: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return fmt.Errorf("не удалось установить права config.env: %w", err)
	}
	if _, err := tmp.Write(updatedData); err != nil {
		tmp.Close()
		return fmt.Errorf("не удалось записать config.env: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("не удалось сохранить config.env: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("не удалось закрыть config.env: %w", err)
	}
	if err := os.Rename(tmpPath, filePath); err != nil {
		return fmt.Errorf("не удалось обновить config.env: %w", err)
	}
	return nil
}
