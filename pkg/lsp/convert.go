package lsp

import (
	"encoding/json"

	"github.com/ionut-t/goeditor/core"
)

// completionItemKind mirrors LSP CompletionItemKind values.
// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#completionItemKind
type completionItemKind int

const (
	kindMethod      completionItemKind = 2
	kindFunction    completionItemKind = 3
	kindConstructor completionItemKind = 4
	kindField       completionItemKind = 5
	kindVariable    completionItemKind = 6
	kindClass       completionItemKind = 7
	kindInterface   completionItemKind = 8
	kindModule      completionItemKind = 9
	kindProperty    completionItemKind = 10
	kindValue       completionItemKind = 12
	kindEnum        completionItemKind = 13
	kindKeyword     completionItemKind = 14
	kindSnippet     completionItemKind = 15
	kindEnumMember  completionItemKind = 20
	kindConstant    completionItemKind = 21
	kindStruct      completionItemKind = 22
)

// lspCompletionItem represents a single item from an LSP completion response.
type lspCompletionItem struct {
	Label            string          `json:"label"`
	Kind             completionItemKind `json:"kind,omitempty"`
	Detail           string          `json:"detail,omitempty"`
	Documentation    json.RawMessage `json:"documentation,omitempty"`
	SortText         string          `json:"sortText,omitempty"`
	FilterText       string          `json:"filterText,omitempty"`
	InsertText       string          `json:"insertText,omitempty"`
}

// kindToType maps an LSP CompletionItemKind to a human-readable type string
// displayed in the goeditor completion menu.
func kindToType(k completionItemKind) string {
	switch k {
	case kindKeyword:
		return "keyword"
	case kindFunction, kindMethod, kindConstructor:
		return "function"
	case kindField, kindProperty, kindVariable:
		return "column"
	case kindClass, kindStruct, kindModule, kindInterface:
		return "table"
	case kindEnum, kindEnumMember:
		return "enum"
	case kindConstant, kindValue:
		return "constant"
	case kindSnippet:
		return "snippet"
	default:
		return "text"
	}
}

// toCompletion converts an LSP CompletionItem to a goeditor core.Completion.
func toCompletion(item lspCompletionItem) core.Completion {
	insertText := item.InsertText
	if insertText == "" {
		insertText = item.Label
	}

	description := item.Detail
	if description == "" && len(item.Documentation) > 0 {
		// Documentation can be a string or a MarkupContent object.
		var s string
		if err := json.Unmarshal(item.Documentation, &s); err == nil {
			description = s
		} else {
			var mc struct {
				Value string `json:"value"`
			}
			if err := json.Unmarshal(item.Documentation, &mc); err == nil {
				description = mc.Value
			}
		}
	}

	return core.Completion{
		Text:        insertText,
		Label:       item.Label,
		Description: description,
		Type:        kindToType(item.Kind),
	}
}
