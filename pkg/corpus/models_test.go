package corpus

import (
	"testing"
	"time"
)

func TestNewCorpusDoc(t *testing.T) {
	title := "Test Document"
	source := "Test Source"
	body := "Test Content"
	guildID := "test-guild"
	agentID := "test-agent"
	tags := []string{"test", "document"}

	doc := NewCorpusDoc(title, source, body, guildID, agentID, tags)

	if doc.Title != title {
		t.Errorf("Expected title %s, got %s", title, doc.Title)
	}

	if doc.Source != source {
		t.Errorf("Expected source %s, got %s", source, doc.Source)
	}

	if doc.Body != body {
		t.Errorf("Expected body %s, got %s", body, doc.Body)
	}

	if doc.GuildID != guildID {
		t.Errorf("Expected guildID %s, got %s", guildID, doc.GuildID)
	}

	if doc.AgentID != agentID {
		t.Errorf("Expected agentID %s, got %s", agentID, doc.AgentID)
	}

	if len(doc.Tags) != len(tags) {
		t.Errorf("Expected %d tags, got %d", len(tags), len(doc.Tags))
	}

	for i, tag := range tags {
		if doc.Tags[i] != tag {
			t.Errorf("Expected tag %s, got %s", tag, doc.Tags[i])
		}
	}

	if doc.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}

	if doc.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should not be zero")
	}

	if len(doc.Links) != 0 {
		t.Errorf("Expected 0 links, got %d", len(doc.Links))
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.CorpusPath == "" {
		t.Error("CorpusPath should not be empty")
	}

	if cfg.ActivitiesPath == "" {
		t.Error("ActivitiesPath should not be empty")
	}

	if cfg.MaxSizeBytes != 10*1024*1024 {
		t.Errorf("Expected MaxSizeBytes to be %d, got %d", 10*1024*1024, cfg.MaxSizeBytes)
	}

	if len(cfg.DefaultTags) != 0 {
		t.Errorf("Expected 0 DefaultTags, got %d", len(cfg.DefaultTags))
	}
}