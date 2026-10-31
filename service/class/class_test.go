package class

import (
	. "github.com/ghfli/gym-jinni/service/gen/go/class/v1alpha"
	"google.golang.org/genproto/googleapis/type/datetime"
	"testing"
)

func TestClassTimeAndCapacityBoundaries(t *testing.T) {
	for _, name := range []string{"valid", "unlimited", "missing_end", "reversed", "invalid_calendar", "negative_capacity", "minimum_above_maximum", "empty_description"} {
		t.Run(name, func(t *testing.T) {
			c := &Class{Description: "Yoga", StartTime: &datetime.DateTime{Year: 2030, Month: 2, Day: 1, Hours: 10}, EndTime: &datetime.DateTime{Year: 2030, Month: 2, Day: 1, Hours: 11}, MinHdcnt: 1, MaxHdcnt: 1}
			valid := false
			switch name {
			case "valid":
				valid = true
			case "unlimited":
				c.MaxHdcnt = 0
				valid = true
			case "missing_end":
				c.EndTime = nil
			case "reversed":
				c.EndTime.Hours = 9
			case "invalid_calendar":
				c.StartTime.Day = 30
			case "negative_capacity":
				c.MinHdcnt = -1
			case "minimum_above_maximum":
				c.MinHdcnt = 2
			case "empty_description":
				c.Description = " "
			}
			if err := validateClass(c); (err == nil) != valid {
				t.Fatalf("valid=%t error=%v", valid, err)
			}
		})
	}
}
