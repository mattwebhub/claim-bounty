package main

import (
	"path/filepath"
	"testing"
)

func TestArchitectureFixtures(t *testing.T) {
	t.Parallel()
	valid, err := check(filepath.Join("testdata", "valid"))
	if err != nil {
		t.Fatal(err)
	}
	if len(valid) != 0 {
		t.Fatalf("valid fixture violations = %#v", valid)
	}
	invalid, err := check(filepath.Join("testdata", "invalid"))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"GO-ARCH-001": false, "GO-ARCH-002": false, "GO-CTX-001": false, "GO-HTTP-001": false}
	for _, item := range invalid {
		want[item.ruleID] = true
	}
	for id, found := range want {
		if !found {
			t.Errorf("negative fixtures did not trigger %s: %#v", id, invalid)
		}
	}
}

func TestRepositoryPassesArchitectureRules(t *testing.T) {
	violations, err := check(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 0 {
		t.Fatalf("repository violations = %#v", violations)
	}
}

func TestUnknownRuleCannotBeExplained(t *testing.T) {
	t.Parallel()
	if err := explainRule("UNKNOWN"); err == nil {
		t.Fatal("explainRule accepted an unknown rule")
	}
}
