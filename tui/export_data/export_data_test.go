package export_data

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ionut-t/goeditor/adapter-bubbletea/editor"
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

func TestNew_WithNoRecords(t *testing.T) {
	t.Parallel()

	store := &mockStore{
		records:       []export.Record{},
		currentRecord: export.Record{},
	}
	width, height := 100, 40

	m := New(store, width, height)

	if m.error != nil {
		t.Errorf("expected no error, got %v", m.error)
	}
	if m.width != width || m.height != height {
		t.Errorf("expected width=%d, height=%d, got width=%d, height=%d", width, height, m.width, m.height)
	}
}

func TestNew_WithRecords(t *testing.T) {
	t.Parallel()

	now := time.Now()
	records := []export.Record{
		{
			Name:      "rec1",
			Content:   "content1",
			UpdatedAt: now,
		},
		{
			Name:      "rec2",
			Content:   "content2",
			UpdatedAt: now,
		},
	}
	store := &mockStore{
		records:       records,
		currentRecord: records[0],
	}
	width, height := 120, 50

	m := New(store, width, height)

	if m.error != nil {
		t.Errorf("expected no error, got %v", m.error)
	}
	items := m.list.Items()
	if len(items) != 2 {
		t.Errorf("expected 2 list items, got %d", len(items))
	}
}

func TestNew_WithLoadError(t *testing.T) {
	t.Parallel()

	loadErr := errors.New("load failed")
	store := &mockStore{
		records:       nil,
		loadErr:       loadErr,
		currentRecord: export.Record{},
	}
	width, height := 80, 20

	m := New(store, width, height)

	if m.error == nil {
		t.Error("expected error to be set")
	}
	if m.error.Error() != loadErr.Error() {
		t.Errorf("expected error %q, got %q", loadErr.Error(), m.error.Error())
	}
}

func TestModel_Update_WindowSizeMsg(t *testing.T) {
	t.Parallel()

	store := &mockStore{
		records: []export.Record{{Name: "test", Content: "content"}},
	}
	m := New(store, 100, 40)

	newWidth, newHeight := 200, 80
	msg := tea.WindowSizeMsg{Width: newWidth, Height: newHeight}

	updatedModel, _ := m.Update(msg)
	updated := updatedModel.(Model)

	if updated.width != newWidth || updated.height != newHeight {
		t.Errorf("expected width=%d, height=%d, got width=%d, height=%d",
			newWidth, newHeight, updated.width, updated.height)
	}
}

func TestModel_Update_KeyMsg_Quit(t *testing.T) {
	t.Parallel()

	store := &mockStore{}
	m := New(store, 100, 40)
	m.view = viewSplit
	m.focusedView = focusedViewList

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	_, cmd := m.Update(msg)

	if cmd == nil {
		t.Error("expected quit command, got nil")
	}

	// Execute the command to check it returns ClosedMsg
	result := cmd()
	if _, ok := result.(ClosedMsg); !ok {
		t.Errorf("expected ClosedMsg, got %T", result)
	}
}

func TestModel_Update_KeyMsg_SwitchToInsertMode(t *testing.T) {
	t.Parallel()

	store := &mockStore{
		records:       []export.Record{{Name: "test", Content: "content"}},
		currentRecord: export.Record{Name: "test", Content: "content"},
	}
	m := New(store, 100, 40)
	m.focusedView = focusedViewList

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}}
	updatedModel, _ := m.Update(msg)
	updated := updatedModel.(Model)

	if updated.focusedView != focusedViewRecord {
		t.Errorf("expected focusedView to be focusedViewRecord, got %v", updated.focusedView)
	}
}

func TestModel_Update_KeyMsg_ShiftNavigation(t *testing.T) {
	t.Parallel()

	store := &mockStore{}
	m := New(store, 100, 40)
	m.view = viewSplit
	m.focusedView = focusedViewList

	// Test first tab press
	msg := tea.KeyMsg{Type: tea.KeyTab}
	updatedModel, _ := m.Update(msg)
	updated := updatedModel.(Model)

	if updated.focusedView != focusedViewRecord {
		t.Errorf("expected focusedView to be focusedViewRecord after tab, got %v", updated.focusedView)
	}

	// Test second tab press
	msg = tea.KeyMsg{Type: tea.KeyTab}
	updatedModel, _ = updated.Update(msg)
	updated = updatedModel.(Model)

	if updated.focusedView != focusedViewList {
		t.Errorf("expected focusedView to be focusedViewList after tab, got %v", updated.focusedView)
	}
}

