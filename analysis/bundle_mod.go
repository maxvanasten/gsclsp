package analysis

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type modManifest struct {
	Name        string `json:"name"`
	Author      string `json:"author"`
	Description string `json:"description"`
	Version     string `json:"version"`
}

func BundleModForURI(uri string) (string, error) {
	sourcePath := uriToPath(uri)
	if sourcePath == "" {
		return "", fmt.Errorf("invalid document uri")
	}
	sourceRoot := filepath.Dir(sourcePath)
	modName := filepath.Base(sourceRoot)
	if modName == "" || modName == "." || modName == string(filepath.Separator) {
		return "", fmt.Errorf("failed to determine mod name from %q", sourceRoot)
	}

	destRoot := filepath.Join(sourceRoot, modName)
	if err := validateBundleDestination(sourceRoot, destRoot); err != nil {
		return "", err
	}

	if err := os.RemoveAll(destRoot); err != nil {
		return "", fmt.Errorf("remove existing bundle dir: %w", err)
	}

	scriptsRoot := filepath.Join(destRoot, "scripts")
	if err := os.MkdirAll(scriptsRoot, 0o755); err != nil {
		return "", fmt.Errorf("create scripts dir: %w", err)
	}
	if err := writeModManifest(filepath.Join(destRoot, "mod.json"), modName); err != nil {
		return "", err
	}

	copied, err := copyScriptsRecursively(sourceRoot, destRoot, scriptsRoot)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Bundled %d script(s) into %s", copied, destRoot), nil
}

func validateBundleDestination(sourceRoot string, destRoot string) error {
	cleanSource := filepath.Clean(sourceRoot)
	cleanDest := filepath.Clean(destRoot)
	if cleanDest == cleanSource {
		return fmt.Errorf("bundle destination cannot equal source root")
	}
	if !strings.HasPrefix(cleanDest, cleanSource+string(filepath.Separator)) {
		return fmt.Errorf("bundle destination must be inside source root")
	}
	return nil
}

func writeModManifest(path string, modName string) error {
	manifest := modManifest{
		Name:        modName,
		Author:      "",
		Description: "",
		Version:     "1.0",
	}
	data, err := json.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("marshal mod manifest: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write mod.json: %w", err)
	}
	return nil
}

func copyScriptsRecursively(sourceRoot string, destRoot string, scriptsRoot string) (int, error) {
	copied := 0
	err := filepath.WalkDir(sourceRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == sourceRoot {
			return nil
		}
		if path == destRoot || strings.HasPrefix(path, destRoot+string(filepath.Separator)) {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") {
				return fs.SkipDir
			}
			return nil
		}
		if strings.ToLower(filepath.Ext(d.Name())) != ".gsc" {
			return nil
		}

		relPath, relErr := filepath.Rel(sourceRoot, path)
		if relErr != nil {
			return relErr
		}
		destPath := filepath.Join(scriptsRoot, relPath)
		if copyErr := copyFile(path, destPath); copyErr != nil {
			return copyErr
		}
		copied++
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("copy scripts: %w", err)
	}
	return copied, nil
}

func copyFile(src string, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}
