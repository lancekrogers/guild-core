package code

// ParserRegistry holds all registered parsers
type ParserRegistry struct {
	parsers map[Language]Parser
}

// NewParserRegistry creates a new parser registry
func NewParserRegistry() *ParserRegistry {
	return &ParserRegistry{
		parsers: make(map[Language]Parser),
	}
}

// Register adds a parser to the registry
func (r *ParserRegistry) Register(lang Language, parser Parser) {
	r.parsers[lang] = parser
}

// Get retrieves a parser for a language
func (r *ParserRegistry) Get(lang Language) (Parser, bool) {
	parser, ok := r.parsers[lang]
	return parser, ok
}

// GetAll returns all registered parsers
func (r *ParserRegistry) GetAll() map[Language]Parser {
	result := make(map[Language]Parser, len(r.parsers))
	for k, v := range r.parsers {
		result[k] = v
	}
	return result
}