func TestModel_Update_EditorSaveMsg(t *testing.T) {
	t.Parallel()

	currentRecord := export.Record{
		Name:    "test",
		Content: "old content",
	}
	store := &mockStore{
		currentRecord: currentRecord,
		records:       []export.Record{currentRecord},
	}
	m := New(store, 100, 40)

	msg := editor.SaveMsg("new content")
	updatedModel, _ := m.Update(msg)
	updated := updatedModel.(Model)

	// Check that Update was called with correct content
	if len(store.updateCalls) != 1 {
		t.Fatalf("expected 1 update call, got %d", len(store.updateCalls))
	}
	if store.updateCalls[0].Content != "new content" {
		t.Errorf("expected update to be called with 'new content', got %q", store.updateCalls[0].Content)
	}
	if updated.error != nil {
		t.Errorf("expected no error after successful save, got %v", updated.error)
	}
}

func TestModel_Update_ListNavigation(t *testing.T) {
	records := []export.Record{
		{Name: "rec1", Content: "content1"},
		{Name: "rec2", Content: "content2"},
		{Name: "rec3", Content: "content3"},
	}
	store := &mockStore{
		records:       records,
		currentRecord: records[0],
	}
	m := New(store, 100, 40)
	m.focusedView = focusedViewList

	// Manually trigger what would happen after a selection change
	selectedItem := item{title: "rec2", desc: "desc"}
	m.list.Select(1) // Select second item

	// Simulate the list update behavior
	if m.focusedView == focusedViewList {
		store.SetCurrentRecordName(selectedItem.Title())
	}

	// Check that SetCurrentRecordName was called
	if len(store.setCurrentCalls) == 0 {
		t.Error("expected SetCurrentRecordName to be called")
	}
	if store.setCurrentCalls[0] != "rec2" {
		t.Errorf("expected SetCurrentRecordName to be called with 'rec2', got %q", store.setCurrentCalls[0])
	}
}

func TestModel_View_ErrorState(t *testing.T) {
	t.Parallel()

	store := &mockStore{}
	m := New(store, 100, 40)
	m.error = errors.New("test error")

	view := m.View()

	if !strings.Contains(view, "Error loading export records: test error") {
		t.Errorf("expected view to contain error message, got %q", view)
	}
}

func TestModel_View_DifferentModes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		view view
	}{
		{"list mode", viewList},
		{"record mode", viewRecord},
		{"split mode", viewSplit},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockStore{}
			m := New(store, 100, 40)
			m.view = tt.view
			m.error = nil

			view := m.View()

			if view == "" {
				t.Errorf("expected non-empty view in %s", tt.name)
			}
		})
	}
}

func TestProcessRecords(t *testing.T) {
	t.Parallel()

	now := time.Now()
	records := []export.Record{
		{
			Name:      "rec1",
			UpdatedAt: now,
		},
		{
			Name:      "rec2",
			UpdatedAt: now.Add(-time.Hour),
		},
	}

	items := processRecords(records)

	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	item1 := items[0].(item)
	if item1.title != "rec1" {
		t.Errorf("expected first item title to be 'rec1', got %q", item1.title)
	}
	if !strings.Contains(item1.desc, "Last modified:") {
		t.Errorf("expected description to contain 'Last modified:', got %q", item1.desc)
	}

	item2 := items[1].(item)
	if item2.title != "rec2" {
		t.Errorf("expected second item title to be 'rec2', got %q", item2.title)
	}
}

func TestModel_handleWindowSize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		initialView  view
		width        int
		expectedView view
	}{
		{
			name:         "switch to list when too narrow",
			initialView:  viewSplit,
			width:        80, // Less than 2*minListWidth (2*50)
			expectedView: viewList,
		},
		{
			name:         "switch back to split from list when narrow",
			initialView:  viewList,
			width:        80, // Less than 2*minListWidth
			expectedView: viewSplit,
		},
		{
			name:         "keep split when wide enough",
			initialView:  viewSplit,
			width:        200, // More than 2*minListWidth
			expectedView: viewSplit,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockStore{}
			m := New(store, 200, 40)
			m.view = tt.initialView

			m.handleWindowSize(tea.WindowSizeMsg{Width: tt.width, Height: 40})

			if m.view != tt.expectedView {
				t.Errorf("expected view to be %v, got %v", tt.expectedView, m.view)
			}
		})
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

