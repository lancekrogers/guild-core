package features

import (
	"context"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/guild-framework/guild-core/pkg/gerror"
	"golang.org/x/term"
)

// Size represents terminal dimensions
type Size struct {
	Width  int
	Height int
}

// SizeDetector detects and monitors terminal size
type SizeDetector struct {
	mu           sync.RWMutex
	currentSize  Size
	callbacks    []SizeCallback
	monitoring   bool
	signalChan   chan os.Signal
	stopChan     chan struct{}
	lastDetected time.Time
}

// SizeCallback is called when terminal size changes
type SizeCallback func(size Size)

// NewSizeDetector creates a new size detector
func NewSizeDetector() *SizeDetector {
	return &SizeDetector{
		callbacks:  make([]SizeCallback, 0),
		signalChan: make(chan os.Signal, 1),
		stopChan:   make(chan struct{}),
	}
}

// Detect gets the current terminal size
func (sd *SizeDetector) Detect(ctx context.Context) (Size, error) {
	if err := ctx.Err(); err != nil {
		return Size{}, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled during size detection")
	}

	// Try to get size from golang.org/x/term
	if width, height, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
		size := Size{Width: width, Height: height}
		sd.updateSize(size)
		return size, nil
	}

	// Fallback to environment variables
	if size, ok := sd.getSizeFromEnv(); ok {
		sd.updateSize(size)
		return size, nil
	}

	// Try ANSI escape sequence method
	if size, err := sd.getSizeFromANSI(ctx); err == nil {
		sd.updateSize(size)
		return size, nil
	}

	// Return cached size if available
	sd.mu.RLock()
	if sd.currentSize.Width > 0 && sd.currentSize.Height > 0 {
		size := sd.currentSize
		sd.mu.RUnlock()
		return size, nil
	}
	sd.mu.RUnlock()

	// Default fallback size
	defaultSize := Size{Width: 80, Height: 24}
	sd.updateSize(defaultSize)
	return defaultSize, nil
}

// getSizeFromEnv tries to get size from environment variables
func (sd *SizeDetector) getSizeFromEnv() (Size, bool) {
	widthStr := os.Getenv("COLUMNS")
	heightStr := os.Getenv("LINES")

	if widthStr == "" || heightStr == "" {
		return Size{}, false
	}

	width, err := strconv.Atoi(widthStr)
	if err != nil {
		return Size{}, false
	}

	height, err := strconv.Atoi(heightStr)
	if err != nil {
		return Size{}, false
	}

	if width <= 0 || height <= 0 {
		return Size{}, false
	}

	return Size{Width: width, Height: height}, true
}

// getSizeFromANSI queries terminal size using ANSI escape sequences
func (sd *SizeDetector) getSizeFromANSI(ctx context.Context) (Size, error) {
	// This is a simplified implementation
	// In a real implementation, you'd:
	// 1. Send cursor position query: \x1b[6n
	// 2. Move cursor to bottom right: \x1b[999;999H
	// 3. Query position again: \x1b[6n
	// 4. Parse the response to get terminal size
	// 5. Restore original cursor position

	// For now, return an error to indicate this method failed
	return Size{}, gerror.New(gerror.ErrCodeNotImplemented, "ANSI size detection not implemented", nil)
}

// updateSize updates the current size and notifies callbacks
func (sd *SizeDetector) updateSize(size Size) {
	sd.mu.Lock()
	oldSize := sd.currentSize
	sd.currentSize = size
	sd.lastDetected = time.Now()
	callbacks := make([]SizeCallback, len(sd.callbacks))
	copy(callbacks, sd.callbacks)
	sd.mu.Unlock()

	// Only notify if size actually changed
	if oldSize.Width != size.Width || oldSize.Height != size.Height {
		for _, callback := range callbacks {
			callback(size)
		}
	}
}

// GetCurrent returns the current cached size
func (sd *SizeDetector) GetCurrent() Size {
	sd.mu.RLock()
	defer sd.mu.RUnlock()
	return sd.currentSize
}

// AddCallback adds a size change callback
func (sd *SizeDetector) AddCallback(callback SizeCallback) {
	sd.mu.Lock()
	defer sd.mu.Unlock()
	sd.callbacks = append(sd.callbacks, callback)
}

// StartMonitoring starts monitoring for size changes
func (sd *SizeDetector) StartMonitoring(ctx context.Context) error {
	sd.mu.Lock()
	if sd.monitoring {
		sd.mu.Unlock()
		return gerror.New(gerror.ErrCodeConflict, "size monitoring already started", nil)
	}
	sd.monitoring = true
	sd.mu.Unlock()

	// Set up signal handler for SIGWINCH (window size change)
	signal.Notify(sd.signalChan, syscall.SIGWINCH)

	go sd.monitorLoop(ctx)

	return nil
}

// StopMonitoring stops monitoring for size changes
func (sd *SizeDetector) StopMonitoring() {
	sd.mu.Lock()
	if !sd.monitoring {
		sd.mu.Unlock()
		return
	}
	sd.monitoring = false
	sd.mu.Unlock()

	signal.Stop(sd.signalChan)
	close(sd.stopChan)
}

