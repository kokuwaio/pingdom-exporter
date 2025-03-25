package pingdom

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTagLabel(t *testing.T) {
	testCases := []struct {
		tag string
		regex string
	}{
		{
			tag: "key:value",
			regex: "^([a-zA-Z0-9_]+):(.+)$",
		},
		{
			tag: "key-value",
			regex: "^([a-zA-Z0-9_]+)-(.+)$",
		},
		{
			tag: "key@value",
			regex: "^([a-zA-Z0-9_]+)@(.+)$",
		},
	}

	for _, testCase := range testCases {
		result, _ := TagLabel(testCase.tag, testCase.regex)
		assert.Equal(t, result.Formatted, 1)
	}
}
