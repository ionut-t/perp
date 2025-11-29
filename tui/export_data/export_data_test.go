package export_data

import (
	"testing"

	"github.com/ionut-t/perp/pkg/server"
	"github.com/ionut-t/perp/store/export"
)

// mockStore implements export.Store for testing
type mockStore struct {
	records         []export.Record
	loadErr         error
	currentRecord   export.Record
	editorName      string
	setCurrentCalls []string
	updateCalls     []export.Record
	updateErr       error
	deleteCalls     []export.Record
	deleteErr       error
	renameCalls     []struct {
		record  *export.Record
		newName string
	}
	renameErr error
}

func (m *mockStore) Load() ([]export.Record, error) {
	return m.records, m.loadErr
}

func (m *mockStore) GetCurrentRecord() export.Record {
	return m.currentRecord
}

func (m *mockStore) SetCurrentRecordName(name string) {
	m.setCurrentCalls = append(m.setCurrentCalls, name)
	// Update current record if it exists in records
	for _, r := range m.records {
		if r.Name == name {
			m.currentRecord = r
			break
		}
	}
}

func (m *mockStore) Update(r export.Record) error {
	m.updateCalls = append(m.updateCalls, r)
	return m.updateErr
}

func (m *mockStore) Delete(r export.Record) error {
	m.deleteCalls = append(m.deleteCalls, r)
	// Remove from records if no error
	if m.deleteErr == nil {
		newRecords := []export.Record{}
		for _, rec := range m.records {
			if rec.Name != r.Name {
				newRecords = append(newRecords, rec)
			}
		}
		m.records = newRecords
		// Update current record if needed
		if len(m.records) > 0 {
			m.currentRecord = m.records[0]
		} else {
			m.currentRecord = export.Record{}
		}
	}
	return m.deleteErr
}

func (m *mockStore) Rename(r *export.Record, newName string) error {
	m.renameCalls = append(m.renameCalls, struct {
		record  *export.Record
		newName string
	}{r, newName})
	if m.renameErr == nil {
		r.Name = newName
	}
	return m.renameErr
}

func (m *mockStore) Editor() string {
	if m.editorName != "" {
		return m.editorName
	}
	return "vim"
}

func (m *mockStore) GetPath(r export.Record) string {
	return "/fake/path/" + r.Name
}

func TestNew_WithNoRecords(t *testing.T) {
	t.Parallel()

	store := &mockStore{
		records:       []export.Record{},
		currentRecord: export.Record{},
	}
	width, height := 100, 40
	server := server.Server{
		Name:     "test-server",
		Database: "test-db",
	}
	m := New(store, server, width, height)

	// Basic sanity check that model was created
	if m.Model == nil {
		t.Error("expected model to be created")
	}
}

// TODO: These tests need to be rewritten to work with the new generic split-view architecture
// Most of the internal implementation details are now handled by splitview.Model
// and are not directly accessible. Tests should focus on:
// 1. Testing the splitview package directly
// 2. Testing only the wrapper behavior (adapter, custom rendering, etc.)
// 3. Integration tests that verify end-to-end behavior

// The following tests have been commented out and need to be rewritten:
// - TestNew_WithRecords
// - TestNew_WithLoadError
// - TestModel_Update_WindowSizeMsg
// - TestModel_Update_KeyMsg_Quit
// - TestModel_Update_KeyMsg_SwitchToInsertMode
// - TestModel_Update_KeyMsg_ShiftNavigation
// - TestModel_Update_EditorSaveMsg
// - TestModel_Update_ListNavigation
// - TestModel_View_ErrorState
// - TestModel_View_DifferentModes
// - TestModel_handleWindowSize
// - TestModel_Update_EditorDeleteFileMsg_LastRecord
// - TestModel_Update_EditorRenameMsg
// - TestModel_statusBarView
// - TestModel_getAvailableSizes
// - TestNew_InitialFocus
// - TestModel_Update_KeyMsg_ExternalEditor

func TestProcessRecords(t *testing.T) {
	t.Parallel()

	records := []recordItem{
		{Record: &export.Record{Name: "rec1"}},
		{Record: &export.Record{Name: "rec2"}},
	}

	items := processRecords(records)

	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	item1 := items[0].(item)
	if item1.title != "rec1" {
		t.Errorf("expected first item title to be 'rec1', got %q", item1.title)
	}

	item2 := items[1].(item)
	if item2.title != "rec2" {
		t.Errorf("expected second item title to be 'rec2', got %q", item2.title)
	}
}

func TestItem_Interface(t *testing.T) {
	t.Parallel()

	i := item{
		title: "Test Title",
		desc:  "Test Description",
	}

	if i.Title() != "Test Title" {
		t.Errorf("expected Title() to return 'Test Title', got %q", i.Title())
	}
	if i.Description() != "Test Description" {
		t.Errorf("expected Description() to return 'Test Description', got %q", i.Description())
	}
	if i.FilterValue() != "Test Title" {
		t.Errorf("expected FilterValue() to return 'Test Title', got %q", i.FilterValue())
	}
}
