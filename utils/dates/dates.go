package dates

import "time"

const dateFormat = "1/2/2006"

// ParseDate create a Date from the passed string in format "M/D/YYYY"
func ParseDate(date string) (time.Time, error) {
	return time.Parse(dateFormat, date)
}
