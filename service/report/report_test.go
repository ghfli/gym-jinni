package report

import "testing"

func TestReportRange(t *testing.T) {
	for _, test := range []struct {
		start, end      string
		required, valid bool
	}{
		{"", "", false, true}, {"", "", true, false},
		{"2030-01-01T00:00:00Z", "2030-01-02T00:00:00Z", true, true},
		{"2030-01-02T00:00:00Z", "2030-01-01T00:00:00Z", true, false},
		{"2030-02-30T00:00:00Z", "2030-03-01T00:00:00Z", true, false},
		{"2030-01-01", "2030-01-02", true, false},
	} {
		_, _, err := reportRange(test.start, test.end, test.required)
		if (err == nil) != test.valid {
			t.Errorf("range %q/%q valid=%v err=%v", test.start, test.end, test.valid, err)
		}
	}
}
