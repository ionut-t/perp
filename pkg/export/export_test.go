package export

import (
	"testing"
)

func TestGenerateUniqueName_NoConflict(t *testing.T) {
	names := []string{"foo", "bar"}
	result := generateUniqueName("baz", names)
	if result != "baz" {
		t.Errorf("expected 'baz', got '%s'", result)
	}
}

func TestGenerateUniqueName_OneConflict(t *testing.T) {
	names := []string{"foo", "bar", "baz"}
	result := generateUniqueName("baz", names)
	if result != "baz-1" {
		t.Errorf("expected 'baz-1', got '%s'", result)
	}
}

func TestGenerateUniqueName_MultipleConflicts(t *testing.T) {
	names := []string{"foo", "bar", "baz", "baz-1", "baz-2"}
	result := generateUniqueName("baz", names)
	if result != "baz-3" {
		t.Errorf("expected 'baz-3', got '%s'", result)
	}
}

func TestGenerateUniqueName_ConflictWithSimilarNames(t *testing.T) {
	names := []string{"baz", "baz-1", "baz-2", "baz-10"}
	result := generateUniqueName("baz", names)
	if result != "baz-3" {
		t.Errorf("expected 'baz-3', got '%s'", result)
	}
}

func TestGenerateUniqueName_EmptyNames(t *testing.T) {
	names := []string{}
	result := generateUniqueName("foo", names)
	if result != "foo" {
		t.Errorf("expected 'foo', got '%s'", result)
	}
}
