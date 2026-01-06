// Package testutil provides shared testing utilities for the OpenPairing backend.
package testutil

import "time"

// StrPtr returns a pointer to the given string value.
func StrPtr(s string) *string {
	return &s
}

// IntPtr returns a pointer to the given int value.
func IntPtr(i int) *int {
	return &i
}

// Int64Ptr returns a pointer to the given int64 value.
func Int64Ptr(i int64) *int64 {
	return &i
}

// Float64Ptr returns a pointer to the given float64 value.
func Float64Ptr(f float64) *float64 {
	return &f
}

// BoolPtr returns a pointer to the given bool value.
func BoolPtr(b bool) *bool {
	return &b
}

// TimePtr returns a pointer to the given time.Time value.
func TimePtr(t time.Time) *time.Time {
	return &t
}
