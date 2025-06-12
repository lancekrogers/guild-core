package lsp

import (
	"encoding/json"
)

// RequestID represents the ID of a JSON-RPC request
type RequestID struct {
	Number int64
	String string
}

// MarshalJSON implements json.Marshaler
func (id RequestID) MarshalJSON() ([]byte, error) {
	if id.String != "" {
		return json.Marshal(id.String)
	}
	return json.Marshal(id.Number)
}

// UnmarshalJSON implements json.Unmarshaler
func (id *RequestID) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &id.Number); err == nil {
		return nil
	}
	return json.Unmarshal(data, &id.String)
}

// Request represents a JSON-RPC request
type Request struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      *RequestID  `json:"id,omitempty"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// Response represents a JSON-RPC response
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *RequestID      `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *ResponseError  `json:"error,omitempty"`
}

// ResponseError represents a JSON-RPC error
type ResponseError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// InitializeParams represents the parameters for the initialize request
type InitializeParams struct {
	ProcessID             *int64             `json:"processId"`
	ClientInfo            *ClientInfo        `json:"clientInfo,omitempty"`
	RootURI               string             `json:"rootUri,omitempty"`
	InitializationOptions interface{}        `json:"initializationOptions,omitempty"`
	Capabilities          ClientCapabilities `json:"capabilities"`
	Trace                 string             `json:"trace,omitempty"`
	WorkspaceFolders      []WorkspaceFolder  `json:"workspaceFolders,omitempty"`
}

// ClientInfo represents information about the client
type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

// ClientCapabilities represents the capabilities of the client
type ClientCapabilities struct {
	Workspace    *WorkspaceClientCapabilities    `json:"workspace,omitempty"`
	TextDocument *TextDocumentClientCapabilities `json:"textDocument,omitempty"`
	Experimental interface{}                     `json:"experimental,omitempty"`
}

// WorkspaceClientCapabilities represents workspace client capabilities
type WorkspaceClientCapabilities struct {
	ApplyEdit              bool        `json:"applyEdit,omitempty"`
	WorkspaceEdit          interface{} `json:"workspaceEdit,omitempty"`
	DidChangeConfiguration interface{} `json:"didChangeConfiguration,omitempty"`
	DidChangeWatchedFiles  interface{} `json:"didChangeWatchedFiles,omitempty"`
	Symbol                 interface{} `json:"symbol,omitempty"`
	ExecuteCommand         interface{} `json:"executeCommand,omitempty"`
}

// TextDocumentClientCapabilities represents text document client capabilities
type TextDocumentClientCapabilities struct {
	Synchronization    interface{}                   `json:"synchronization,omitempty"`
	Completion         *CompletionClientCapabilities `json:"completion,omitempty"`
	Hover              interface{}                   `json:"hover,omitempty"`
	SignatureHelp      interface{}                   `json:"signatureHelp,omitempty"`
	Declaration        interface{}                   `json:"declaration,omitempty"`
	Definition         interface{}                   `json:"definition,omitempty"`
	TypeDefinition     interface{}                   `json:"typeDefinition,omitempty"`
	Implementation     interface{}                   `json:"implementation,omitempty"`
	References         interface{}                   `json:"references,omitempty"`
	DocumentHighlight  interface{}                   `json:"documentHighlight,omitempty"`
	DocumentSymbol     interface{}                   `json:"documentSymbol,omitempty"`
	CodeAction         interface{}                   `json:"codeAction,omitempty"`
	CodeLens           interface{}                   `json:"codeLens,omitempty"`
	DocumentLink       interface{}                   `json:"documentLink,omitempty"`
	ColorProvider      interface{}                   `json:"colorProvider,omitempty"`
	Formatting         interface{}                   `json:"formatting,omitempty"`
	RangeFormatting    interface{}                   `json:"rangeFormatting,omitempty"`
	OnTypeFormatting   interface{}                   `json:"onTypeFormatting,omitempty"`
	Rename             interface{}                   `json:"rename,omitempty"`
	PublishDiagnostics interface{}                   `json:"publishDiagnostics,omitempty"`
	FoldingRange       interface{}                   `json:"foldingRange,omitempty"`
}

// CompletionClientCapabilities represents completion client capabilities
type CompletionClientCapabilities struct {
	DynamicRegistration bool                        `json:"dynamicRegistration,omitempty"`
	CompletionItem      *CompletionItemCapabilities `json:"completionItem,omitempty"`
}

