package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/japorto100/specdag/dag"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func RunCheckCatalogs(specsDir string) (int, int, error) {
	fmt.Printf("=== SpecDAG Catalog Check: Analyzing %s ===\n\n", specsDir)

	issues := &issueCounter{}
	err := filepath.Walk(specsDir, func(path string, info os.FileInfo, walkErr error) error {
		return checkCatalogWalkPath(specsDir, path, info, walkErr, issues)
	})

	if err != nil {
		issues.Report(true, fmt.Sprintf("Error walking directory: %v", err))
	}

	fmt.Printf("\n=== Catalog Check Summary ===\n")
	if issues.Errors == 0 && issues.Warnings == 0 {
		fmt.Println("PASS: All referenced events and contracts match their catalog definitions perfectly!")
	} else {
		fmt.Printf("FAIL: Found %d error(s) and %d warning(s).\n", issues.Errors, issues.Warnings)
	}

	return issues.Errors, issues.Warnings, nil
}

type issueCounter struct {
	Errors   int
	Warnings int
}

func (i *issueCounter) Report(isError bool, msg string) {
	if isError {
		fmt.Printf("[ERROR] %s\n", msg)
		i.Errors++
		return
	}
	fmt.Printf("[WARN]  %s\n", msg)
	i.Warnings++
}

func checkCatalogWalkPath(specsDir string, path string, info os.FileInfo, walkErr error, issues *issueCounter) error {
	if walkErr != nil {
		return fmt.Errorf("walk %s: %w", path, walkErr)
	}
	if info.IsDir() || !isDependencyMapFilename(info.Name()) {
		return nil
	}

	depMap, err := LoadDependencyMap(path)
	if err != nil {
		return fmt.Errorf("load dependency map %s: %w", path, err)
	}

	for _, node := range depMap.Nodes {
		if node.Type == "event" || node.Type == "contract" {
			checkCatalogNode(specsDir, path, info.Name(), depMap, node, issues)
		}
	}
	return nil
}

func checkCatalogNode(specsDir string, mapPath string, mapName string, depMap *dag.DependencyMap, node dag.Node, issues *issueCounter) {
	if node.Ref == "" {
		issues.Report(false, fmt.Sprintf("Node '%s' (%s) in %s has no 'ref' pointing to its catalog file.", node.ID, node.Type, mapName))
		return
	}

	finalPath := resolveSpecRef(specsDir, mapPath, node.Ref)
	if finalPath == "" {
		issues.Report(true, fmt.Sprintf("Broken reference: Node '%s' (%s) in %s points to '%s', but file does not exist.", node.ID, node.Type, mapName, node.Ref))
		return
	}
	if strings.HasSuffix(strings.ToLower(finalPath), ".md") {
		checkCatalogFrontmatter(finalPath, depMap, node, issues)
	}
}

func resolveSpecRef(specsDir string, mapPath string, refPath string) string {
	candidates := []string{
		refPath,
		filepath.Join(filepath.Dir(mapPath), refPath),
		filepath.Join(specsDir, "..", refPath),
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}

func parseFrontmatter(filePath string) (map[string]any, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read frontmatter file %s: %w", filePath, err)
	}

	str := string(content)
	if !strings.HasPrefix(str, "---") {
		return nil, fmt.Errorf("no frontmatter found (missing starting ---)")
	}

	parts := strings.SplitN(str, "---", 3)
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid frontmatter syntax")
	}

	var data map[string]any
	if err := yaml.Unmarshal([]byte(parts[1]), &data); err != nil {
		return nil, fmt.Errorf("parse frontmatter YAML in %s: %w", filePath, err)
	}
	return data, nil
}

func checkCatalogFrontmatter(finalPath string, depMap *dag.DependencyMap, node dag.Node, issues *issueCounter) {
	frontmatter, err := parseFrontmatter(finalPath)
	if err != nil {
		issues.Report(false, fmt.Sprintf("Catalog file '%s' (referenced by '%s') has invalid or missing frontmatter: %v", finalPath, node.ID, err))
		return
	}

	checkCatalogName(finalPath, frontmatter, node, issues)
	checkCatalogStatus(finalPath, frontmatter, depMap, node, issues)
}

func checkCatalogName(finalPath string, frontmatter map[string]any, node dag.Node, issues *issueCounter) {
	frontmatterName, _ := frontmatter["name"].(string)
	if frontmatterName == "" {
		issues.Report(false, fmt.Sprintf("Catalog file '%s' has no 'name' property in frontmatter.", finalPath))
		return
	}

	cleanID := strings.TrimPrefix(node.ID, "event.")
	cleanID = strings.TrimPrefix(cleanID, "contract.")
	if frontmatterName != cleanID && frontmatterName != node.ID && frontmatterName != node.Title {
		issues.Report(false, fmt.Sprintf("Name mismatch: Catalog file '%s' defines name '%s', but map node ID is '%s' and Title is '%s'.", finalPath, frontmatterName, node.ID, node.Title))
	}
}

func checkCatalogStatus(finalPath string, frontmatter map[string]any, depMap *dag.DependencyMap, node dag.Node, issues *issueCounter) {
	frontmatterStatus, _ := frontmatter["status"].(string)
	if frontmatterStatus == "" {
		return
	}

	mapNodeStatus := node.Status
	if mapNodeStatus == "" {
		mapNodeStatus = depMap.Graph.Status
	}
	if mapNodeStatus == "accepted" && (frontmatterStatus == "draft" || frontmatterStatus == "deprecated") {
		issues.Report(false, fmt.Sprintf("Status conflict: Map node '%s' is '%s', but catalog file '%s' status is '%s'.", node.ID, mapNodeStatus, finalPath, frontmatterStatus))
	}
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
