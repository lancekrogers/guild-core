package cli

type InitOptions struct {
	ProjectName     string
	TemplateName    string
	OutputDirectory string
	Variables       map[string]interface{}
	DryRun          bool
	Force           bool
	ConfigFile      string
	Provider        string
	Model           string
	Interactive     bool
	Verbose         bool
}