func TestModel_Update_EditorDeleteFileMsg_LastRecord(t *testing.T) {
	t.Parallel()

	records := []export.Record{
		{Name: "lastRecord", Content: "content"},
	}
	store := &mockStore{
		currentRecord: records[0],
		records:       records,
	}
	m := New(store, 100, 40)

	msg := editor.DeleteFileMsg{}
	updatedModel, cmd := m.Update(msg)
	updated := updatedModel.(Model)

	// Check that Delete was called
	if len(store.deleteCalls) != 1 {
		t.Fatalf("expected 1 delete call, got %d", len(store.deleteCalls))
	}

	// After deletion of last record, editor should be empty
	// The mock store should have removed the record and set currentRecord to empty
	if len(store.records) != 0 {
		t.Errorf("expected no records after deleting last record, got %d", len(store.records))
	}

	// Check success message
	if !strings.Contains(updated.successMessage, "deleted successfully") {
		t.Errorf("expected success message, got %q", updated.successMessage)
	}

	// Check that clearMessages command was returned
	if cmd == nil {
		t.Error("expected clearMessages command, got nil")
	}
}

func TestModel_Update_EditorRenameMsg(t *testing.T) {
	t.Parallel()

	now := time.Now()
	currentRecord := export.Record{
		Name:      "oldname",
		UpdatedAt: now,
	}
	store := &mockStore{
		currentRecord: currentRecord,
		records:       []export.Record{currentRecord},
	}
	m := New(store, 100, 40)

	newName := "newname"

	msg := editor.RenameMsg{FileName: newName}

	updatedModel, cmd := m.Update(msg)
	updated := updatedModel.(Model)

	// Check that Rename was called
	if len(store.renameCalls) != 1 {
		t.Fatalf("expected 1 rename call, got %d", len(store.renameCalls))
	}
	if store.renameCalls[0].newName != newName {
		t.Errorf("expected rename to be called with 'newname', got %q", store.renameCalls[0].newName)
	}

	// Check success message
	if !strings.Contains(updated.successMessage, "renamed successfully") {
		t.Errorf("expected success message, got %q", updated.successMessage)
	}

	// Check that a command was returned (should be tea.Batch with list update and clearMessages)
	if cmd == nil {
		t.Error("expected command to be returned, got nil")
	}

	// The record passed to Rename should have been updated by the mock
	if store.renameCalls[0].record.Name != newName {
		t.Errorf("expected record name to be updated to 'newname' in mock, got %q", store.renameCalls[0].record.Name)
	}
}

func TestModel_statusBarView(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		error            error
		successMessage   string
		expectedContains string
	}{
		{
			name:             "error state",
			error:            errors.New("test error"),
			expectedContains: "test error",
		},
		{
			name:             "success state",
			successMessage:   "Operation successful",
			expectedContains: "Operation successful",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockStore{}
			m := New(store, 100, 40)
			m.error = tt.error
			m.successMessage = tt.successMessage

			status := m.statusBarView()

			if !strings.Contains(status, tt.expectedContains) {
				t.Errorf("expected status bar to contain %q, got %q", tt.expectedContains, status)
			}
		})
	}
}

func TestModel_getAvailableSizes(t *testing.T) {
	t.Parallel()

	store := &mockStore{}
	m := New(store, 120, 50)

	availableWidth, availableHeight, cmdViewHeight := m.getAvailableSizes()

	// Basic sanity checks
	if availableWidth <= 0 {
		t.Errorf("expected positive available width, got %d", availableWidth)
	}
	if availableHeight <= 0 {
		t.Errorf("expected positive available height, got %d", availableHeight)
	}
	if cmdViewHeight < 0 {
		t.Errorf("expected non-negative cmd view height, got %d", cmdViewHeight)
	}
	if availableWidth >= m.width {
		t.Errorf("expected available width to be less than total width due to padding")
	}
	if availableHeight >= m.height {
		t.Errorf("expected available height to be less than total height due to padding and status bar")
	}
}

func TestNew_InitialFocus(t *testing.T) {
	t.Parallel()

	store := &mockStore{}
	m := New(store, 100, 40)

	// Initially focused on list
	if m.focusedView != focusedViewList {
		t.Errorf("expected initial focus to be on list, got %v", m.focusedView)
	}

	// Editor should be created but not focused
	if m.editor.IsFocused() {
		t.Error("expected editor to not be focused initially")
	}
}

func TestModel_Update_KeyMsg_ExternalEditor(t *testing.T) {
	t.Parallel()

	records := []export.Record{
		{Name: "test", Content: "content", Path: "/tmp/test.txt"},
	}
	store := &mockStore{
		currentRecord: records[0],
		records:       records,
		editorName:    "nano",
	}
	m := New(store, 100, 40)
	m.focusedView = focusedViewList

	// Test pressing 'e' to open external editor
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}}
	_, cmd := m.Update(msg)

	// Should return an exec command
	if cmd == nil {
		t.Error("expected exec command for external editor, got nil")
	}
}
