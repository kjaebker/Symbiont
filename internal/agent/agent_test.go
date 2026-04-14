package agent

import (
	"strings"
	"testing"

	"github.com/kjaebker/symbiont/internal/db"
)

// --- parseSkill ---

func TestParseSkillValid(t *testing.T) {
	input := "---\nname: test-skill\ndescription: Does something useful.\n---\n\nHere is the body.\n"
	s, err := parseSkill([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Name != "test-skill" {
		t.Errorf("name = %q, want %q", s.Name, "test-skill")
	}
	if s.Description != "Does something useful." {
		t.Errorf("description = %q", s.Description)
	}
	if s.Body != "Here is the body." {
		t.Errorf("body = %q", s.Body)
	}
}

func TestParseSkillMissingFence(t *testing.T) {
	_, err := parseSkill([]byte("name: x\ndescription: y\n"))
	if err == nil {
		t.Fatal("expected error for missing frontmatter fence")
	}
}

func TestParseSkillUnclosedFence(t *testing.T) {
	_, err := parseSkill([]byte("---\nname: x\ndescription: y\n"))
	if err == nil {
		t.Fatal("expected error for unclosed frontmatter fence")
	}
}

func TestParseSkillMissingName(t *testing.T) {
	input := "---\ndescription: No name here.\n---\n\nBody.\n"
	_, err := parseSkill([]byte(input))
	if err == nil {
		t.Fatal("expected error for missing name field")
	}
}

// --- LoadSkills ---

func TestLoadSkillsReturnsAll(t *testing.T) {
	skills, err := LoadSkills()
	if err != nil {
		t.Fatalf("LoadSkills: %v", err)
	}
	wantAtLeast := 6
	if len(skills) < wantAtLeast {
		t.Errorf("got %d skills, want at least %d", len(skills), wantAtLeast)
	}
	for _, s := range skills {
		if s.Name == "" {
			t.Errorf("skill has empty name")
		}
		if s.Description == "" {
			t.Errorf("skill %q has empty description", s.Name)
		}
		if s.Body == "" {
			t.Errorf("skill %q has empty body", s.Name)
		}
	}
}

func TestGetSkillFound(t *testing.T) {
	s, err := GetSkill("water-test-analysis")
	if err != nil {
		t.Fatalf("GetSkill: %v", err)
	}
	if s.Name != "water-test-analysis" {
		t.Errorf("name = %q", s.Name)
	}
}

func TestGetSkillNotFound(t *testing.T) {
	_, err := GetSkill("nonexistent-skill")
	if err == nil {
		t.Fatal("expected error for missing skill")
	}
}

func TestSkillFile(t *testing.T) {
	s := Skill{
		SkillMeta: SkillMeta{Name: "foo", Description: "Does foo."},
		Body:      "The body text.",
	}
	got := s.SkillFile()
	if !strings.HasPrefix(got, "---\n") {
		t.Errorf("SkillFile() should start with ---\\n, got: %q", got[:10])
	}
	if !strings.Contains(got, "name: foo") {
		t.Errorf("SkillFile() missing name field")
	}
	if !strings.Contains(got, "The body text.") {
		t.Errorf("SkillFile() missing body")
	}
}

// --- BuildContext ---

func ptrStr(s string) *string  { return &s }
func ptrF64(f float64) *float64 { return &f }

func TestBuildContextMinimal(t *testing.T) {
	in := ContextInput{
		Settings: &db.AgentSettings{
			Tone:              "analytical",
			DosingProductLine: "generic",
			EnabledSkills:     []string{},
		},
	}
	got := BuildContext(in)
	if !strings.Contains(got, "Symbiont") {
		t.Errorf("context missing Symbiont header")
	}
	if !strings.Contains(got, "analytical") {
		t.Errorf("context missing tone")
	}
	if !strings.Contains(got, "No livestock recorded") {
		t.Errorf("context missing livestock empty state")
	}
}

func TestBuildContextWithTankAndLivestock(t *testing.T) {
	display := &db.TankProfile{
		Section:       "display",
		VolumeGallons: ptrF64(90),
		LengthIn:      ptrF64(48),
		WidthIn:       ptrF64(24),
		HeightIn:      ptrF64(24),
		TankType:      ptrStr("reef"),
	}
	sump := &db.TankProfile{
		Section:       "sump",
		VolumeGallons: ptrF64(30),
	}
	livestock := []db.LivestockItem{
		{Name: "Clownfish", Type: "fish", Quantity: 2, Status: "healthy", Species: ptrStr("Amphiprioninae")},
	}
	in := ContextInput{
		Settings:  &db.AgentSettings{Tone: "casual", DosingProductLine: "brs_pharma", EnabledSkills: []string{}},
		Display:   display,
		Sump:      sump,
		Livestock: livestock,
	}
	got := BuildContext(in)

	if !strings.Contains(got, "120.0 gallons") {
		t.Errorf("expected summed volume 120.0, got:\n%s", got)
	}
	if !strings.Contains(got, "Clownfish") {
		t.Errorf("expected livestock name in context")
	}
	if !strings.Contains(got, "brs_pharma") {
		t.Errorf("expected product line in context")
	}
}

func TestBuildContextVolumeOverride(t *testing.T) {
	override := 75.0
	in := ContextInput{
		Settings: &db.AgentSettings{
			NetVolumeGallons: &override,
			EnabledSkills:    []string{},
		},
		Display: &db.TankProfile{VolumeGallons: ptrF64(90)},
	}
	got := BuildContext(in)
	if !strings.Contains(got, "75.0 gallons") {
		t.Errorf("expected override volume 75.0 in context:\n%s", got)
	}
}

func TestBuildContextAlertRuleTargets(t *testing.T) {
	lo, hi := 8.0, 9.5
	in := ContextInput{
		Settings: &db.AgentSettings{EnabledSkills: []string{}},
		AlertRules: []db.AlertRule{
			{ProbeName: "Alk", Condition: "outside_range", Enabled: true, ThresholdLow: &lo, ThresholdHigh: &hi},
		},
	}
	got := BuildContext(in)
	if !strings.Contains(got, "Alk") {
		t.Errorf("expected Alk probe in context:\n%s", got)
	}
	if !strings.Contains(got, "8") {
		t.Errorf("expected threshold value in context")
	}
}

func TestBuildContextCustomGuardrails(t *testing.T) {
	in := ContextInput{
		Settings: &db.AgentSettings{
			EnabledSkills:    []string{},
			CustomGuardrails: ptrStr("Never recommend Alk above 10 dKH for this tank."),
		},
	}
	got := BuildContext(in)
	if !strings.Contains(got, "Never recommend Alk above 10 dKH") {
		t.Errorf("expected custom guardrails in context:\n%s", got)
	}
}

func TestBuildContextEnabledSkills(t *testing.T) {
	in := ContextInput{
		Settings: &db.AgentSettings{EnabledSkills: []string{}},
		Skills: []Skill{
			{SkillMeta: SkillMeta{Name: "water-test-analysis", Description: "Analyze water tests."}, Body: "..."},
		},
	}
	got := BuildContext(in)
	if !strings.Contains(got, "water-test-analysis") {
		t.Errorf("expected skill name in context:\n%s", got)
	}
}
