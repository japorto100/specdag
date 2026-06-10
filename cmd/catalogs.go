package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func RunCheckCatalogs(specsDir string) (int, int, error) {
	fmt.Printf("=== SpecDAG Catalog Check: Analyzing %s ===\n\n", specsDir)

	errorsCount := 0
	warningsCount := 0

	reportIssue := func(isError bool, msg string) {
		if isError {
			fmt.Printf("[ERROR] %s\n", msg)
			errorsCount++
		} else {
			fmt.Printf("[WARN]  %s\n", msg)
			warningsCount++
		}
	}

	// Hilfsfunktion: Frontmatter aus einer MD-Datei extrahieren
	parseFrontmatter := func(filePath string) (map[string]interface{}, error) {
		content, err := os.ReadFile(filePath)
		if err != nil {
			return nil, err
		}

		str := string(content)
		if !strings.HasPrefix(str, "---") {
			return nil, fmt.Errorf("no frontmatter found (missing starting ---)")
		}

		parts := strings.SplitN(str, "---", 3)
		if len(parts) < 3 {
			return nil, fmt.Errorf("invalid frontmatter syntax")
		}

		var data map[string]interface{}
		if err := yaml.Unmarshal([]byte(parts[1]), &data); err != nil {
			return nil, err
		}
		return data, nil
	}

	err := filepath.Walk(specsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		name := info.Name()
		if !info.IsDir() && (name == "dependency-map.yaml" || name == "dependency-map.yml" || name == "dependency-map.json") {
			depMap, err := LoadDependencyMap(path)
			if err != nil {
				// validate hat das schon gefangen, aber zur sicherheit
				return nil
			}

			for _, node := range depMap.Nodes {
				if node.Type == "event" || node.Type == "contract" {
					if node.Ref == "" {
						reportIssue(false, fmt.Sprintf("Node '%s' (%s) in %s has no 'ref' pointing to its catalog file.", node.ID, node.Type, name))
						continue
					}

					// Auflösen des Pfades
					refPath := node.Ref
					var finalPath string
					if _, err := os.Stat(refPath); err == nil {
						finalPath = refPath
					} else {
						// Map Ordner rel
						mapDir := filepath.Dir(path)
						relPath := filepath.Join(mapDir, refPath)
						if _, err := os.Stat(relPath); err == nil {
							finalPath = relPath
						} else {
							// Specs Ordner rel
							specsRel := filepath.Join(specsDir, "..", refPath)
							if _, err := os.Stat(specsRel); err == nil {
								finalPath = specsRel
							}
						}
					}

					if finalPath == "" {
						reportIssue(true, fmt.Sprintf("Broken reference: Node '%s' (%s) in %s points to '%s', but file does not exist.", node.ID, node.Type, name, node.Ref))
						continue
					}

					// MD-Frontmatter check
					if strings.HasSuffix(strings.ToLower(finalPath), ".md") {
						fm, err := parseFrontmatter(finalPath)
						if err != nil {
							reportIssue(false, fmt.Sprintf("Catalog file '%s' (referenced by '%s') has invalid or missing frontmatter: %v", finalPath, node.ID, err))
							continue
						}

						// Name vergleichen
						fmName, _ := fm["name"].(string)
						if fmName == "" {
							reportIssue(false, fmt.Sprintf("Catalog file '%s' has no 'name' property in frontmatter.", finalPath))
						} else {
							// Event ID ist z.B. event.document.uploaded, oder contract.bot.config.v1
							// Frontmatter Name ist z.B. document.uploaded
							cleanID := node.ID
							cleanID = strings.TrimPrefix(cleanID, "event.")
							cleanID = strings.TrimPrefix(cleanID, "contract.")

							if fmName != cleanID && fmName != node.ID && fmName != node.Title {
								reportIssue(false, fmt.Sprintf("Name mismatch: Catalog file '%s' defines name '%s', but map node ID is '%s' and Title is '%s'.", finalPath, fmName, node.ID, node.Title))
							}
						}

						// Status vergleichen
						fmStatus, _ := fm["status"].(string)
						if fmStatus != "" {
							mapNodeStatus := node.Status
							if mapNodeStatus == "" {
								mapNodeStatus = depMap.Graph.Status
							}

							if mapNodeStatus == "accepted" && (fmStatus == "draft" || fmStatus == "deprecated") {
								reportIssue(false, fmt.Sprintf("Status conflict: Map node '%s' is '%s', but catalog file '%s' status is '%s'.", node.ID, mapNodeStatus, finalPath, fmStatus))
							}
						}
					}
				}
			}
		}
		return nil
	})

	if err != nil {
		reportIssue(true, fmt.Sprintf("Error walking directory: %v", err))
	}

	fmt.Printf("\n=== Catalog Check Summary ===\n")
	if errorsCount == 0 && warningsCount == 0 {
		fmt.Println("PASS: All referenced events and contracts match their catalog definitions perfectly!")
	} else {
		fmt.Printf("FAIL: Found %d error(s) and %d warning(s).\n", errorsCount, warningsCount)
	}

	return errorsCount, warningsCount, nil
}

var catalogsStrictFlag bool

var checkCatalogsCmd = &cobra.Command{
	Use:   "check-catalogs [specs_dir]",
	Short: "Cross-checks referenced events and contracts against catalog files",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		specsDir := args[0]

		if catalogsStrictFlag {
			// TODO (v0.4.0): Implement strict schema properties validation.
			// This will validate that the catalog frontmatter contains and validates:
			// - domain
			// - producer
			// - consumers
			// - schema_ref
			// - privacy_class
			// - idempotent
			// - ordering
			// - replay_safe
			// - tenant_scoped
			fmt.Println("INFO: Strict catalog validation enabled. (TODO: Schema properties validation will be enforced in v0.4.0)")
		}

		errorsCount, _, err := RunCheckCatalogs(specsDir)
		if err != nil {
			fmt.Printf("FAIL: %v\n", err)
			os.Exit(1)
		}
		if errorsCount > 0 {
			os.Exit(1)
		}
	},
}

func init() {
	checkCatalogsCmd.Flags().BoolVar(&catalogsStrictFlag, "strict", false, "Enforce strict catalog frontmatter fields (TODO: v0.4.0)")
}
