package claudecode

// Minimal stub to avoid external dependency in default builds.
// The real implementation is behind the 'claudecode' build tag in client.go.

type Client struct{}

func NewClient(binPath, model string) *Client { return &Client{} }

