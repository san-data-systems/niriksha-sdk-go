package nirikshaai_test

import (
	"testing"

	nirikshaai "github.com/san-data-systems/niriksha-sdk-go"
)

func TestRedactPII_Email(t *testing.T) {
	got := nirikshaai.RedactPII("contact user@example.com please")
	if got != "contact [REDACTED_EMAIL] please" {
		t.Errorf("unexpected: %q", got)
	}
}

func TestRedactPII_Phone(t *testing.T) {
	got := nirikshaai.RedactPII("call 555-123-4567 now")
	if got != "call [REDACTED_PHONE] now" {
		t.Errorf("unexpected: %q", got)
	}
}

func TestRedactPII_SSN(t *testing.T) {
	got := nirikshaai.RedactPII("ssn 123-45-6789 found")
	if got != "ssn [REDACTED_SSN] found" {
		t.Errorf("unexpected: %q", got)
	}
}

func TestRedactPII_CreditCard(t *testing.T) {
	got := nirikshaai.RedactPII("card 4111 1111 1111 1111 stored")
	if got != "card [REDACTED_CC] stored" {
		t.Errorf("unexpected: %q", got)
	}
}

func TestRedactPII_NoMatch(t *testing.T) {
	input := "hello world"
	got := nirikshaai.RedactPII(input)
	if got != input {
		t.Errorf("unexpected mutation: %q", got)
	}
}
