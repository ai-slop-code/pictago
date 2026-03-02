package utils

import (
	"fmt"
	"time"
)

func LastDayOfPreviousMonth(now time.Time) time.Time {
	firstOfThisMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	lastOfPrevMonth := firstOfThisMonth.AddDate(0, 0, -1)
	return time.Date(
		lastOfPrevMonth.Year(),
		lastOfPrevMonth.Month(),
		lastOfPrevMonth.Day(),
		23, 59, 59, int(time.Nanosecond*time.Second)-1,
		now.Location(),
	)
}

func FirstDayOfPreviousMonth(now time.Time) time.Time {
	lastOfPrevMonth := LastDayOfPreviousMonth(now)
	return time.Date(
		lastOfPrevMonth.Year(),
		lastOfPrevMonth.Month(),
		1,
		0,
		0,
		0,
		0,
		now.Location(),
	)
}

func SegmentSizeMinutes(start, end time.Time, numSegments int) (int64, error) {
	if numSegments <= 0 {
		return 0, fmt.Errorf("numSegments must be a positive number")
	}
	if end.Before(start) || end.Equal(start) {
		return 0, fmt.Errorf("end date must be after start date")
	}
	totalMinutes := int64(end.Sub(start).Minutes())
	if totalMinutes < int64(numSegments) {
		return 60, nil
	}
	segmentSize := totalMinutes / int64(numSegments)
	if segmentSize < 1 {
		return 1, nil
	}
	return segmentSize, nil
}
