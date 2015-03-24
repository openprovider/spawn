package spawn

import (
	"bufio"
	"encoding/json"
	"os"
	"testing"
)

// loadFixtures - loads fixtures
func loadFixtures(path string) (nodes []Node, err error) {
	_, err = os.Stat(path)
	if err != nil {
		return
	}
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()
	if err = json.NewDecoder(bufio.NewReader(file)).Decode(&nodes); err != nil {
		return
	}

	return
}

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
