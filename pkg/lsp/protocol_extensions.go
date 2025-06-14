// +build example

// This file shows the additional protocol types that would need to be added
// to protocol.go to support the new LSP operations

package lsp

// WorkspaceSymbolParams represents parameters for workspace/symbol request
type WorkspaceSymbolParams struct {
	Query string `json:"query"`
}

// SymbolInformation represents information about a symbol
type SymbolInformation struct {
	Name          string   `json:"name"`
	Kind          SymbolKind `json:"kind"`
	Deprecated    bool     `json:"deprecated,omitempty"`
	Location      Location `json:"location"`
	ContainerName string   `json:"containerName,omitempty"`
}

// CodeActionContext contains additional information about the context in which a code action is run
type CodeActionContext struct {
	Diagnostics []Diagnostic `json:"diagnostics"`
	Only        []string     `json:"only,omitempty"`
}

// Diagnostic represents a diagnostic, such as a compiler error or warning
type Diagnostic struct {
	Range            Range              `json:"range"`
	Severity         DiagnosticSeverity `json:"severity,omitempty"`
	Code             interface{}        `json:"code,omitempty"`
	Source           string             `json:"source,omitempty"`
	Message          string             `json:"message"`
	RelatedInfo      []DiagnosticRelatedInformation `json:"relatedInformation,omitempty"`
}

// DiagnosticSeverity represents the severity of a diagnostic
type DiagnosticSeverity int

const (
	DiagnosticSeverityError       DiagnosticSeverity = 1
	DiagnosticSeverityWarning     DiagnosticSeverity = 2
	DiagnosticSeverityInformation DiagnosticSeverity = 3
	DiagnosticSeverityHint        DiagnosticSeverity = 4
)

// DiagnosticRelatedInformation represents related diagnostic information
type DiagnosticRelatedInformation struct {
	Location Location `json:"location"`
	Message  string   `json:"message"`
}

// CodeAction represents a code action
type CodeAction struct {
	Title       string         `json:"title"`
	Kind        string         `json:"kind,omitempty"`
	Diagnostics []Diagnostic   `json:"diagnostics,omitempty"`
	IsPreferred bool           `json:"isPreferred,omitempty"`
	Edit        *WorkspaceEdit `json:"edit,omitempty"`
	Command     *Command       `json:"command,omitempty"`
}

// Command represents a command to be executed
type Command struct {
	Title     string        `json:"title"`
	Command   string        `json:"command"`
	Arguments []interface{} `json:"arguments,omitempty"`
}

// WorkspaceEdit represents changes to many resources managed in the workspace
type WorkspaceEdit struct {
	Changes         map[string][]TextEdit `json:"changes,omitempty"`
	DocumentChanges []DocumentChange      `json:"documentChanges,omitempty"`
}

// DocumentChange represents a change to a document
type DocumentChange struct {
	TextDocument VersionedTextDocumentIdentifier `json:"textDocument"`
	Edits        []TextEdit                      `json:"edits"`
}

// PrepareRenameParams represents parameters for textDocument/prepareRename
type PrepareRenameParams struct {
	TextDocumentPositionParams
}

// RenameParams represents parameters for textDocument/rename
type RenameParams struct {
	TextDocumentPositionParams
	NewName string `json:"newName"`
}

// DocumentFormattingParams represents parameters for textDocument/formatting
type DocumentFormattingParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Options      FormattingOptions      `json:"options"`
}

// FormattingOptions represents formatting options
type FormattingOptions struct {
	TabSize                int    `json:"tabSize"`
	InsertSpaces           bool   `json:"insertSpaces"`
	TrimTrailingWhitespace bool   `json:"trimTrailingWhitespace,omitempty"`
	InsertFinalNewline     bool   `json:"insertFinalNewline,omitempty"`
	TrimFinalNewlines      bool   `json:"trimFinalNewlines,omitempty"`
}

// Common SymbolKind values
const (
	SymbolKindFile          SymbolKind = 1
	SymbolKindModule        SymbolKind = 2
	SymbolKindNamespace     SymbolKind = 3
	SymbolKindPackage       SymbolKind = 4
	SymbolKindClass         SymbolKind = 5
	SymbolKindMethod        SymbolKind = 6
	SymbolKindProperty      SymbolKind = 7
	SymbolKindField         SymbolKind = 8
	SymbolKindConstructor   SymbolKind = 9
	SymbolKindEnum          SymbolKind = 10
	SymbolKindInterface     SymbolKind = 11
	SymbolKindFunction      SymbolKind = 12
	SymbolKindVariable      SymbolKind = 13
	SymbolKindConstant      SymbolKind = 14
	SymbolKindString        SymbolKind = 15
	SymbolKindNumber        SymbolKind = 16
	SymbolKindBoolean       SymbolKind = 17
	SymbolKindArray         SymbolKind = 18
	SymbolKindObject        SymbolKind = 19
	SymbolKindKey           SymbolKind = 20
	SymbolKindNull          SymbolKind = 21
	SymbolKindEnumMember    SymbolKind = 22
	SymbolKindStruct        SymbolKind = 23
	SymbolKindEvent         SymbolKind = 24
	SymbolKindOperator      SymbolKind = 25
	SymbolKindTypeParameter SymbolKind = 26
)