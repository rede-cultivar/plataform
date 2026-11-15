package cultivation

import (
	"errors"
	"testing"
	"time"
)

func validCycleInput() CycleInput {
	return CycleInput{
		Name:        "Tomate no quintal",
		Species:     "Solanum lycopersicum",
		Environment: EnvironmentOutdoor,
		Stage:       StageSeedling,
		Visibility:  VisibilityPrivate,
		StartedOn:   time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
	}
}

func TestValidateCycleInput(t *testing.T) {
	input := validCycleInput()
	if err := ValidateCycleInput(input); err != nil {
		t.Fatalf("valid input rejected: %v", err)
	}

	input.Name = ""
	if err := ValidateCycleInput(input); err == nil {
		t.Fatal("empty name accepted")
	}

	input = validCycleInput()
	input.StartedOn = time.Time{}
	if err := ValidateCycleInput(input); err == nil {
		t.Fatal("zero start date accepted")
	}
}

func TestValidateCycleUpdateInput(t *testing.T) {
	input := CycleUpdateInput{
		Name:        "Tomate no quintal",
		Species:     "Solanum lycopersicum",
		Environment: EnvironmentOutdoor,
		Stage:       StageFruiting,
		Status:      StatusActive,
		Visibility:  VisibilityPublic,
		StartedOn:   time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
	}

	if err := ValidateCycleUpdateInput(input); err != nil {
		t.Fatalf("valid input rejected: %v", err)
	}

	before := input.StartedOn.Add(-24 * time.Hour)
	input.FinishedOn = &before
	if err := ValidateCycleUpdateInput(input); err == nil {
		t.Fatal("finish date before start accepted")
	}
}

func TestNormalizeEntryInputDeduplicatesLists(t *testing.T) {
	input := NormalizeEntryInput(EntryInput{
		Interventions: []string{" rega ", "rega", "", "poda"},
		Occurrences:   []string{" praga ", "praga"},
	})

	if len(input.Interventions) != 2 {
		t.Fatalf("interventions = %v, want 2 items", input.Interventions)
	}
	if len(input.Occurrences) != 1 {
		t.Fatalf("occurrences = %v, want 1 item", input.Occurrences)
	}
	if input.Visibility != EntryVisibilityInherit {
		t.Errorf("visibility = %q, want inherit", input.Visibility)
	}
}

func TestValidateEntryInputRejectsNegativeHarvest(t *testing.T) {
	quantity := -1.0
	err := ValidateEntryInput(EntryInput{
		OccurredAt:      time.Now(),
		HarvestQuantity: &quantity,
		Visibility:      EntryVisibilityInherit,
	})

	var validation *ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("error = %v, want ValidationError", err)
	}
	if validation.Field != "harvest_quantity" {
		t.Errorf("field = %q, want harvest_quantity", validation.Field)
	}
}

func TestEntryEffectiveVisibility(t *testing.T) {
	cases := []struct {
		name      string
		cycle     Visibility
		entry     EntryVisibility
		effective Visibility
	}{
		{"private cycle hides public entry", VisibilityPrivate, EntryVisibilityPublic, VisibilityPrivate},
		{"inherit uses cycle", VisibilityPublic, EntryVisibilityInherit, VisibilityPublic},
		{"entry cannot exceed followers", VisibilityFollowers, EntryVisibilityPublic, VisibilityFollowers},
		{"private entry in public cycle", VisibilityPublic, EntryVisibilityPrivate, VisibilityPrivate},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cycle := &Cycle{Visibility: tc.cycle}
			entry := &Entry{Visibility: tc.entry}
			if got := entry.EffectiveVisibility(cycle); got != tc.effective {
				t.Errorf("EffectiveVisibility() = %q, want %q", got, tc.effective)
			}
		})
	}
}

func TestCycleCanView(t *testing.T) {
	private := &Cycle{PersonID: "person-1", Visibility: VisibilityPrivate}
	if private.CanView("") {
		t.Error("anonymous viewer can see private cycle")
	}
	if private.CanView("person-2") {
		t.Error("other person can see private cycle")
	}
	if !private.CanView("person-1") {
		t.Error("owner cannot see private cycle")
	}

	public := &Cycle{PersonID: "person-1", Visibility: VisibilityPublic}
	if !public.CanView("") {
		t.Error("anonymous viewer cannot see public cycle")
	}
}
