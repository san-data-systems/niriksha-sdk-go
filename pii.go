package nirikshaai

import "regexp"

// Package-level compiled regexes for PII detection.
// These are safe for concurrent use once initialised.
var (
	_reEmail = regexp.MustCompile(
		`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`,
	)
	_rePhone = regexp.MustCompile(
		`(\+?1[\s.\-]?)?\(?\d{3}\)?[\s.\-]?\d{3}[\s.\-]?\d{4}`,
	)
	_reSSN = regexp.MustCompile(
		`\b\d{3}-\d{2}-\d{4}\b`,
	)
	_reCreditCard = regexp.MustCompile(
		`\b(?:\d[ -]?){13,16}\b`,
	)
)

// RedactPII replaces common PII patterns with a placeholder.
// Handles: email addresses, US phone numbers, US SSNs, credit card numbers.
// The replacements are applied in order: email, phone, SSN, credit card.
func RedactPII(s string) string {
	s = _reEmail.ReplaceAllString(s, "[REDACTED_EMAIL]")
	s = _rePhone.ReplaceAllString(s, "[REDACTED_PHONE]")
	s = _reSSN.ReplaceAllString(s, "[REDACTED_SSN]")
	s = _reCreditCard.ReplaceAllString(s, "[REDACTED_CC]")
	return s
}