// CompletionItemCapabilities represents completion item capabilities
type CompletionItemCapabilities struct {
	SnippetSupport          bool     `json:"snippetSupport,omitempty"`
	CommitCharactersSupport bool     `json:"commitCharactersSupport,omitempty"`
	DocumentationFormat     []string `json:"documentationFormat,omitempty"`
	DeprecatedSupport       bool     `json:"deprecatedSupport,omitempty"`
	PreselectSupport        bool     `json:"preselectSupport,omitempty"`
}

// WorkspaceFolder represents a workspace folder
type WorkspaceFolder struct {
	URI  string `json:"uri"`
	Name string `json:"name"`
}

// InitializeResult represents the result of the initialize request
type InitializeResult struct {
	Capabilities ServerCapabilities `json:"capabilities"`
	ServerInfo   *ServerInfo        `json:"serverInfo,omitempty"`
}

// ServerInfo represents information about the server
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

// ServerCapabilities represents the capabilities of the server
type ServerCapabilities struct {
	TextDocumentSync                 interface{}                  `json:"textDocumentSync,omitempty"`
	CompletionProvider               *CompletionOptions           `json:"completionProvider,omitempty"`
	HoverProvider                    interface{}                  `json:"hoverProvider,omitempty"`
	SignatureHelpProvider            interface{}                  `json:"signatureHelpProvider,omitempty"`
	DeclarationProvider              interface{}                  `json:"declarationProvider,omitempty"`
	DefinitionProvider               interface{}                  `json:"definitionProvider,omitempty"`
	TypeDefinitionProvider           interface{}                  `json:"typeDefinitionProvider,omitempty"`
	ImplementationProvider           interface{}                  `json:"implementationProvider,omitempty"`
	ReferencesProvider               interface{}                  `json:"referencesProvider,omitempty"`
	DocumentHighlightProvider        interface{}                  `json:"documentHighlightProvider,omitempty"`
	DocumentSymbolProvider           interface{}                  `json:"documentSymbolProvider,omitempty"`
	CodeActionProvider               interface{}                  `json:"codeActionProvider,omitempty"`
	CodeLensProvider                 interface{}                  `json:"codeLensProvider,omitempty"`
	DocumentLinkProvider             interface{}                  `json:"documentLinkProvider,omitempty"`
	ColorProvider                    interface{}                  `json:"colorProvider,omitempty"`
	DocumentFormattingProvider       interface{}                  `json:"documentFormattingProvider,omitempty"`
	DocumentRangeFormattingProvider  interface{}                  `json:"documentRangeFormattingProvider,omitempty"`
	DocumentOnTypeFormattingProvider interface{}                  `json:"documentOnTypeFormattingProvider,omitempty"`
	RenameProvider                   interface{}                  `json:"renameProvider,omitempty"`
	FoldingRangeProvider             interface{}                  `json:"foldingRangeProvider,omitempty"`
	ExecuteCommandProvider           interface{}                  `json:"executeCommandProvider,omitempty"`
	WorkspaceSymbolProvider          interface{}                  `json:"workspaceSymbolProvider,omitempty"`
	Workspace                        *WorkspaceServerCapabilities `json:"workspace,omitempty"`
	Experimental                     interface{}                  `json:"experimental,omitempty"`
}

// CompletionOptions represents completion options
type CompletionOptions struct {
	ResolveProvider   bool     `json:"resolveProvider,omitempty"`
	TriggerCharacters []string `json:"triggerCharacters,omitempty"`
}

// WorkspaceServerCapabilities represents workspace server capabilities
type WorkspaceServerCapabilities struct {
	WorkspaceFolders *WorkspaceFoldersServerCapabilities `json:"workspaceFolders,omitempty"`
}

// WorkspaceFoldersServerCapabilities represents workspace folders server capabilities
type WorkspaceFoldersServerCapabilities struct {
	Supported           bool        `json:"supported,omitempty"`
	ChangeNotifications interface{} `json:"changeNotifications,omitempty"`
}

// InitializedParams represents the parameters for the initialized notification
type InitializedParams struct{}

// TextDocumentIdentifier represents a text document
type TextDocumentIdentifier struct {
	URI string `json:"uri"`
}

// VersionedTextDocumentIdentifier represents a versioned text document
type VersionedTextDocumentIdentifier struct {
	TextDocumentIdentifier
	Version *int `json:"version"`
}

// Position represents a position in a text document
type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

// Range represents a range in a text document
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// Location represents a location in a text document
type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

// TextDocumentPositionParams represents parameters for text document position requests
type TextDocumentPositionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

