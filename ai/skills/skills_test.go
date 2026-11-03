package skills

import (
	"strings"
	"testing"
)

func TestParseSkillInvocationControlsAndStandardName(t *testing.T) {
	skill, err := parseSkill(strings.NewReader(`---
name: standard-skill
description: A test skill
user-invocable: false
disable-model-invocation: true
---
Instructions.
`))
	if err != nil {
		t.Fatal(err)
	}
	if skill.Name != "standard-skill" {
		t.Fatalf("skill command ID = %q, want standard name", skill.Name)
	}
	if skill.IsUserInvocable() || skill.IsModelInvocable() {
		t.Fatalf("invocation controls = user:%t model:%t, want both disabled", skill.IsUserInvocable(), skill.IsModelInvocable())
	}
}

func TestParseSkillIgnoresLegacyFunctionNameFields(t *testing.T) {
	skill, err := parseSkill(strings.NewReader(`---
name: standard-name
description: A test skill
function-name: old-command
metadata:
  function-name: old-metadata-command
---
Instructions.
`))
	if err != nil {
		t.Fatal(err)
	}
	if skill.Name != "standard-name" {
		t.Fatalf("skill name = %q, want standard-name", skill.Name)
	}
}

func TestParseSkillInvocationControlsDefaultToBothEnabled(t *testing.T) {
	skill, err := parseSkill(strings.NewReader(`---
name: default-skill
description: A test skill
---
Instructions.
`))
	if err != nil {
		t.Fatal(err)
	}
	if !skill.IsUserInvocable() || !skill.IsModelInvocable() {
		t.Fatalf("invocation defaults = user:%t model:%t, want both enabled", skill.IsUserInvocable(), skill.IsModelInvocable())
	}
}

func TestInvocationViewsFilterIndependently(t *testing.T) {
	manualOnly := true
	backgroundOnly := false
	skills := Skills{
		{Name: "both", UserInvocable: nil},
		{Name: "manual", DisableModelInvocation: true},
		{Name: "automatic", UserInvocable: &backgroundOnly},
		{Name: "neither", UserInvocable: &backgroundOnly, DisableModelInvocation: true},
		{Name: "explicit-true", UserInvocable: &manualOnly},
	}
	user := skills.UserInvocable()
	model := skills.ModelInvocable()
	if len(user) != 3 || user[0].Name != "both" || user[1].Name != "manual" || user[2].Name != "explicit-true" {
		t.Fatalf("user-invocable skills = %#v", user)
	}
	if len(model) != 3 || model[0].Name != "both" || model[1].Name != "automatic" || model[2].Name != "explicit-true" {
		t.Fatalf("model-invocable skills = %#v", model)
	}
}

func TestParseSelectionKeepsKnownUniqueSkills(t *testing.T) {
	first := &SkillT{Name: "first"}
	second := &SkillT{Name: "second"}
	selected, err := ParseSelection(`["second","unknown","first","second"]`, Skills{first, second})
	if err != nil {
		t.Fatal(err)
	}
	if len(selected) != 2 || selected[0] != second || selected[1] != first {
		t.Fatalf("selection = %#v, want the known skills once in model order", selected)
	}
}

func TestParseSelectionAcceptsEmptyListAndFencedJSON(t *testing.T) {
	if selected, err := ParseSelection("[]", Skills{{Name: "only"}}); err != nil || len(selected) != 0 {
		t.Fatalf("empty selection = %#v, %v", selected, err)
	}
	selected, err := ParseSelection("```json\n[\"only\"]\n```", Skills{{Name: "only"}})
	if err != nil || len(selected) != 1 || selected[0].Name != "only" {
		t.Fatalf("fenced selection = %#v, %v", selected, err)
	}
}

func TestParseSelectionRejectsNonJSONOutput(t *testing.T) {
	if _, err := ParseSelection("use the first skill", Skills{{Name: "first"}}); err == nil {
		t.Fatal("non-JSON skill selection unexpectedly succeeded")
	}
}
