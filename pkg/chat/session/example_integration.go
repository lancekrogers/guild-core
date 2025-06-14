package session

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

// ExampleChatIntegration demonstrates how to integrate the session store
// into a chat interface. This would typically be used in cmd/guild/chat.go
func ExampleChatIntegration() {
	// Open database connection
	db, err := sql.Open("sqlite3", ".guild/memory.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create store and manager
	store := NewSQLiteStore(db)
	manager := NewManager(store)

	// Create or load session
	var session *Session
	sessions, err := store.ListSessions(context.Background(), 1, 0)
	if err != nil || len(sessions) == 0 {
		// Create new session
		session, err = manager.NewSession("Interactive Chat", nil)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Created new session: %s\n", session.ID)
	} else {
		// Load existing session
		session = sessions[0]
		fmt.Printf("Loaded session: %s\n", session.Name)
		
		// Show recent messages
		messages, err := manager.GetContext(session.ID, 5)
		if err == nil {
			fmt.Println("\nRecent messages:")
			for _, msg := range messages {
				fmt.Printf("[%s] %s: %s\n", msg.CreatedAt.Format("15:04"), msg.Role, msg.Content)
			}
		}
	}

	// Example: User sends a message
	userMsg, err := manager.AppendMessage(session.ID, RoleUser, "What's the weather like?", nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("\nUser: %s\n", userMsg.Content)

	// Example: Stream assistant response
	stream, err := manager.StreamMessage(session.ID, RoleAssistant)
	if err != nil {
		log.Fatal(err)
	}

	// Simulate streaming response
	chunks := []string{"The ", "weather ", "today ", "is ", "sunny ", "and ", "warm."}
	for _, chunk := range chunks {
		if err := stream.Write(chunk); err != nil {
			log.Fatal(err)
		}
		fmt.Print(chunk)
	}
	fmt.Println()

	// Close stream and save message
	assistantMsg, err := stream.Close()
	if err != nil {
		log.Fatal(err)
	}

	// Example: Bookmark important message
	bookmark := &Bookmark{
		SessionID: session.ID,
		MessageID: assistantMsg.ID,
		Name:      "Weather Info",
	}
	if err := store.CreateBookmark(context.Background(), bookmark); err != nil {
		log.Fatal(err)
	}

	// Example: Export session
	markdown, err := manager.ExportSession(session.ID, ExportFormatMarkdown)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("\nExported session (%d bytes)\n", len(markdown))
}

// ExampleSessionCommands shows how to implement chat commands for session management
func ExampleSessionCommands() {
	db, _ := sql.Open("sqlite3", ".guild/memory.db")
	defer db.Close()

	store := NewSQLiteStore(db)
	manager := NewManager(store)
	ctx := context.Background()

	// Command: /sessions - List all sessions
	sessions, _ := store.ListSessions(ctx, 10, 0)
	fmt.Println("Available sessions:")
	for i, s := range sessions {
		fmt.Printf("%d. %s (ID: %s, Updated: %s)\n", 
			i+1, s.Name, s.ID, s.UpdatedAt.Format("2006-01-02 15:04"))
	}

	// Command: /new <name> - Create new session
	newSession, _ := manager.NewSession("Project Planning", nil)
	fmt.Printf("\nCreated session: %s\n", newSession.Name)

	// Command: /switch <id> - Switch to different session
	sessionID := sessions[0].ID
	session, _ := manager.LoadSession(sessionID)
	fmt.Printf("\nSwitched to: %s\n", session.Name)

	// Command: /fork <name> - Fork current session
	forked, _ := manager.ForkSession(sessionID, "Alternative Approach")
	fmt.Printf("\nForked session: %s (from %s)\n", forked.Name, sessionID)

	// Command: /clear - Clear current session
	_ = manager.ClearContext(sessionID)
	fmt.Println("\nCleared session messages")

	// Command: /export <format> - Export session
	data, _ := manager.ExportSession(sessionID, ExportFormatJSON)
	fmt.Printf("\nExported to JSON: %d bytes\n", len(data))

	// Command: /bookmarks - Show bookmarks
	bookmarks, _ := store.GetBookmarks(ctx, sessionID)
	fmt.Println("\nBookmarks:")
	for _, b := range bookmarks {
		fmt.Printf("- %s: %s\n", b.Name, b.MessageContent[:50])
	}

	// Command: /search <query> - Search messages
	results, _ := store.SearchMessages(ctx, "project", 10, 0)
	fmt.Printf("\nSearch results for 'project': %d matches\n", len(results))
}