// CompletionParams represents parameters for completion requests
type CompletionParams struct {
	TextDocumentPositionParams
	Context *CompletionContext `json:"context,omitempty"`
}

// CompletionContext represents the context of a completion request
type CompletionContext struct {
	TriggerKind      CompletionTriggerKind `json:"triggerKind"`
	TriggerCharacter string                `json:"triggerCharacter,omitempty"`
}

// CompletionTriggerKind represents how a completion was triggered
type CompletionTriggerKind int

const (
	// CompletionTriggerKindInvoked means completion was invoked manually
	CompletionTriggerKindInvoked CompletionTriggerKind = 1
	// CompletionTriggerKindTriggerCharacter means completion was triggered by a character
	CompletionTriggerKindTriggerCharacter CompletionTriggerKind = 2
	// CompletionTriggerKindTriggerForIncompleteCompletions means completion was re-triggered
	CompletionTriggerKindTriggerForIncompleteCompletions CompletionTriggerKind = 3
)

// CompletionList represents a list of completion items
type CompletionList struct {
	IsIncomplete bool             `json:"isIncomplete"`
	Items        []CompletionItem `json:"items"`
}

// CompletionItem represents a completion item
type CompletionItem struct {
	Label               string             `json:"label"`
	Kind                CompletionItemKind `json:"kind,omitempty"`
	Detail              string             `json:"detail,omitempty"`
	Documentation       interface{}        `json:"documentation,omitempty"`
	Deprecated          bool               `json:"deprecated,omitempty"`
	Preselect           bool               `json:"preselect,omitempty"`
	SortText            string             `json:"sortText,omitempty"`
	FilterText          string             `json:"filterText,omitempty"`
	InsertText          string             `json:"insertText,omitempty"`
	InsertTextFormat    InsertTextFormat   `json:"insertTextFormat,omitempty"`
	TextEdit            *TextEdit          `json:"textEdit,omitempty"`
	AdditionalTextEdits []TextEdit         `json:"additionalTextEdits,omitempty"`
	CommitCharacters    []string           `json:"commitCharacters,omitempty"`
	Command             *Command           `json:"command,omitempty"`
	Data                interface{}        `json:"data,omitempty"`
}

// CompletionItemKind represents the kind of a completion item
type CompletionItemKind int

const (
	CompletionItemKindText          CompletionItemKind = 1
	CompletionItemKindMethod        CompletionItemKind = 2
	CompletionItemKindFunction      CompletionItemKind = 3
	CompletionItemKindConstructor   CompletionItemKind = 4
	CompletionItemKindField         CompletionItemKind = 5
	CompletionItemKindVariable      CompletionItemKind = 6
	CompletionItemKindClass         CompletionItemKind = 7
	CompletionItemKindInterface     CompletionItemKind = 8
	CompletionItemKindModule        CompletionItemKind = 9
	CompletionItemKindProperty      CompletionItemKind = 10
	CompletionItemKindUnit          CompletionItemKind = 11
	CompletionItemKindValue         CompletionItemKind = 12
	CompletionItemKindEnum          CompletionItemKind = 13
	CompletionItemKindKeyword       CompletionItemKind = 14
	CompletionItemKindSnippet       CompletionItemKind = 15
	CompletionItemKindColor         CompletionItemKind = 16
	CompletionItemKindFile          CompletionItemKind = 17
	CompletionItemKindReference     CompletionItemKind = 18
	CompletionItemKindFolder        CompletionItemKind = 19
	CompletionItemKindEnumMember    CompletionItemKind = 20
	CompletionItemKindConstant      CompletionItemKind = 21
	CompletionItemKindStruct        CompletionItemKind = 22
	CompletionItemKindEvent         CompletionItemKind = 23
	CompletionItemKindOperator      CompletionItemKind = 24
	CompletionItemKindTypeParameter CompletionItemKind = 25
)

// InsertTextFormat represents how the insert text should be interpreted
type InsertTextFormat int

const (
	// InsertTextFormatPlainText means the text is plain text
	InsertTextFormatPlainText InsertTextFormat = 1
	// InsertTextFormatSnippet means the text is a snippet
	InsertTextFormatSnippet InsertTextFormat = 2
)

// TextEdit represents a text edit
type TextEdit struct {
	Range   Range  `json:"range"`
	NewText string `json:"newText"`
}

// Command represents a command
type Command struct {
	Title     string        `json:"title"`
	Command   string        `json:"command"`
	Arguments []interface{} `json:"arguments,omitempty"`
}

