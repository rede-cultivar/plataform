package practices

import (
	"errors"
	"testing"
)

func validInput() Input {
	return Input{
		Title:      "Adubo caseiro de casca de banana",
		Summary:    "Adubo simples para hortas domésticas.",
		Category:   "Adubação",
		Difficulty: DifficultyBeginner,
		Materials:  []string{"Cascas de banana", "Água"},
		Steps:      []string{"Ferva as cascas", "Deixe descansar", "Dilua e aplique"},
		License:    "CC BY-SA 4.0",
	}
}

func TestValidateInputDraftAllowsIncomplete(t *testing.T) {
	input := Input{Title: "Rascunho"}
	input = NormalizeInput(input)

	if err := ValidateInput(input, false); err != nil {
		t.Fatalf("draft rejected: %v", err)
	}
}

func TestValidateInputPublishedRequiresSteps(t *testing.T) {
	input := NormalizeInput(Input{Title: "Prática sem etapas", License: "CC BY-SA 4.0"})
	err := ValidateInput(input, true)

	var validation *ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("error = %v, want ValidationError", err)
	}
	if validation.Field != "steps" {
		t.Errorf("field = %q, want steps", validation.Field)
	}
}

func TestValidateInputPublishedRequiresLicense(t *testing.T) {
	input := validInput()
	input.License = ""
	input = NormalizeInput(input)

	err := ValidateInput(input, true)
	var validation *ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("error = %v, want ValidationError", err)
	}
	if validation.Field != "license" {
		t.Errorf("field = %q, want license", validation.Field)
	}
}

func TestValidateInputValidPublished(t *testing.T) {
	if err := ValidateInput(NormalizeInput(validInput()), true); err != nil {
		t.Fatalf("valid input rejected: %v", err)
	}
}

func TestPracticeCanView(t *testing.T) {
	draft := &Practice{AuthorID: "person-1", Status: StatusDraft, Visibility: VisibilityPrivate}
	if draft.CanView("") {
		t.Error("anonymous viewer can see draft")
	}
	if draft.CanView("person-2") {
		t.Error("other person can see draft")
	}
	if !draft.CanView("person-1") {
		t.Error("author cannot see draft")
	}

	published := &Practice{AuthorID: "person-1", Status: StatusPublished, Visibility: VisibilityPublic}
	if !published.CanView("") {
		t.Error("anonymous viewer cannot see published practice")
	}

	archived := &Practice{AuthorID: "person-1", Status: StatusArchived, Visibility: VisibilityPublic}
	if archived.CanView("person-2") {
		t.Error("other person can see archived practice")
	}
}
