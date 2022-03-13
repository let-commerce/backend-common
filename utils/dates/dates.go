package dates

import "time"

const dateFormat = "2021-05-15"

// ParseDate create a new Date from the passed string in format "2021-05-15"
func ParseDate(date string) (time.Time, error) {
	return time.Parse(dateFormat, date)
}
