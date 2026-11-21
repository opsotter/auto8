// SPDX-License-Identifier: AGPL-3.0-or-later

package flow

import (
	"io/fs"
	"os"
	"strings"
	"testing"
	"testing/fstest"
)

// osDir returns the root of this repository as a file system.
func osDir(t *testing.T) fs.FS {
	t.Helper()
	return os.DirFS("../..")
}

const unifiedPlan = `---
title: Example - Plan
topic: example
artifact_contract: ce-unified-plan/v1
---

# Example - Plan

## Goal Capsule

- Objective: something.

## Product Contract

### Key Decisions

- A decision. Governs R1, R2.

### Requirements

Group one

- R1. The first requirement.
- R2. The second requirement
  continues on an indented line.

Group two

- R3. The third requirement.

### Acceptance Examples

- AE1. **Covers R1.** Given: a thing. Then: a result.

## Planning Contract

- R9. This line is outside the Requirements section.
`

func TestParsePlanReadsOnlyTheRequirementsSection(t *testing.T) {
	p, reports, ok := ParsePlan("docs/plans/example.md", []byte(unifiedPlan))
	if !ok {
		t.Fatalf("a unified plan must parse")
	}
	if len(reports) != 0 {
		t.Fatalf("unexpected reports: %v", reports)
	}
	if p.Topic != "example" || p.Path != "docs/plans/example.md" {
		t.Fatalf("wrong plan identity: %+v", p)
	}
	var ids []string
	for _, r := range p.Requirements {
		ids = append(ids, r.ID)
	}
	if got := strings.Join(ids, ","); got != "R1,R2,R3" {
		t.Fatalf("requirement IDs = %s, want R1,R2,R3", got)
	}
	if got := p.Requirements[1].Text; got != "The second requirement continues on an indented line." {
		t.Fatalf("continuation not joined: %q", got)
	}
}

func TestParsePlanReadsNames(t *testing.T) {
	plan := strings.Replace(unifiedPlan, "- R1. The first requirement.\n",
		"- R1. **Jump to a spot.** When chase-queue shows a spot, jump.\n"+
			"- R4. **Bold but no period** stays text.\n"+
			"- R5. **Two lines.** The text starts here\n  and goes on here.\n", 1)
	p, reports, ok := ParsePlan("docs/plans/example.md", []byte(plan))
	if !ok || len(reports) != 0 {
		t.Fatalf("ok = %v, reports = %v", ok, reports)
	}
	want := map[string][2]string{
		"R1": {"Jump to a spot", "When chase-queue shows a spot, jump."},
		"R2": {"", "The second requirement continues on an indented line."},
		"R4": {"", "**Bold but no period** stays text."},
		"R5": {"Two lines", "The text starts here and goes on here."},
	}
	for _, r := range p.Requirements {
		w, ok := want[r.ID]
		if !ok {
			continue
		}
		if r.Name != w[0] || r.Text != w[1] {
			t.Fatalf("%s: name %q, text %q, want %q, %q", r.ID, r.Name, r.Text, w[0], w[1])
		}
		if r.Hash != HashText(w[1]) {
			t.Fatalf("%s: the hash must cover only the text after the name", r.ID)
		}
	}
}

func TestHashIgnoresTheName(t *testing.T) {
	parse := func(line string) Requirement {
		p, _, _ := ParsePlan("docs/plans/example.md", []byte(strings.Replace(unifiedPlan, "- R1. The first requirement.", line, 1)))
		return p.Requirements[0]
	}
	plain := parse("- R1. The first requirement.")
	named := parse("- R1. **A name.** The first requirement.")
	renamed := parse("- R1. **Another name.** The first requirement.")
	if plain.Hash != named.Hash || named.Hash != renamed.Hash {
		t.Fatalf("hashes differ: %s %s %s", plain.Hash, named.Hash, renamed.Hash)
	}
	if named.Name != "A name" || renamed.Name != "Another name" {
		t.Fatalf("names = %q, %q", named.Name, renamed.Name)
	}
}

func TestParsePlanSkipsFilesWithoutTheUnifiedContract(t *testing.T) {
	draft := "---\ntitle: Draft\nstatus: draft\n---\n\n# Draft\n\n### Requirements\n\n- R1. Something.\n"
	if _, _, ok := ParsePlan("docs/plans/draft.md", []byte(draft)); ok {
		t.Fatalf("a plan without the unified contract must be skipped")
	}
}

func TestHashIgnoresWhiteSpace(t *testing.T) {
	a := HashText("The sync step   keeps\none issue.")
	b := HashText("The sync step keeps one issue.")
	if a != b {
		t.Fatalf("hashes differ for the same text: %s %s", a, b)
	}
	if len(a) != 12 {
		t.Fatalf("hash length = %d, want 12", len(a))
	}
	if a == HashText("The sync step keeps two issues.") {
		t.Fatalf("different text must give a different hash")
	}
}

func TestParsePlanReportsDuplicateIDs(t *testing.T) {
	dup := strings.Replace(unifiedPlan, "- R3. The third", "- R1. The third", 1)
	p, reports, ok := ParsePlan("docs/plans/example.md", []byte(dup))
	if !ok {
		t.Fatalf("the plan must still parse")
	}
	if len(reports) != 1 || !strings.Contains(reports[0], "R1") {
		t.Fatalf("want one report about R1, got %v", reports)
	}
	if len(p.Requirements) != 2 {
		t.Fatalf("the duplicate must not become a second requirement, got %d", len(p.Requirements))
	}
}

func TestLoadPlansRefusesDuplicateTopics(t *testing.T) {
	fsys := fstest.MapFS{
		"docs/plans/a.md": {Data: []byte(unifiedPlan)},
		"docs/plans/b.md": {Data: []byte(unifiedPlan)},
		"docs/plans/c.md": {Data: []byte(strings.Replace(unifiedPlan, "topic: example", "topic: other", 1))},
	}
	plans, reports := LoadPlans(fsys, "docs/plans")
	if len(plans) != 1 || plans[0].Topic != "other" {
		t.Fatalf("only the plan with a unique topic must load, got %d plans", len(plans))
	}
	if len(reports) != 1 || !strings.Contains(reports[0], "example") {
		t.Fatalf("want one report about the topic example, got %v", reports)
	}
}

func TestLoadPlansOfThisRepository(t *testing.T) {
	plans, reports := LoadPlans(osDir(t), "docs/plans")
	if len(reports) != 0 {
		t.Fatalf("the plans of this repository give reports: %v", reports)
	}
	if len(plans) == 0 {
		t.Fatalf("no unified plan found in docs/plans")
	}
	for _, p := range plans {
		if p.Topic == "requirements-workflow" {
			if n := len(p.Requirements); n != 35 {
				t.Fatalf("the workflow plan has %d requirements, want R1 to R35", n)
			}
		}
		if len(p.Requirements) == 0 {
			t.Fatalf("plan %s has no requirements", p.Path)
		}
		for _, r := range p.Requirements {
			if strings.Contains(r.Text, "Governs") || strings.Contains(r.Text, "Covers") {
				t.Fatalf("a cross-reference line became a requirement: %s %s", r.ID, r.Text)
			}
		}
	}
}
