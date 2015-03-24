package spawn

import (
	"testing"
)

func test(t *testing.T, expected bool, messages ...interface{}) {
	if !expected {
		t.Error(messages)
	}
}

func TestIsAlphaNumeric(t *testing.T) {
	str1 := "abcd1234.35_df-12"
	test(t, isAlphaNumeric(str1), "Expected "+str1+" is alpha numeric, got false")
	str2 := "abcd1234.%*35_df-12"
	test(t, !isAlphaNumeric(str2), "Expected "+str2+" is not alpha numeric, got true")
}
