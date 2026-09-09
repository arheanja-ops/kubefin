package cli

import "testing"

func TestParseWindowDays(t *testing.T) {
	cases := []struct {
		in      string
		want    int
		wantErr bool
	}{
		{"30d", 30, false},
		{"7d", 7, false},
		{"3m", 90, false},
		{"", 30, false},
		{"0d", 0, true},
		{"-5d", 0, true},
		{"10x", 0, true},
		{"abc", 0, true},
	}
	for _, c := range cases {
		got, err := parseWindowDays(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("parseWindowDays(%q) esperaba error", c.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseWindowDays(%q) error inesperado: %v", c.in, err)
		}
		if got != c.want {
			t.Errorf("parseWindowDays(%q) = %d, esperaba %d", c.in, got, c.want)
		}
	}
}
