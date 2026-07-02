package validators

import "testing"

func TestEscapeLikePattern(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"plain", "plain"},
		{"100%", `100\%`},
		{"a_b", `a\_b`},
		{`back\slash`, `back\\slash`},
		{"%_\\", `\%\_\\`},
	}

	for _, c := range cases {
		if got := EscapeLikePattern(c.in); got != c.want {
			t.Errorf("EscapeLikePattern(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
