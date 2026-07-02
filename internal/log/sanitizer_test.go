package log

import "testing"

func TestMaskEmail(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"alice@example.com", "a****@example.com"},
		{"a@example.com", "*@example.com"},
		{"", "***"},
		{"not-an-email", "***"},
	}

	for _, c := range cases {
		if got := MaskEmail(c.in); got != c.want {
			t.Errorf("MaskEmail(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