// Hover represents hover information
type Hover struct {
	Contents interface{} `json:"contents"`
	Range    *Range      `json:"range,omitempty"`
}

// MarkupContent represents markup content
type MarkupContent struct {
	Kind  MarkupKind `json:"kind"`
	Value string     `json:"value"`
}

// MarkupKind represents the kind of markup
type MarkupKind string

const (
	// MarkupKindPlainText represents plain text
	MarkupKindPlainText MarkupKind = "plaintext"
	// MarkupKindMarkdown represents markdown
	MarkupKindMarkdown MarkupKind = "markdown"
)

// DidOpenTextDocumentParams represents parameters for textDocument/didOpen
type DidOpenTextDocumentParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}

// TextDocumentItem represents a text document
type TextDocumentItem struct {
	URI        string `json:"uri"`
	LanguageID string `json:"languageId"`
	Version    int    `json:"version"`
	Text       string `json:"text"`
}

// DidChangeTextDocumentParams represents parameters for textDocument/didChange
type DidChangeTextDocumentParams struct {
	TextDocument   VersionedTextDocumentIdentifier  `json:"textDocument"`
	ContentChanges []TextDocumentContentChangeEvent `json:"contentChanges"`
}

// TextDocumentContentChangeEvent represents a change to a text document
type TextDocumentContentChangeEvent struct {
	Range       *Range `json:"range,omitempty"`
	RangeLength *int   `json:"rangeLength,omitempty"`
	Text        string `json:"text"`
}

// DidCloseTextDocumentParams represents parameters for textDocument/didClose
type DidCloseTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

// ReferenceParams represents parameters for textDocument/references
type ReferenceParams struct {
	TextDocumentPositionParams
	Context ReferenceContext `json:"context"`
}

// ReferenceContext represents the context for finding references
type ReferenceContext struct {
	IncludeDeclaration bool `json:"includeDeclaration"`
}

// Type aliases for consistency
type HoverParams = TextDocumentPositionParams
type DefinitionParams = TextDocumentPositionParams

// DocumentSymbolParams represents parameters for textDocument/documentSymbol
type DocumentSymbolParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

// CodeActionParams represents parameters for textDocument/codeAction
type CodeActionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Range        Range                  `json:"range"`
	Context      CodeActionContext      `json:"context"`
}

// CodeActionContext represents the context for code actions
type CodeActionContext struct {
	Diagnostics []Diagnostic `json:"diagnostics"`
	Only        []string     `json:"only,omitempty"`
}

// ExecuteCommandParams represents parameters for workspace/executeCommand
type ExecuteCommandParams struct {
	Command   string        `json:"command"`
	Arguments []interface{} `json:"arguments,omitempty"`
}

// WorkspaceSymbolParams represents parameters for workspace/symbol
type WorkspaceSymbolParams struct {
	Query string `json:"query"`
}

// SignatureHelpParams represents parameters for textDocument/signatureHelp
type SignatureHelpParams struct {
	TextDocumentPositionParams
	Context *SignatureHelpContext `json:"context,omitempty"`
}

// SignatureHelpContext represents the context for signature help
type SignatureHelpContext struct {
	TriggerKind         SignatureHelpTriggerKind `json:"triggerKind"`
	TriggerCharacter    string                   `json:"triggerCharacter,omitempty"`
	IsRetrigger         bool                     `json:"isRetrigger"`
	ActiveSignatureHelp *SignatureHelp           `json:"activeSignatureHelp,omitempty"`
}

// SignatureHelpTriggerKind represents how signature help was triggered
type SignatureHelpTriggerKind int

const (
	// SignatureHelpTriggerKindInvoked means signature help was invoked manually
	SignatureHelpTriggerKindInvoked SignatureHelpTriggerKind = 1
	// SignatureHelpTriggerKindTriggerCharacter means signature help was triggered by a character
	SignatureHelpTriggerKindTriggerCharacter SignatureHelpTriggerKind = 2
	// SignatureHelpTriggerKindContentChange means signature help was triggered by content change
	SignatureHelpTriggerKindContentChange SignatureHelpTriggerKind = 3
)

// CodeLensParams represents parameters for textDocument/codeLens
type CodeLensParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

