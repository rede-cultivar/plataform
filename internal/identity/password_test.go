package identity

import (
	"strings"
	"testing"
)

func TestHashPasswordAndVerify(t *testing.T) {
	hash, err := HashPassword("uma-senha-bem-longa")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	if !strings.HasPrefix(hash, "$argon2id$") {
		t.Fatalf("HashPassword() = %q, want Argon2id PHC format", hash)
	}
	if !VerifyPassword(hash, "uma-senha-bem-longa") {
		t.Fatal("VerifyPassword() = false, want true for the original password")
	}
	if VerifyPassword(hash, "outra-senha-qualquer") {
		t.Fatal("VerifyPassword() = true, want false for a different password")
	}
}

func TestVerifyPasswordRejectsMalformedHash(t *testing.T) {
	cases := []string{
		"",
		"texto-puro",
		"$argon2id$v=19$m=65536,t=3,p=4$sal$hash",
		"$bcrypt$v=19$m=65536,t=3,p=4$c2FsdA$aGFzaA",
	}

	for _, encoded := range cases {
		if VerifyPassword(encoded, "qualquer-senha") {
			t.Errorf("VerifyPassword(%q) = true, want false", encoded)
		}
	}
}

func TestHashPasswordUsesUniqueSalts(t *testing.T) {
	first, err := HashPassword("uma-senha-bem-longa")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	second, err := HashPassword("uma-senha-bem-longa")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	if first == second {
		t.Fatal("HashPassword() generated identical hashes for the same password")
	}
}
