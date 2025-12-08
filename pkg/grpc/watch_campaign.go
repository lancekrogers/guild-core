// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package grpc

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"unsafe"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/guild-framework/guild-core/pkg/gerror"
	pb "github.com/guild-framework/guild-core/pkg/grpc/pb/guild/v1"
)

// Screen represents the terminal screen state
type Screen struct {
	prev   [][]byte
	width  int
	height int
}

// NewScreen creates a new screen
func NewScreen() *Screen {
	w, h := getTermSize()
	return &Screen{
		prev:   make([][]byte, h),
		width:  w,
		height: h,
	}
}

// Clear clears the screen
func (s *Screen) Clear() {
	// Clear screen and move cursor to top
	fmt.Print("\033[2J\033[H")
}

// HideCursor hides the cursor
func (s *Screen) HideCursor() {
	fmt.Print("\033[?25l")
}

// ShowCursor shows the cursor
func (s *Screen) ShowCursor() {
	fmt.Print("\033[?25h")
}

// Render renders a frame using shadow-diff algorithm
func (s *Screen) Render(frame []byte) {
	lines := bytes.Split(frame, []byte{'\n'})

	// Ensure we have enough lines in prev
	if len(lines) > len(s.prev) {
		newPrev := make([][]byte, len(lines))
		copy(newPrev, s.prev)
		s.prev = newPrev
	}

	for row, line := range lines {
		if row >= s.height {
			break // Don't render beyond screen height
		}

		if !bytes.Equal(line, s.prev[row]) {
			// Move cursor to row and render line
			fmt.Printf("\033[%d;1H%s\033[K", row+1, line)
			s.prev[row] = append([]byte(nil), line...)
		}
	}

	// Clear any remaining lines from previous frame
	for row := len(lines); row < len(s.prev) && row < s.height; row++ {
		if len(s.prev[row]) > 0 {
			fmt.Printf("\033[%d;1H\033[K", row+1)
			s.prev[row] = nil
		}
	}
}

// Resize handles terminal resize
func (s *Screen) Resize() {
	w, h := getTermSize()
	s.width = w
	s.height = h

	// Reallocate prev buffer if needed
	if h > len(s.prev) {
		newPrev := make([][]byte, h)
		copy(newPrev, s.prev)
		s.prev = newPrev
	}

	// Force full redraw
	s.Clear()
}

// WatchCampaignClient handles the client side of campaign watching
type WatchCampaignClient struct {
	client pb.GuildClient
	screen *Screen
}

// NewWatchCampaignClient creates a new watch campaign client
func NewWatchCampaignClient(address string) (*WatchCampaignClient, error) {
	// Create gRPC connection
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeConnection, "failed to connect").
			WithComponent("grpc").
			WithOperation("NewWatchCampaignClient").
			WithDetails("address", address)
	}

	client := pb.NewGuildClient(conn)
	screen := NewScreen()

	return &WatchCampaignClient{
		client: client,
		screen: screen,
	}, nil
}

// Watch starts watching a campaign
func (c *WatchCampaignClient) Watch(ctx context.Context, campaignID string, options WatchOptions) error {
	// Set up terminal
	c.screen.Clear()
	c.screen.HideCursor()
	defer c.screen.ShowCursor()

	// Set up signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGWINCH)
	defer signal.Stop(sigCh)

	// Create watch request
	req := &pb.WatchRequest{
		CampaignId:      campaignID,
		IncludeAgents:   options.IncludeAgents,
		IncludeKanban:   options.IncludeKanban,
		IncludeProgress: options.IncludeProgress,
	}

	// Start streaming
	stream, err := c.client.WatchCampaign(ctx, req)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to start watch").
			WithComponent("grpc").
			WithOperation("Watch").
			WithDetails("campaign_id", campaignID).
			FromContext(ctx)
	}

	// Main watch loop
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case sig := <-sigCh:
			switch sig {
			case syscall.SIGINT, syscall.SIGTERM:
				return nil
			case syscall.SIGWINCH:
				c.screen.Resize()
			}

		default:
			// Receive update from stream
			update, err := stream.Recv()
			if err == io.EOF {
				return nil
			}
			if err != nil {
				return gerror.Wrap(err, gerror.ErrCodeInternal, "stream error").
					WithComponent("grpc").
					WithOperation("Watch").
					WithDetails("campaign_id", campaignID).
					FromContext(ctx)
			}

			// Render frame
			c.screen.Render([]byte(update.Frame))

			// Optional: display metadata
			if options.ShowMetadata && update.Metadata != nil {
				c.displayMetadata(update.Metadata)
			}
		}
	}
}

// displayMetadata shows metadata in a status line
func (c *WatchCampaignClient) displayMetadata(meta *pb.BoardMetadata) {
	// Move to bottom of screen
	fmt.Printf("\033[%d;1H", c.screen.height)

	// Display metadata
	status := fmt.Sprintf("Size: %dx%d | Agents: %d | Tasks: %d/%d | FPS: %.1f",
		meta.Width, meta.Height,
		meta.ActiveAgents,
		meta.CompletedTasks, meta.TotalTasks,
		meta.Fps)

	fmt.Printf("\033[K%s", status) // Clear to end of line and print status
}

// WatchOptions contains options for watching campaigns
type WatchOptions struct {
	IncludeAgents   bool
	IncludeKanban   bool
	IncludeProgress bool
	ShowMetadata    bool
	FPSCap          int
}

// DefaultWatchOptions returns default watch options
func DefaultWatchOptions() WatchOptions {
	return WatchOptions{
		IncludeAgents:   true,
		IncludeKanban:   true,
		IncludeProgress: true,
		ShowMetadata:    false,
		FPSCap:          60,
	}
}

// getTermSize gets terminal dimensions (Unix/Linux/macOS)
func getTermSize() (width, height int) {
	var ws struct {
		Row    uint16
		Col    uint16
		Xpixel uint16
		Ypixel uint16
	}

	_, _, _ = syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(syscall.Stdout),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(&ws)),
	)

	if ws.Col == 0 || ws.Row == 0 {
		// Fallback to reasonable defaults
		return 80, 24
	}

	return int(ws.Col), int(ws.Row)
}

// WatchCampaign is a convenience function to watch a campaign
func WatchCampaign(ctx context.Context, address, campaignID string, options WatchOptions) error {
	client, err := NewWatchCampaignClient(address)
	if err != nil {
		return err
	}

	return client.Watch(ctx, campaignID, options)
}
