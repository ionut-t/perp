package export

import (
	"testing"
)

func TestGenerateUniqueName_NoConflict(t *testing.T) {
	records := []Record{
		{Name: "foo"},
		{Name: "bar"},
	}
	name := "baz"
	got := generateUniqueName(name, records)
	want := "baz"
	if got != want {
		t.Errorf("generateUniqueName(%q, records) = %q; want %q", name, got, want)
	}
}

func TestGenerateUniqueName_OneConflict(t *testing.T) {
	records := []Record{
		{Name: "foo"},
		{Name: "bar"},
	}
	name := "foo"
	got := generateUniqueName(name, records)
	want := "foo-1"
	if got != want {
		t.Errorf("generateUniqueName(%q, records) = %q; want %q", name, got, want)
	}
}

func TestGenerateUniqueName_MultipleConflicts(t *testing.T) {
	records := []Record{
		{Name: "foo"},
		{Name: "foo-1"},
		{Name: "foo-2"},
	}
	name := "foo"
	got := generateUniqueName(name, records)
	want := "foo-3"
	if got != want {
		t.Errorf("generateUniqueName(%q, records) = %q; want %q", name, got, want)
	}
}

func TestGenerateUniqueName_ConflictWithSimilarButNotExact(t *testing.T) {
	records := []Record{
		{Name: "foo"},
		{Name: "foobar"},
	}
	name := "foo"
	got := generateUniqueName(name, records)
	want := "foo-1"
	if got != want {
		t.Errorf("generateUniqueName(%q, records) = %q; want %q", name, got, want)
	}
}

func TestGenerateUniqueName_EmptyRecords(t *testing.T) {
	records := []Record{}
	name := "foo"
	got := generateUniqueName(name, records)
	want := "foo"
	if got != want {
		t.Errorf("generateUniqueName(%q, records) = %q; want %q", name, got, want)
	}
}
