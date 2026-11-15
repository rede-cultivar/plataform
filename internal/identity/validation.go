package identity

import (
	"net/mail"
	"regexp"
	"strings"
	"unicode/utf8"
)

const (
	minPasswordLength = 12
	maxPasswordLength = 128
	maxEmailLength    = 254
	maxDisplayName    = 80
)

var handlePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{2,29}$`)

// NormalizeEmail remove espaços e converte o e-mail para minúsculas.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// NormalizeHandle remove espaços e converte o identificador para minúsculas.
func NormalizeHandle(handle string) string {
	return strings.ToLower(strings.TrimSpace(handle))
}

// ValidateEmail verifica se o e-mail possui formato aceitável.
func ValidateEmail(email string) error {
	if email == "" || len(email) > maxEmailLength {
		return ErrInvalidEmail
	}

	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email {
		return ErrInvalidEmail
	}

	return nil
}

// ValidateHandle verifica o formato do identificador público.
//
// O identificador deve começar com uma letra e conter apenas letras
// minúsculas, números e sublinhado, entre 3 e 30 caracteres.
func ValidateHandle(handle string) error {
	if !handlePattern.MatchString(handle) {
		return ErrInvalidHandle
	}
	return nil
}

// ValidateDisplayName verifica o nome de exibição.
func ValidateDisplayName(name string) error {
	length := utf8.RuneCountInString(strings.TrimSpace(name))
	if length == 0 || length > maxDisplayName {
		return ErrInvalidDisplayName
	}
	return nil
}

// ValidatePassword verifica o comprimento da senha.
func ValidatePassword(password string) error {
	length := utf8.RuneCountInString(password)
	if length < minPasswordLength || length > maxPasswordLength {
		return ErrWeakPassword
	}
	return nil
}
