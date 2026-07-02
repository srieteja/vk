// Package log provides helpers for redacting PII before it reaches log
// output, so application logs can be shipped to third-party aggregators
// without leaking user emails or secrets.
package log

import "strings"

// MaskEmail redacts most of an email's local part, keeping the first
// character and the domain so log entries for the same user can still be
// correlated without exposing the full address.
func MaskEmail(email string) string {
	at := strings.IndexByte(email, '@')
	if at <= 0 {
		return "***"
	}
	local := email[:at]
	domain := email[at:]
	if len(local) <= 1 {
		return "*" + domain
	}
	return local[:1] + strings.Repeat("*", len(local)-1) + domain
}
