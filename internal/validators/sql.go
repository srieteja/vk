package validators

import "strings"

// EscapeLikePattern escapes SQL LIKE wildcard metacharacters (\, %, _) in
// user-supplied input so it can be safely embedded in a LIKE pattern with
// an explicit ESCAPE '\' clause. Without this, user input containing % or _
// is interpreted as a wildcard instead of a literal character.
func EscapeLikePattern(s string) string {
	replacer := strings.NewReplacer(
		`\`, `\\`,
		`%`, `\%`,
		`_`, `\_`,
	)
	return replacer.Replace(s)
}
