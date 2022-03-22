package dates

import "time"

const dateFormat = "05-15-2021"

// ParseDate create a new Date from the passed string in format "05-15-2021"
func ParseDate(date string) (time.Time, error) {
	return time.Parse(dateFormat, date)
}
