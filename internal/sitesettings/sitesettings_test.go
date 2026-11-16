package sitesettings

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateAcceptsKnownModes(t *testing.T) {
	for _, mode := range []string{ModeOpen, ModeApproved, ModeNone} {
		if err := Validate(UpdateInput{RegistrationsMode: mode}); err != nil {
			t.Errorf("Validate(%q) = %v, want nil", mode, err)
		}
	}
}

func TestValidateRejectsUnknownMode(t *testing.T) {
	err := Validate(UpdateInput{RegistrationsMode: "closed"})
	if !errors.Is(err, ErrInvalidMode) {
		t.Fatalf("Validate() = %v, want ErrInvalidMode", err)
	}
}

func TestValidateRejectsLongClosedMessage(t *testing.T) {
	err := Validate(UpdateInput{
		RegistrationsMode:          ModeNone,
		ClosedRegistrationsMessage: strings.Repeat("a", MaxClosedMessageLength+1),
	})
	if !errors.Is(err, ErrClosedMessageTooLong) {
		t.Fatalf("Validate() = %v, want ErrClosedMessageTooLong", err)
	}
}

func TestSettingsOpen(t *testing.T) {
	cases := []struct {
		mode string
		want bool
	}{
		{ModeOpen, true},
		{ModeApproved, true},
		{ModeNone, false},
	}

	for _, tc := range cases {
		settings := &Settings{RegistrationsMode: tc.mode}
		if got := settings.Open(); got != tc.want {
			t.Errorf("Settings{%q}.Open() = %v, want %v", tc.mode, got, tc.want)
		}
	}
}

func TestSettingsRequiresApproval(t *testing.T) {
	if !(&Settings{RegistrationsMode: ModeApproved}).RequiresApproval() {
		t.Error("approved mode should require approval")
	}
	if (&Settings{RegistrationsMode: ModeOpen}).RequiresApproval() {
		t.Error("open mode should not require approval")
	}
}

func TestClosedMessageUsesDefaultWhenEmpty(t *testing.T) {
	settings := &Settings{RegistrationsMode: ModeNone}
	if got := settings.ClosedMessage(); got != DefaultClosedMessage {
		t.Errorf("ClosedMessage() = %q, want %q", got, DefaultClosedMessage)
	}
}

func TestClosedMessageUsesConfiguredText(t *testing.T) {
	settings := &Settings{
		RegistrationsMode:          ModeNone,
		ClosedRegistrationsMessage: "  Voltamos em breve.  ",
	}
	if got := settings.ClosedMessage(); got != "Voltamos em breve." {
		t.Errorf("ClosedMessage() = %q, want trimmed message", got)
	}
}
