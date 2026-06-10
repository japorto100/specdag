package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func RunDoctor(specsDir string) (int, int, error) {
	fmt.Printf("=== SpecDAG Doctor: Analyzing %s ===\n\n", specsDir)

	errorsCount := 0
	warningsCount := 0

	// Helper function to report issues
	reportIssue := func(isError bool, msg string) {
		if isError {
			fmt.Printf("[ERROR] %s\n", msg)
			errorsCount++
		} else {
			fmt.Printf("[WARN]  %s\n", msg)
			warningsCount++
		}
	}

	// 1. Prüfen, ob specsDir existiert
	fi, err := os.Stat(specsDir)
	if err != nil {
		reportIssue(true, fmt.Sprintf("Directory %s does not exist.", specsDir))
		fmt.Printf("\nFAIL: Doctor found %d errors and %d warnings.\n", errorsCount, warningsCount)
		return errorsCount, warningsCount, nil
	}
	if !fi.IsDir() {
		reportIssue(true, fmt.Sprintf("%s is not a directory.", specsDir))
		fmt.Printf("\nFAIL: Doctor found %d errors and %d warnings.\n", errorsCount, warningsCount)
		return errorsCount, warningsCount, nil
	}

	// 2. Suche nach generated files und deren Alter
	var latestLocalMod time.Time
	localMapsCount := 0

	var generatedFiles []string

	// Wir wandern durch das Verzeichnis
	err = filepath.Walk(specsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		name := info.Name()
		if info.IsDir() {
			// Prüfen, ob es ein Feature-Ordner ist und ob er eine dependency-map hat
			// Ein Feature-Ordner ist üblicherweise ein Unterordner von specsDir/features/
			rel, _ := filepath.Rel(specsDir, path)
			parts := strings.Split(rel, string(filepath.Separator))
			if len(parts) == 2 && parts[0] == "features" {
				// Dies ist ein konkreter Feature-Ordner, z.B. specs/features/012-agent-run
				mapYAML := filepath.Join(path, "dependency-map.yaml")
				mapYML := filepath.Join(path, "dependency-map.yml")
				mapJSON := filepath.Join(path, "dependency-map.json")

				_, errYAML := os.Stat(mapYAML)
				_, errYML := os.Stat(mapYML)
				_, errJSON := os.Stat(mapJSON)

				// Versuche severity aus spec.md Frontmatter zu lesen
				specMDPath := filepath.Join(path, "spec.md")
				severity := -1 // default to -1 (unknown)
				hasSpecMD := false

				if content, err := os.ReadFile(specMDPath); err == nil {
					hasSpecMD = true
					str := string(content)
					if strings.HasPrefix(str, "---") {
						parts := strings.SplitN(str, "---", 3)
						if len(parts) >= 3 {
							var fm map[string]interface{}
							if err := yaml.Unmarshal([]byte(parts[1]), &fm); err == nil {
								if sevVal, ok := fm["severity"]; ok {
									switch v := sevVal.(type) {
									case int:
										severity = v
									case float64:
										severity = int(v)
									}
								}
							}
						}
					}
				}

				hasMap := !os.IsNotExist(errYAML) || !os.IsNotExist(errYML) || !os.IsNotExist(errJSON)

				if !hasMap {
					if !hasSpecMD {
						reportIssue(false, fmt.Sprintf("Feature folder '%s' has no spec.md or dependency map. Severity is unknown.", rel))
					} else if severity == -1 {
						reportIssue(false, fmt.Sprintf("Feature folder '%s' has spec.md but no 'severity' defined in frontmatter. Severity is unknown.", rel))
					} else if severity >= 2 {
						reportIssue(false, fmt.Sprintf("Feature folder '%s' (severity: %d) has no dependency-map.yaml/json. Level 2/3 features require a dependency map.", rel, severity))
					}
				}
			}
		} else {
			// Es ist eine Datei
			if name == "dependency-map.yaml" || name == "dependency-map.yml" || name == "dependency-map.json" {
				// Lokale Map
				localMapsCount++
				if info.ModTime().After(latestLocalMod) {
					latestLocalMod = info.ModTime()
				}

				// Prüfen, ob es ein event-flow.md im selben Ordner gibt
				dir := filepath.Dir(path)
				flowMD := filepath.Join(dir, "event-flow.md")
				if _, err := os.Stat(flowMD); os.IsNotExist(err) {
					reportIssue(false, fmt.Sprintf("Dependency map at %s has no corresponding event-flow.md in the same directory.", path))
				}

				// Kanten/Knoten refs prüfen
				depMap, err := LoadDependencyMap(path)
				if err != nil {
					reportIssue(true, fmt.Sprintf("Failed to load map %s: %v", path, err))
				} else {
					for _, node := range depMap.Nodes {
						if node.Ref != "" {
							// ref auflösen relativ zum root (oder als relativer pfad)
							refPath := node.Ref
							// Wenn der ref-Pfad relativ ist, prüfen wir ihn relativ zum aktuellen Arbeitsverzeichnis oder relativ zur Map
							_, errRef := os.Stat(refPath)
							if os.IsNotExist(errRef) {
								// Versuchen relativ zum Map-Ordner
								mapDir := filepath.Dir(path)
								_, errRefRel := os.Stat(filepath.Join(mapDir, refPath))
								if os.IsNotExist(errRefRel) {
									// Versuchen relativ zum specsDir
									_, errRefSpecs := os.Stat(filepath.Join(specsDir, "..", refPath))
									if os.IsNotExist(errRefSpecs) {
										reportIssue(false, fmt.Sprintf("Node '%s' in %s has broken ref path: '%s'", node.ID, name, node.Ref))
									}
								}
							}
						}
					}
				}
			} else if strings.Contains(path, "_generated") {
				generatedFiles = append(generatedFiles, path)
			}
		}
		return nil
	})

	if err != nil {
		reportIssue(true, fmt.Sprintf("Error walking specs directory: %v", err))
	}

	// 3. Generated Folder Check
	genFolder := filepath.Join(specsDir, "_generated")
	if fiGen, err := os.Stat(genFolder); os.IsNotExist(err) || !fiGen.IsDir() {
		reportIssue(false, fmt.Sprintf("Generated directory %s does not exist. Run 'specdag assemble' and 'specdag report' to generate artifacts.", genFolder))
	} else {
		// Prüfen, ob die generierten Dateien älter sind als die neuesten lokalen Änderungen
		for _, genPath := range generatedFiles {
			info, err := os.Stat(genPath)
			if err == nil {
				if info.ModTime().Before(latestLocalMod) {
					reportIssue(false, fmt.Sprintf("Generated file %s is older than the latest local map change (Stale output). Latest map changed: %s, file changed: %s.", filepath.Base(genPath), latestLocalMod.Format(time.RFC3339), info.ModTime().Format(time.RFC3339)))
				}
			}
		}
	}

	// Summary
	fmt.Printf("\n=== Doctor Summary ===\n")
	if errorsCount == 0 && warningsCount == 0 {
		fmt.Println("PASS: Everything looks great! No issues found.")
	} else {
		fmt.Printf("FAIL: Found %d error(s) and %d warning(s).\n", errorsCount, warningsCount)
	}

	return errorsCount, warningsCount, nil
}

var doctorCmd = &cobra.Command{
	Use:   "doctor [specs_dir]",
	Short: "Checks the specifications directory structure for consistency and best practices",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		specsDir := args[0]
		errorsCount, _, err := RunDoctor(specsDir)
		if err != nil {
			fmt.Printf("FAIL: %v\n", err)
			os.Exit(1)
		}
		if errorsCount > 0 {
			os.Exit(1)
		}
	},
}
