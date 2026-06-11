package excel

import "testing"

func TestFormatTransactionDate(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "MySQL datetime with seconds", input: "2026-06-11 14:05:30", expected: "2026-06-11 14:05"},
		{name: "RFC3339 UTC", input: "2026-06-11T14:05:30Z", expected: "2026-06-11 14:05"},
		{name: "RFC3339 +08:00", input: "2026-06-11T14:05:30+08:00", expected: "2026-06-11 14:05"},
		{name: "RFC3339 -08:00", input: "2026-06-11T14:05:30-08:00", expected: "2026-06-11 14:05"},
		{name: "short date time", input: "2026-06-11 14:05", expected: "2026-06-11 14:05"},
		{name: "date only", input: "2026-06-11", expected: "2026-06-11 00:00"},
		{name: "unparseable with seconds", input: "2026/06/11 14:05:30", expected: "2026/06/11 14:05"},
		{name: "unparseable short", input: "20260611 1405", expected: "20260611 1405"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := formatTransactionDate(c.input)
			if got != c.expected {
				t.Fatalf("formatTransactionDate(%q) = %q; want %q", c.input, got, c.expected)
			}
		})
	}
}
