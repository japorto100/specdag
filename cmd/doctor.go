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

	issues := &issueCounter{}
	if !checkDoctorRoot(specsDir, issues) {
		printDoctorSummary(issues)
		return issues.Errors, issues.Warnings, nil
	}

	state := &doctorState{}
	if walkFailure := walkDoctorSpecs(specsDir, state, issues); walkFailure != "" {
		issues.Report(true, fmt.Sprintf("Error walking specs directory: %s", walkFailure))
	}
	checkGeneratedArtifacts(specsDir, state, issues)
	printDoctorSummary(issues)

	return issues.Errors, issues.Warnings, nil
}

type doctorState struct {
	latestLocalMod time.Time
	generatedFiles []string
}

func checkDoctorRoot(specsDir string, issues *issueCounter) bool {
	fileInfo, err := os.Stat(specsDir)
	if err != nil {
		issues.Report(true, fmt.Sprintf("Directory %s does not exist.", specsDir))
		return false
	}
	if !fileInfo.IsDir() {
		issues.Report(true, fmt.Sprintf("%s is not a directory.", specsDir))
		return false
	}
	return true
}

func walkDoctorSpecs(specsDir string, state *doctorState, issues *issueCounter) string {
	err := filepath.Walk(specsDir, func(path string, info os.FileInfo, walkErr error) error {
		return checkDoctorWalkPath(specsDir, path, info, walkErr, state, issues)
	})
	if err != nil {
		return err.Error()
	}
	return ""
}

func checkDoctorWalkPath(specsDir string, path string, info os.FileInfo, walkErr error, state *doctorState, issues *issueCounter) error {
	if walkErr != nil {
		return fmt.Errorf("walk %s: %w", path, walkErr)
	}
	if info.IsDir() {
		checkFeatureDirectory(specsDir, path, issues)
		return nil
	}
	checkDoctorFile(specsDir, path, info, state, issues)
	return nil
}

func checkFeatureDirectory(specsDir string, path string, issues *issueCounter) {
	relPath, err := filepath.Rel(specsDir, path)
	if err != nil {
		return
	}
	parts := strings.Split(relPath, string(filepath.Separator))
	if len(parts) != 2 || parts[0] != "features" {
		return
	}
	if featureHasDependencyMap(path) {
		return
	}

	hasSpecMD, severity := readFeatureSeverity(path)
	switch {
	case !hasSpecMD:
		issues.Report(false, fmt.Sprintf("Feature folder '%s' has no spec.md or dependency map. severity missing; cannot determine whether dependency-map is required.", relPath))
	case severity == -1:
		issues.Report(false, fmt.Sprintf("Feature folder '%s' has spec.md but no 'severity' or 'level' defined in frontmatter. severity missing; cannot determine whether dependency-map is required.", relPath))
	case severity >= 2:
		issues.Report(false, fmt.Sprintf("Feature folder '%s' (severity: %d) has no dependency-map.yaml/json. Level 2/3 features require a dependency map.", relPath, severity))
	}
}

func featureHasDependencyMap(featurePath string) bool {
	for _, name := range []string{"dependency-map.yaml", "dependency-map.yml", "dependency-map.json"} {
		if _, err := os.Stat(filepath.Join(featurePath, name)); err == nil {
			return true
		}
	}
	return false
}

func readFeatureSeverity(featurePath string) (bool, int) {
	content, err := os.ReadFile(filepath.Join(featurePath, "spec.md"))
	if err != nil {
		return false, -1
	}

	str := string(content)
	if !strings.HasPrefix(str, "---") {
		return true, -1
	}

	parts := strings.SplitN(str, "---", 3)
	if len(parts) < 3 {
		return true, -1
	}

	var frontmatter map[string]any
	if err := yaml.Unmarshal([]byte(parts[1]), &frontmatter); err != nil {
		return true, -1
	}
	return true, frontmatterSeverity(frontmatter)
}

func frontmatterSeverity(frontmatter map[string]any) int {
	if severity, ok := severityValue(frontmatter["severity"]); ok {
		return severity
	}
	if level, ok := severityValue(frontmatter["level"]); ok {
		return level
	}
	return -1
}

func severityValue(value any) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case float64:
		return int(v), true
	default:
		return -1, false
	}
}

func checkDoctorFile(specsDir string, path string, info os.FileInfo, state *doctorState, issues *issueCounter) {
	name := info.Name()
	if isDependencyMapFilename(name) {
		checkDoctorDependencyMap(specsDir, path, info, state, issues)
		return
	}
	if strings.Contains(path, "_generated") {
		state.generatedFiles = append(state.generatedFiles, path)
	}
}

func checkDoctorDependencyMap(specsDir string, path string, info os.FileInfo, state *doctorState, issues *issueCounter) {
	if info.ModTime().After(state.latestLocalMod) {
		state.latestLocalMod = info.ModTime()
	}

	flowMD := filepath.Join(filepath.Dir(path), "event-flow.md")
	if _, err := os.Stat(flowMD); os.IsNotExist(err) {
		issues.Report(false, fmt.Sprintf("Dependency map at %s has no corresponding event-flow.md in the same directory.", path))
	}

	depMap, err := LoadDependencyMap(path)
	if err != nil {
		issues.Report(true, fmt.Sprintf("Failed to load map %s: %v", path, err))
		return
	}
	for _, node := range depMap.Nodes {
		if node.Ref != "" && resolveSpecRef(specsDir, path, node.Ref) == "" {
			issues.Report(false, fmt.Sprintf("Node '%s' in %s has broken ref path: '%s'", node.ID, info.Name(), node.Ref))
		}
	}
}

func checkGeneratedArtifacts(specsDir string, state *doctorState, issues *issueCounter) {
	genFolder := filepath.Join(specsDir, "_generated")
	fileInfo, err := os.Stat(genFolder)
	if os.IsNotExist(err) || (err == nil && !fileInfo.IsDir()) {
		issues.Report(false, fmt.Sprintf("Generated directory %s does not exist. Run 'specdag assemble' and 'specdag report' to generate artifacts.", genFolder))
		return
	}

	for _, genPath := range state.generatedFiles {
		info, err := os.Stat(genPath)
		if err == nil && info.ModTime().Before(state.latestLocalMod) {
			issues.Report(false, fmt.Sprintf("Generated file %s is older than the latest local map change (Stale output). Latest map changed: %s, file changed: %s.", filepath.Base(genPath), state.latestLocalMod.Format(time.RFC3339), info.ModTime().Format(time.RFC3339)))
		}
	}
}

func printDoctorSummary(issues *issueCounter) {
	fmt.Printf("\n=== Doctor Summary ===\n")
	if issues.Errors == 0 && issues.Warnings == 0 {
		fmt.Println("PASS: Everything looks great! No issues found.")
		return
	}
	fmt.Printf("FAIL: Found %d error(s) and %d warning(s).\n", issues.Errors, issues.Warnings)
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
