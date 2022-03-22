package dates

import "time"

const dateFormat = "01-02-2006"

// ParseDate create a new Date from the passed string in format "MM-DD-YYYY"
func ParseDate(date string) (time.Time, error) {
	return time.Parse(dateFormat, date)
}
