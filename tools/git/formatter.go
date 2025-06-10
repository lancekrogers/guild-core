package git

import (
	"fmt"
	"strings"
	"time"
)

// formatCommitHistory formats a list of commits for display
func formatCommitHistory(commits []CommitInfo) string {
	if len(commits) == 0 {
		return "No commits found"
	}

	var output []string
	for _, commit := range commits {
		line := fmt.Sprintf("%s %s", commit.ShortHash, commit.Subject)
		output = append(output, line)
	}

	return strings.Join(output, "\n")
}

// formatCommitHistoryVerbose formats commits with full details
func formatCommitHistoryVerbose(commits []CommitInfo) string {
	if len(commits) == 0 {
		return "No commits found"
	}

	var output []string
	for i, commit := range commits {
		if i > 0 {
			output = append(output, "") // Add blank line between commits
		}

		output = append(output, fmt.Sprintf("commit %s", commit.Hash))
		if commit.Author != "" {
			output = append(output, fmt.Sprintf("Author: %s", commit.Author))
		}
		if !commit.AuthorDate.IsZero() {
			output = append(output, fmt.Sprintf("Date:   %s", formatGitDate(commit.AuthorDate)))
		}
		output = append(output, "")

		// Indent commit message
		messageLines := strings.Split(commit.Message, "\n")
		for _, line := range messageLines {
			output = append(output, fmt.Sprintf("    %s", line))
		}
	}

	return strings.Join(output, "\n")
}

// formatGitDate formats a time.Time in git's preferred format
func formatGitDate(t time.Time) string {
	return t.Format("Mon Jan 2 15:04:05 2006 -0700")
}

// formatBlameOutput formats blame information for display
func formatBlameOutput(blameInfo []BlameInfo) string {
	if len(blameInfo) == 0 {
		return "No blame information available"
	}

	var output []string

	// Find the maximum line number for padding
	maxLineNum := 0
	for _, info := range blameInfo {
		if info.LineNumber > maxLineNum {
			maxLineNum = info.LineNumber
		}
	}
	lineNumWidth := len(fmt.Sprintf("%d", maxLineNum))

	// Format each line
	for _, info := range blameInfo {
		author := info.Author
		if len(author) > 20 {
			author = author[:17] + "..."
		}

		line := fmt.Sprintf("%8s %-20s %*d: %s",
			info.Commit[:8],
			author,
			lineNumWidth,
			info.LineNumber,
			info.LineContent,
		)
		output = append(output, line)
	}

	return strings.Join(output, "\n")
}

// formatBlameOutputWithDates includes dates in blame output
func formatBlameOutputWithDates(blameInfo []BlameInfo) string {
	if len(blameInfo) == 0 {
		return "No blame information available"
	}

	var output []string

	// Find the maximum line number for padding
	maxLineNum := 0
	for _, info := range blameInfo {
		if info.LineNumber > maxLineNum {
			maxLineNum = info.LineNumber
		}
	}
	lineNumWidth := len(fmt.Sprintf("%d", maxLineNum))

	// Format each line
	for _, info := range blameInfo {
		author := info.Author
		if len(author) > 15 {
			author = author[:12] + "..."
		}

		dateStr := info.AuthorTime.Format("2006-01-02")

		line := fmt.Sprintf("%8s (%s %s %*d) %s",
			info.Commit[:8],
			author,
			dateStr,
			lineNumWidth,
			info.LineNumber,
			info.LineContent,
		)
		output = append(output, line)
	}

	return strings.Join(output, "\n")
}

// formatConflictList formats a list of conflicts for display
func formatConflictList(conflicts []ConflictInfo) string {
	if len(conflicts) == 0 {
		return "No merge conflicts found"
	}

	var output []string
	output = append(output, fmt.Sprintf("Found %d file(s) with conflicts:", len(conflicts)))
	output = append(output, "")

	for _, conflict := range conflicts {
		output = append(output, fmt.Sprintf("  %s:", conflict.File))
		output = append(output, fmt.Sprintf("    - %d conflict block(s)", conflict.ConflictCount))

		// Show line ranges for each conflict
		for i, block := range conflict.ConflictBlocks {
			output = append(output, fmt.Sprintf("    - Block %d: lines %d-%d", i+1, block.StartLine, block.EndLine))
		}
		output = append(output, "")
	}

	// Summary
	totalConflicts := 0
	for _, c := range conflicts {
		totalConflicts += c.ConflictCount
	}
	output = append(output, fmt.Sprintf("Total: %d conflict(s) in %d file(s)", totalConflicts, len(conflicts)))

	return strings.Join(output, "\n")
}

// formatConflictDetails formats detailed conflict information
func formatConflictDetails(conflict ConflictInfo) string {
	var output []string

	output = append(output, fmt.Sprintf("Conflicts in %s:", conflict.File))
	output = append(output, fmt.Sprintf("Total conflict blocks: %d", conflict.ConflictCount))
	output = append(output, "")

	for i, block := range conflict.ConflictBlocks {
		output = append(output, fmt.Sprintf("=== Conflict Block %d (lines %d-%d) ===", i+1, block.StartLine, block.EndLine))

		// Show our version
		output = append(output, "<<<<<<< OURS")
		for _, line := range block.OurLines {
			output = append(output, line)
		}

		// Show base version if available (3-way merge)
		if len(block.BaseLines) > 0 {
			output = append(output, "||||||| BASE")
			for _, line := range block.BaseLines {
				output = append(output, line)
			}
		}

		output = append(output, "=======")

		// Show their version
		for _, line := range block.TheirLines {
			output = append(output, line)
		}
		output = append(output, ">>>>>>> THEIRS")
		output = append(output, "")
	}

	return strings.Join(output, "\n")
}

// formatConflictSummary provides a brief summary of conflicts
func formatConflictSummary(conflicts []ConflictInfo) string {
	if len(conflicts) == 0 {
		return "✓ No merge conflicts"
	}

	totalBlocks := 0
	for _, c := range conflicts {
		totalBlocks += c.ConflictCount
	}

	if len(conflicts) == 1 {
		return fmt.Sprintf("⚠️  1 file with %d conflict block(s)", totalBlocks)
	}

	return fmt.Sprintf("⚠️  %d files with %d total conflict block(s)", len(conflicts), totalBlocks)
}

// countConflictMarkers counts the total number of conflict markers
func countConflictMarkers(conflicts []ConflictInfo) int {
	total := 0
	for _, c := range conflicts {
		total += c.OurMarkers + c.TheirMarkers
	}
	return total
}

// countUniqueAuthors counts unique authors in blame info
func countUniqueAuthors(blameInfo []BlameInfo) int {
	authors := make(map[string]bool)
	for _, info := range blameInfo {
		authors[info.Author] = true
	}
	return len(authors)
}

// findOldestCommit finds the oldest commit in blame info
func findOldestCommit(blameInfo []BlameInfo) string {
	if len(blameInfo) == 0 {
		return ""
	}

	oldest := blameInfo[0]
	for _, info := range blameInfo[1:] {
		if info.AuthorTime.Before(oldest.AuthorTime) {
			oldest = info
		}
	}

	return oldest.Commit[:8]
}
