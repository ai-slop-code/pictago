package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLastDayOfPreviousMonthFrom(t *testing.T) {
	t.Run("Should be able to determine last day of previous month", func(t *testing.T) {
		loc := time.UTC
		tests := []struct {
			now      time.Time
			expected time.Time
		}{
			{
				now:      time.Date(2024, 3, 15, 0, 0, 0, 0, loc),
				expected: time.Date(2024, 2, 29, 23, 59, 59, 999999999, loc),
			},
			{
				now:      time.Date(2023, 3, 15, 0, 0, 0, 0, loc),
				expected: time.Date(2023, 2, 28, 23, 59, 59, 999999999, loc),
			},
			{
				now:      time.Date(2023, 1, 1, 0, 0, 0, 0, loc),
				expected: time.Date(2022, 12, 31, 23, 59, 59, 999999999, loc),
			},
		}

		for _, tt := range tests {
			got := LastDayOfPreviousMonth(tt.now)
			if !got.Equal(tt.expected) {
				t.Errorf("LastDayOfPreviousMonthFrom(%v) = %v, want %v", tt.now, got, tt.expected)
			}
		}
	})
}

func TestFirstDayOfPreviousMonthFrom(t *testing.T) {
	t.Run("Should be able to determine first day of previous month", func(t *testing.T) {
		loc := time.UTC
		tests := []struct {
			now      time.Time
			expected time.Time
		}{
			{
				now:      time.Date(2024, 3, 15, 0, 0, 0, 0, loc),
				expected: time.Date(2024, 2, 1, 0, 0, 0, 0, loc),
			},
			{
				now:      time.Date(2023, 1, 1, 0, 0, 0, 0, loc),
				expected: time.Date(2022, 12, 1, 0, 0, 0, 0, loc),
			},
		}

		for _, tt := range tests {
			got := FirstDayOfPreviousMonth(tt.now)
			if !got.Equal(tt.expected) {
				t.Errorf("FirstDayOfPreviousMonthFrom(%v) = %v, want %v", tt.now, got, tt.expected)
			}
		}
	})
}

func TestSegmentSizeMinutes(t *testing.T) {
	loc := time.UTC

	t.Run("Should return correct segment size for a 31-day period", func(t *testing.T) {
		start := time.Date(2024, time.January, 1, 0, 0, 0, 0, loc)
		end := time.Date(2024, time.February, 1, 0, 0, 0, 0, loc)
		// January 2024 has 31 days = 44640 minutes
		got, err := SegmentSizeMinutes(start, end, 31)
		require.NoError(t, err)
		assert.Equal(t, int64(1440), got) // 1 day in minutes
	})

	t.Run("Should return correct segment size for a leap year February", func(t *testing.T) {
		start := time.Date(2024, time.February, 1, 0, 0, 0, 0, loc)
		end := time.Date(2024, time.March, 1, 0, 0, 0, 0, loc)
		// February 2024 has 29 days = 41760 minutes
		got, err := SegmentSizeMinutes(start, end, 29)
		require.NoError(t, err)
		assert.Equal(t, int64(1440), got)
	})

	t.Run("Should return error for non-positive numSegments", func(t *testing.T) {
		start := time.Date(2024, time.January, 1, 0, 0, 0, 0, loc)
		end := time.Date(2024, time.February, 1, 0, 0, 0, 0, loc)
		got, err := SegmentSizeMinutes(start, end, 0)
		require.Error(t, err)
		assert.Equal(t, int64(0), got)
	})

	t.Run("Should return correct segment size for December", func(t *testing.T) {
		start := time.Date(2023, time.December, 1, 0, 0, 0, 0, loc)
		end := time.Date(2024, time.January, 1, 0, 0, 0, 0, loc)
		// December 2023 has 31 days = 44640 minutes
		got, err := SegmentSizeMinutes(start, end, 31)
		require.NoError(t, err)
		assert.Equal(t, int64(1440), got)
	})

	t.Run("Should not return segment size less than 1 minute", func(t *testing.T) {
		start := time.Date(2024, time.January, 1, 0, 0, 0, 0, loc)
		end := time.Date(2024, time.February, 1, 0, 0, 0, 0, loc)
		got, err := SegmentSizeMinutes(start, end, 100000)
		require.NoError(t, err)
		assert.Equal(t, int64(60), got)
	})

	t.Run("Should return 1 when totalMinutes < numSegments", func(t *testing.T) {
		start := time.Date(2024, time.February, 1, 0, 0, 0, 0, loc)
		end := time.Date(2024, time.March, 1, 0, 0, 0, 0, loc)
		// February 2024 has 41760 minutes, 50000 segments -> should return 1
		got, err := SegmentSizeMinutes(start, end, 50000)
		require.NoError(t, err)
		assert.Equal(t, int64(60), got)
	})

	t.Run("Should return error when end is before start", func(t *testing.T) {
		start := time.Date(2024, time.February, 1, 0, 0, 0, 0, loc)
		end := time.Date(2024, time.January, 1, 0, 0, 0, 0, loc)
		got, err := SegmentSizeMinutes(start, end, 10)
		require.Error(t, err)
		assert.Equal(t, int64(0), got)
	})

	t.Run("Should work with custom date ranges", func(t *testing.T) {
		start := time.Date(2024, time.January, 15, 10, 30, 0, 0, loc)
		end := time.Date(2024, time.January, 16, 10, 30, 0, 0, loc)
		// 1 day = 1440 minutes, 24 segments = 60 minutes each
		got, err := SegmentSizeMinutes(start, end, 24)
		require.NoError(t, err)
		assert.Equal(t, int64(60), got)
	})
}
