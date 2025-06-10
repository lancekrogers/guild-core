package git

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// CommitInfo represents parsed commit information
type CommitInfo struct {
	Hash       string
	ShortHash  string
	Author     string
	AuthorDate time.Time
	Message    string
	Subject    string
	Body       string
	Files      []string
}

// BlameInfo represents parsed blame information for a line
type BlameInfo struct {
	Commit      string
	Author      string
	AuthorTime  time.Time
	LineNumber  int
	LineContent string
}

// ConflictInfo represents a merge conflict
type ConflictInfo struct {
	File           string
	ConflictCount  int
	OurMarkers     int
	TheirMarkers   int
	ConflictBlocks []ConflictBlock
}

// ConflictBlock represents a single conflict block
type ConflictBlock struct {
	StartLine int
	EndLine   int
	OurLines  []string
	BaseLines []string
	TheirLines []string
}

// parseGitLog parses git log output in a specific format
func parseGitLog(output string) []CommitInfo {
	commits := []CommitInfo{}
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		// Parse simple format: hash message
		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			continue
		}

		commit := CommitInfo{
			Hash:      parts[0],
			ShortHash: parts[0][:7],
			Message:   parts[1],
			Subject:   parts[1],
		}

		commits = append(commits, commit)
	}

	return commits
}

// parseGitLogVerbose parses detailed git log output
func parseGitLogVerbose(output string) []CommitInfo {
	commits := []CommitInfo{}
	lines := strings.Split(output, "\n")

	var current *CommitInfo
	inMessage := false

	for _, line := range lines {
		// New commit starts
		if strings.HasPrefix(line, "commit ") {
			if current != nil {
				commits = append(commits, *current)
			}
			hash := strings.TrimPrefix(line, "commit ")
			current = &CommitInfo{
				Hash:      hash,
				ShortHash: hash[:7],
			}
			inMessage = false
		} else if current != nil {
			if strings.HasPrefix(line, "Author: ") {
				current.Author = strings.TrimPrefix(line, "Author: ")
			} else if strings.HasPrefix(line, "Date: ") {
				dateStr := strings.TrimPrefix(line, "Date: ")
				// Parse git date format
				if t, err := parseGitDate(dateStr); err == nil {
					current.AuthorDate = t
				}
			} else if strings.TrimSpace(line) == "" && !inMessage {
				inMessage = true
			} else if inMessage {
				// Message lines are indented with 4 spaces
				if strings.HasPrefix(line, "    ") {
					msgLine := strings.TrimPrefix(line, "    ")
					if current.Subject == "" {
						current.Subject = msgLine
						current.Message = msgLine
					} else {
						current.Message += "\n" + msgLine
						current.Body += msgLine + "\n"
					}
				}
			}
		}
	}

	if current != nil {
		commits = append(commits, *current)
	}

	return commits
}

// parseGitDate parses git's date format
func parseGitDate(dateStr string) (time.Time, error) {
	// Git date format: "Thu Jan 2 15:04:05 2006 -0700"
	formats := []string{
		"Mon Jan 2 15:04:05 2006 -0700",
		"Mon Jan 02 15:04:05 2006 -0700",
		"2006-01-02 15:04:05 -0700",
	}

	dateStr = strings.TrimSpace(dateStr)
	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", dateStr)
}

