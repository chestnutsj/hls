package tools

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateUniqueFilename(t *testing.T) {
	// Test case 1: Test with an existing file
	original := "testfile.txt"
	expected := "testfile_1.txt"
	path := filepath.Join(".", original)
	err := os.WriteFile(path, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(path)

	result, err := GenerateUniqueFilename(original)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}

	// Test case 2: Test with a non-existing file
	original = "nonexistingfile.txt"
	expected = "nonexistingfile.txt"
	result, err = GenerateUniqueFilename(original)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func Test_FindPositions(t *testing.T) {

	total := int64(1000) // 假设total非常大
	// 50 ,10, 100  = 160
	covered := []int64{22, 72, 120, 130, 500, 600}

	N := int64(100)
	uncovered := FindUncoveredPositions(total, covered, N)
	x := int64(0)

	for k, v := range uncovered {
		t.Log(k, v)
		x += v - k
	}
	if total-160 != x {
		t.Fatal("wrong")
	}
	t.Log(len(uncovered))

}

func Test_util(t *testing.T) {
	d := filepath.Dir("./test/xx.d/")
	t.Log(d)
}
