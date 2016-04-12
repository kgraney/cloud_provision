package aws

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDeduplicateBufferStream(t *testing.T) {
	testCases := []struct {
		Previous string
		Current  string
		Expected string
	}{{
		Previous: "abc",
		Current:  "abcdef",
		Expected: "def",
	}, {
		Previous: "",
		Current:  "abcdef",
		Expected: "abcdef",
	}, {
		Previous: "",
		Current:  "",
		Expected: "",
	}, {
		Previous: "abcd",
		Current:  "",
		Expected: "",
	}, {
		Previous: "abcd",
		Current:  "efgh",
		Expected: "efgh",
	}}

	for _, tc := range testCases {
		assert.Equal(t, tc.Expected, DeduplicatedLogStream(tc.Previous, tc.Current))
	}
}