// DocumentHighlightParams represents parameters for textDocument/documentHighlight
type DocumentHighlightParams = TextDocumentPositionParams

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
	TabSize                int  `json:"tabSize"`
	InsertSpaces           bool `json:"insertSpaces"`
	TrimTrailingWhitespace bool `json:"trimTrailingWhitespace,omitempty"`
	InsertFinalNewline     bool `json:"insertFinalNewline,omitempty"`
	TrimFinalNewlines      bool `json:"trimFinalNewlines,omitempty"`
}

// Additional types needed for the interface methods

// DocumentSymbol represents a symbol in a document
type DocumentSymbol struct {
	Name           string           `json:"name"`
	Detail         string           `json:"detail,omitempty"`
	Kind           SymbolKind       `json:"kind"`
	Deprecated     bool             `json:"deprecated,omitempty"`
	Range          Range            `json:"range"`
	SelectionRange Range            `json:"selectionRange"`
	Children       []DocumentSymbol `json:"children,omitempty"`
}

// SymbolKind represents the kind of a symbol
type SymbolKind int

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

// CodeAction represents a code action
type CodeAction struct {
	Title       string         `json:"title"`
	Kind        string         `json:"kind,omitempty"`
	Diagnostics []Diagnostic   `json:"diagnostics,omitempty"`
	IsPreferred bool           `json:"isPreferred,omitempty"`
	Edit        *WorkspaceEdit `json:"edit,omitempty"`
	Command     *Command       `json:"command,omitempty"`
}

// WorkspaceEdit represents changes to many resources
type WorkspaceEdit struct {
	Changes map[string][]TextEdit `json:"changes,omitempty"`
}

// Diagnostic represents a diagnostic
type Diagnostic struct {
	Range              Range                          `json:"range"`
	Severity           DiagnosticSeverity             `json:"severity,omitempty"`
	Code               interface{}                    `json:"code,omitempty"`
	Source             string                         `json:"source,omitempty"`
	Message            string                         `json:"message"`
	Tags               []DiagnosticTag                `json:"tags,omitempty"`
	RelatedInformation []DiagnosticRelatedInformation `json:"relatedInformation,omitempty"`
}

// DiagnosticSeverity represents the severity of a diagnostic
type DiagnosticSeverity int

const (
	DiagnosticSeverityError       DiagnosticSeverity = 1
	DiagnosticSeverityWarning     DiagnosticSeverity = 2
	DiagnosticSeverityInformation DiagnosticSeverity = 3
	DiagnosticSeverityHint        DiagnosticSeverity = 4
)

// DiagnosticTag represents a diagnostic tag
type DiagnosticTag int

const (
	DiagnosticTagUnnecessary DiagnosticTag = 1
	DiagnosticTagDeprecated  DiagnosticTag = 2
)

// DiagnosticRelatedInformation represents related information for a diagnostic
type DiagnosticRelatedInformation struct {
	Location Location `json:"location"`
	Message  string   `json:"message"`
}

// SymbolInformation represents information about a symbol
type SymbolInformation struct {
	Name          string     `json:"name"`
	Kind          SymbolKind `json:"kind"`
	Deprecated    bool       `json:"deprecated,omitempty"`
	Location      Location   `json:"location"`
	ContainerName string     `json:"containerName,omitempty"`
}

// SignatureHelp represents signature help
type SignatureHelp struct {
	Signatures      []SignatureInformation `json:"signatures"`
	ActiveSignature int                    `json:"activeSignature,omitempty"`
	ActiveParameter int                    `json:"activeParameter,omitempty"`
}

// SignatureInformation represents information about a signature
type SignatureInformation struct {
	Label           string                 `json:"label"`
	Documentation   interface{}            `json:"documentation,omitempty"`
	Parameters      []ParameterInformation `json:"parameters,omitempty"`
	ActiveParameter int                    `json:"activeParameter,omitempty"`
}

// ParameterInformation represents information about a parameter
type ParameterInformation struct {
	Label         interface{} `json:"label"`
	Documentation interface{} `json:"documentation,omitempty"`
}

// CodeLens represents a code lens
type CodeLens struct {
	Range   Range       `json:"range"`
	Command *Command    `json:"command,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// DocumentHighlight represents a document highlight
type DocumentHighlight struct {
	Range Range                 `json:"range"`
	Kind  DocumentHighlightKind `json:"kind,omitempty"`
}

// DocumentHighlightKind represents the kind of a document highlight
type DocumentHighlightKind int

const (
	DocumentHighlightKindText  DocumentHighlightKind = 1
	DocumentHighlightKindRead  DocumentHighlightKind = 2
	DocumentHighlightKindWrite DocumentHighlightKind = 3
)
