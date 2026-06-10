package cmd

import (
	"embed"
	"fmt"
	"io/fs"
	"strings"
)

//go:embed skill/*
var skillFS embed.FS

// GetSkillRules liest rekursiv alle Markdown-Dateien aus dem eingebetteten
// skill/ Ordner und gibt sie als formatierten Text zurück.
func GetSkillRules() (string, error) {
	var builder strings.Builder

	err := fs.WalkDir(skillFS, "skill", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// Nur Markdown-Dateien einlesen (.md)
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".md") {
			content, err := skillFS.ReadFile(path)
			if err != nil {
				return fmt.Errorf("error reading embedded file %s: %w", path, err)
			}
			// Wir entfernen den führenden Pfad "skill/" zur besseren Lesbarkeit
			displayPath := strings.TrimPrefix(path, "skill/")
			builder.WriteString(fmt.Sprintf("=== File: %s ===\n\n", displayPath))
			builder.Write(content)
			builder.WriteString("\n\n")
		}
		return nil
	})

	if err != nil {
		return "", err
	}

	return builder.String(), nil
}
