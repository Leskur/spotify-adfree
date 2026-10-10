package update

import "testing"

func TestIsNewer(t *testing.T) {
	cases := []struct {
		latest, current string
		newer           bool
	}{
		{"0.0.2", "0.0.1", true},
		{"v0.1.0", "0.0.9", true},
		{"0.0.1", "0.0.1", false},
		{"0.0.1", "0.0.2", false},
		{"1.0", "0.9.9", true},
		{"v1.2.3-beta", "1.2.2", true},
	}
	for _, c := range cases {
		got, err := isNewer(c.latest, c.current)
		if err != nil {
			t.Fatalf("%s vs %s: %v", c.latest, c.current, err)
		}
		if got != c.newer {
			t.Fatalf("%s vs %s: got %v want %v", c.latest, c.current, got, c.newer)
		}
	}
}