// monitorLoop runs the monitoring loop
func (sd *SizeDetector) monitorLoop(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second) // Fallback polling
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-sd.stopChan:
			return
		case <-sd.signalChan:
			// SIGWINCH received, update size
			if size, err := sd.Detect(ctx); err == nil {
				sd.updateSize(size)
			}
		case <-ticker.C:
			// Periodic fallback check
			sd.mu.RLock()
			lastCheck := sd.lastDetected
			sd.mu.RUnlock()

			// Only check if we haven't detected recently
			if time.Since(lastCheck) > 5*time.Second {
				if size, err := sd.Detect(ctx); err == nil {
					sd.updateSize(size)
				}
			}
		}
	}
}

// IsValidSize checks if a size is valid
func (s Size) IsValidSize() bool {
	return s.Width > 0 && s.Height > 0
}

// AspectRatio returns the aspect ratio (width/height)
func (s Size) AspectRatio() float64 {
	if s.Height == 0 {
		return 0
	}
	return float64(s.Width) / float64(s.Height)
}

// Area returns the total area (width * height)
func (s Size) Area() int {
	return s.Width * s.Height
}

// String returns a string representation of the size
func (s Size) String() string {
	return strconv.Itoa(s.Width) + "x" + strconv.Itoa(s.Height)
}

// FitsInside checks if this size fits inside another size
func (s Size) FitsInside(other Size) bool {
	return s.Width <= other.Width && s.Height <= other.Height
}

// Constrain constrains this size to fit within bounds
func (s Size) Constrain(maxSize Size) Size {
	width := s.Width
	height := s.Height

	if width > maxSize.Width {
		width = maxSize.Width
	}
	if height > maxSize.Height {
		height = maxSize.Height
	}

	return Size{Width: width, Height: height}
}

// Scale scales the size by a factor
func (s Size) Scale(factor float64) Size {
	return Size{
		Width:  int(float64(s.Width) * factor),
		Height: int(float64(s.Height) * factor),
	}
}

// Add adds another size to this size
func (s Size) Add(other Size) Size {
	return Size{
		Width:  s.Width + other.Width,
		Height: s.Height + other.Height,
	}
}

// Subtract subtracts another size from this size
func (s Size) Subtract(other Size) Size {
	return Size{
		Width:  s.Width - other.Width,
		Height: s.Height - other.Height,
	}
}

// Min returns the minimum of each dimension
func (s Size) Min(other Size) Size {
	width := s.Width
	if other.Width < width {
		width = other.Width
	}

	height := s.Height
	if other.Height < height {
		height = other.Height
	}

	return Size{Width: width, Height: height}
}

// Max returns the maximum of each dimension
func (s Size) Max(other Size) Size {
	width := s.Width
	if other.Width > width {
		width = other.Width
	}

	height := s.Height
	if other.Height > height {
		height = other.Height
	}

	return Size{Width: width, Height: height}
}

// SizeClass represents different size categories
type SizeClass int

const (
	SizeClassTiny   SizeClass = iota // < 40x10
	SizeClassSmall                   // < 80x24
	SizeClassMedium                  // < 120x40
	SizeClassLarge                   // < 160x60
	SizeClassHuge                    // >= 160x60
)

// GetSizeClass returns the size class for this size
func (s Size) GetSizeClass() SizeClass {
	if s.Width < 40 || s.Height < 10 {
		return SizeClassTiny
	}
	if s.Width < 80 || s.Height < 24 {
		return SizeClassSmall
	}
	if s.Width < 120 || s.Height < 40 {
		return SizeClassMedium
	}
	if s.Width < 160 || s.Height < 60 {
		return SizeClassLarge
	}
	return SizeClassHuge
}

// String returns a string representation of the size class
func (sc SizeClass) String() string {
	switch sc {
	case SizeClassTiny:
		return "tiny"
	case SizeClassSmall:
		return "small"
	case SizeClassMedium:
		return "medium"
	case SizeClassLarge:
		return "large"
	case SizeClassHuge:
		return "huge"
	default:
		return "unknown"
	}
}

// ResponsiveLayout provides layout recommendations based on size
type ResponsiveLayout struct {
	Columns     int
	MaxWidth    int
	ShowSidebar bool
	ShowDetails bool
}

// GetLayout returns a recommended layout for the given size
func GetLayout(size Size) ResponsiveLayout {
	class := size.GetSizeClass()

	switch class {
	case SizeClassTiny:
		return ResponsiveLayout{
			Columns:     1,
			MaxWidth:    size.Width,
			ShowSidebar: false,
			ShowDetails: false,
		}
	case SizeClassSmall:
		return ResponsiveLayout{
			Columns:     1,
			MaxWidth:    min(size.Width, 70),
			ShowSidebar: false,
			ShowDetails: true,
		}
	case SizeClassMedium:
		return ResponsiveLayout{
			Columns:     2,
			MaxWidth:    min(size.Width-20, 100),
			ShowSidebar: true,
			ShowDetails: true,
		}
	case SizeClassLarge:
		return ResponsiveLayout{
			Columns:     3,
			MaxWidth:    min(size.Width-30, 120),
			ShowSidebar: true,
			ShowDetails: true,
		}
	default: // SizeClassHuge
		return ResponsiveLayout{
			Columns:     4,
			MaxWidth:    min(size.Width-40, 140),
			ShowSidebar: true,
			ShowDetails: true,
		}
	}
}

// min returns the smaller of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
