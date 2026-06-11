package cmd

import (
	"strings"
	"testing"
)

func TestGetSkillRulesIncludesEmbeddedCompletionRules(t *testing.T) {
	rules, err := GetSkillRules()
	if err != nil {
		t.Fatalf("GetSkillRules failed: %v", err)
	}

	for _, expected := range []string{
		"SKILL.md",
		"Completion Attestation Rules",
		"specdag hash",
		"specdag verify",
	} {
		if !strings.Contains(rules, expected) {
			t.Fatalf("expected embedded rules to include %q", expected)
		}
	}
}
