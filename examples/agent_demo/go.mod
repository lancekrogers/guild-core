module agent-demo

go 1.24.2

replace github.com/guild-ventures/guild-core => ../..

require github.com/guild-ventures/guild-core v0.0.0-00010101000000-000000000000

require (
	github.com/google/uuid v1.6.0 // indirect
	github.com/lancekrogers/claude-code-go v0.1.0 // indirect
	github.com/philippgille/chromem-go v0.7.0 // indirect
	github.com/sashabaranov/go-openai v1.39.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
