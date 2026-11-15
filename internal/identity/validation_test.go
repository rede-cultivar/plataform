package identity

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateEmail(t *testing.T) {
	valid := []string{
		"ana@example.com",
		"ana.silva+cultivo@example.org",
	}
	for _, email := range valid {
		if err := ValidateEmail(email); err != nil {
			t.Errorf("ValidateEmail(%q) error = %v, want nil", email, err)
		}
	}

	invalid := []string{
		"",
		"sem-arroba",
		"ana@",
		"Ana <ana@example.com>",
		strings.Repeat("a", 250) + "@example.com",
	}
	for _, email := range invalid {
		if err := ValidateEmail(email); !errors.Is(err, ErrInvalidEmail) {
			t.Errorf("ValidateEmail(%q) error = %v, want ErrInvalidEmail", email, err)
		}
	}
}

func TestValidateHandle(t *testing.T) {
	valid := []string{"ana", "horta_comunitaria", "a1b", strings.Repeat("a", 30)}
	for _, handle := range valid {
		if err := ValidateHandle(handle); err != nil {
			t.Errorf("ValidateHandle(%q) error = %v, want nil", handle, err)
		}
	}

	invalid := []string{"", "ab", "1ana", "Ana", "ana-silva", strings.Repeat("a", 31)}
	for _, handle := range invalid {
		if err := ValidateHandle(handle); !errors.Is(err, ErrInvalidHandle) {
			t.Errorf("ValidateHandle(%q) error = %v, want ErrInvalidHandle", handle, err)
		}
	}
}

func TestValidateDisplayName(t *testing.T) {
	if err := ValidateDisplayName("Ana Silva"); err != nil {
		t.Errorf("ValidateDisplayName() error = %v, want nil", err)
	}
	if err := ValidateDisplayName("   "); !errors.Is(err, ErrInvalidDisplayName) {
		t.Errorf("ValidateDisplayName() error = %v, want ErrInvalidDisplayName", err)
	}
	if err := ValidateDisplayName(strings.Repeat("a", 81)); !errors.Is(err, ErrInvalidDisplayName) {
		t.Errorf("ValidateDisplayName() error = %v, want ErrInvalidDisplayName", err)
	}
}

func TestValidatePassword(t *testing.T) {
	if err := ValidatePassword("senha-com-12"); err != nil {
		t.Errorf("ValidatePassword() error = %v, want nil", err)
	}
	if err := ValidatePassword("curta"); !errors.Is(err, ErrWeakPassword) {
		t.Errorf("ValidatePassword() error = %v, want ErrWeakPassword", err)
	}
	if err := ValidatePassword(strings.Repeat("a", 129)); !errors.Is(err, ErrWeakPassword) {
		t.Errorf("ValidatePassword() error = %v, want ErrWeakPassword", err)
	}
}