// parseGitBlame parses git blame output
func parseGitBlame(output string) []BlameInfo {
	blameInfo := []BlameInfo{}
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		// Parse format: hash (author date time timezone linenum) content
		// Example: abc123 (John Doe 2024-01-01 12:00:00 +0000   1) package main
		
		// Find the parentheses
		openParen := strings.Index(line, "(")
		closeParen := strings.Index(line, ")")
		
		if openParen == -1 || closeParen == -1 || openParen >= closeParen {
			continue
		}

		hash := strings.TrimSpace(line[:openParen])
		metaInfo := line[openParen+1 : closeParen]
		content := ""
		if closeParen+1 < len(line) {
			content = line[closeParen+1:]
		}

		// Parse metadata
		// Split by last occurrence of line number to handle names with spaces
		parts := strings.Fields(metaInfo)
		if len(parts) < 5 {
			continue
		}

		// Last field is line number
		lineNumStr := parts[len(parts)-1]
		lineNum, err := strconv.Atoi(lineNumStr)
		if err != nil {
			continue
		}

		// Previous 4 fields are date components
		dateEndIdx := len(parts) - 1
		dateStartIdx := dateEndIdx - 4
		if dateStartIdx < 0 {
			continue
		}

		// Everything before date is author name
		author := strings.Join(parts[:dateStartIdx], " ")
		
		// Parse date
		dateStr := strings.Join(parts[dateStartIdx:dateEndIdx], " ")
		authorTime, _ := parseGitDate(dateStr)

		blame := BlameInfo{
			Commit:      hash,
			Author:      author,
			AuthorTime:  authorTime,
			LineNumber:  lineNum,
			LineContent: strings.TrimSpace(content),
		}

		blameInfo = append(blameInfo, blame)
	}

	return blameInfo
}

// parseConflictedFiles parses the output of git diff --name-only --diff-filter=U
func parseConflictedFiles(output string) []string {
	files := []string{}
	lines := strings.Split(strings.TrimSpace(output), "\n")
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			files = append(files, line)
		}
	}
	
	return files
}

// parseConflictMarkers analyzes a file's content for conflict markers
func parseConflictMarkers(content string) ConflictInfo {
	lines := strings.Split(content, "\n")
	info := ConflictInfo{
		ConflictBlocks: []ConflictBlock{},
	}

	var currentBlock *ConflictBlock
	inOurs := false
	inBase := false
	inTheirs := false

	for i, line := range lines {
		if strings.HasPrefix(line, "<<<<<<<") {
			// Start of conflict
			currentBlock = &ConflictBlock{
				StartLine: i + 1,
				OurLines:  []string{},
				BaseLines: []string{},
				TheirLines: []string{},
			}
			inOurs = true
			info.OurMarkers++
		} else if strings.HasPrefix(line, "|||||||") && currentBlock != nil {
			// Base section (3-way merge)
			inOurs = false
			inBase = true
		} else if strings.HasPrefix(line, "=======") && currentBlock != nil {
			// Separator
			inOurs = false
			inBase = false
			inTheirs = true
		} else if strings.HasPrefix(line, ">>>>>>>") && currentBlock != nil {
			// End of conflict
			currentBlock.EndLine = i + 1
			info.ConflictBlocks = append(info.ConflictBlocks, *currentBlock)
			info.TheirMarkers++
			currentBlock = nil
			inOurs = false
			inBase = false
			inTheirs = false
		} else if currentBlock != nil {
			// Add line to appropriate section
			if inOurs {
				currentBlock.OurLines = append(currentBlock.OurLines, line)
			} else if inBase {
				currentBlock.BaseLines = append(currentBlock.BaseLines, line)
			} else if inTheirs {
				currentBlock.TheirLines = append(currentBlock.TheirLines, line)
			}
		}
	}

	info.ConflictCount = len(info.ConflictBlocks)
	return info
}

// parseGitStatus parses git status output for file states
func parseGitStatus(output string) map[string]string {
	status := make(map[string]string)
	lines := strings.Split(output, "\n")
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		// Parse status codes
		if len(line) > 3 {
			statusCode := line[:2]
			filename := strings.TrimSpace(line[3:])
			
			switch statusCode {
			case "??":
				status[filename] = "untracked"
			case "A ":
				status[filename] = "added"
			case "M ":
				status[filename] = "modified"
			case "D ":
				status[filename] = "deleted"
			case "R ":
				status[filename] = "renamed"
			case "C ":
				status[filename] = "copied"
			case "UU":
				status[filename] = "both modified"
			case "AA":
				status[filename] = "both added"
			case "DD":
				status[filename] = "both deleted"
			}
		}
	}
	
	return status
}