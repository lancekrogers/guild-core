package chat

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guild-framework/guild-core/internal/ui/chat/commands"
	"github.com/guild-framework/guild-core/internal/ui/chat/common"
	"github.com/guild-framework/guild-core/internal/ui/chat/messages"
	"github.com/guild-framework/guild-core/internal/ui/chat/panes"
	"github.com/guild-framework/guild-core/pkg/config"
)

func newMinimalApp(t *testing.T) *App {
	cfg := &common.ChatConfig{ProjectRoot: ".", GuildConfig: &config.GuildConfig{}}
	app := &App{ctx: context.Background(), config: cfg}

	op, err := panes.NewOutputPane(80, 20, false)
	require.NoError(t, err)
	ip, err := panes.NewInputPane(80, 3, false)
	require.NoError(t, err)
	sp, err := panes.NewStatusPane(80, 1)
	require.NoError(t, err)

	app.outputPane = op
	app.inputPane = ip
	app.statusPane = sp
	app.commandHistory = commands.NewCommandHistory(10)
	app.commandProcessor = commands.NewCommandProcessor(app.ctx, cfg, app.commandHistory, nil, nil, nil, nil)
	return app
}

func TestApp_HandleCommandPalette(t *testing.T) {
	app := newMinimalApp(t)
	_, cmd := app.handleCommandPalette()
	if cmd != nil {
		cmd()
	}
	msgs := app.outputPane.GetMessages()
	require.NotEmpty(t, msgs)
	assert.Contains(t, msgs[len(msgs)-1].Content, "Available Commands")
}

func TestApp_HandleGlobalSearch(t *testing.T) {
	if _, err := exec.LookPath("ag"); err != nil {
		t.Skip("ag not installed")
	}
	dir := t.TempDir()
	data := []byte("hello world")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "test.txt"), data, 0o644))

	app := newMinimalApp(t)
	app.config.ProjectRoot = dir
	app.inputPane.SetValue("hello")
	_, c := app.handleGlobalSearch()
	require.NotNil(t, c)
	msg := c().(messages.SearchMsg)
	assert.Equal(t, "hello", msg.Pattern)
	assert.NotEmpty(t, app.outputPane.GetMessages())
}
