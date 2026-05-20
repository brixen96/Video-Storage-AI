package services

import (
	"fmt"
	"regexp"
	"strings"
)

// Known tags to extract from titles
var knownTags = []string{
	"XXX", "OnlyFans", "BBW", "T H I C C", "MILF", "Petite",
	"Teen", "Asian", "Indian", "Ebony", "Latina", "Feet", "Retired",
}

// extractPerformerNames attempts to extract performer names from title
func (s *ScraperService) extractPerformerNames(title string) []string {
	var performers []string

	// Common patterns: [Name], (Name), Name - Content
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`\[([^\]]+)\]`),
		regexp.MustCompile(`\(([^)]+)\)`),
	}

	for _, pattern := range patterns {
		matches := pattern.FindAllStringSubmatch(title, -1)
		for _, match := range matches {
			if len(match) > 1 {
				name := strings.TrimSpace(match[1])
				// Filter out common non-name terms
				if !s.isCommonTerm(name) {
					performers = append(performers, name)
				}
			}
		}
	}

	return performers
}

// extractStudioNames attempts to extract studio names from title
func (s *ScraperService) extractStudioNames(title string) []string {
	var studios []string

	// Common studio indicators
	studioKeywords := []string{"Official", "Network", "Productions", "Studios", "Entertainment"}

	words := strings.Fields(title)
	for i, word := range words {
		for _, keyword := range studioKeywords {
			if strings.Contains(word, keyword) && i > 0 {
				studios = append(studios, words[i-1]+" "+word)
			}
		}
	}

	return studios
}

// isCommonTerm checks if a term is a common non-name word
func (s *ScraperService) isCommonTerm(term string) bool {
	commonTerms := []string{"NEW", "HD", "4K", "LEAKED", "EXCLUSIVE", "UPDATED", "MEGA", "PACK"}
	termUpper := strings.ToUpper(term)
	for _, common := range commonTerms {
		if termUpper == common {
			return true
		}
	}
	return false
}

// ExtractTagsFromTitle extracts known tags from thread title
func (s *ScraperService) ExtractTagsFromTitle(title string) []string {
	var extractedTags []string
	titleUpper := strings.ToUpper(title)

	for _, tag := range knownTags {
		// Case-insensitive search
		tagUpper := strings.ToUpper(tag)
		if strings.Contains(titleUpper, tagUpper) {
			// Add tag if not already in list
			if !contains(extractedTags, tag) {
				extractedTags = append(extractedTags, tag)
			}
		}
	}

	return extractedTags
}

// CleanTitleFromTags removes known tags from title and cleans it up
func (s *ScraperService) CleanTitleFromTags(title string) string {
	cleanedTitle := title

	// Remove tags (case-insensitive)
	for _, tag := range knownTags {
		// Try various bracket formats: [TAG], (TAG), {TAG}
		patterns := []string{
			fmt.Sprintf("[%s]", tag),
			fmt.Sprintf("(%s)", tag),
			fmt.Sprintf("{%s}", tag),
			fmt.Sprintf("【%s】", tag), // Japanese brackets
		}

		for _, pattern := range patterns {
			// Case-insensitive replacement
			re := regexp.MustCompile(`(?i)` + regexp.QuoteMeta(pattern))
			cleanedTitle = re.ReplaceAllString(cleanedTitle, "")
		}

		// Also remove standalone tags with spaces
		re := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(tag) + `\b`)
		cleanedTitle = re.ReplaceAllString(cleanedTitle, "")
	}

	// Clean up multiple spaces, dashes, and pipes
	cleanedTitle = regexp.MustCompile(`\s+`).ReplaceAllString(cleanedTitle, " ")
	cleanedTitle = regexp.MustCompile(`\s*[-|]\s*$`).ReplaceAllString(cleanedTitle, "")
	cleanedTitle = regexp.MustCompile(`^\s*[-|]\s*`).ReplaceAllString(cleanedTitle, "")
	cleanedTitle = strings.TrimSpace(cleanedTitle)

	return cleanedTitle
}

// ExtractPerformerNameFromTitle extracts potential performer name from thread title
func (s *ScraperService) ExtractPerformerNameFromTitle(title string) []string {
	// First, clean the title from tags
	cleanedTitle := s.CleanTitleFromTags(title)

	// Common patterns for performer threads:
	// "Performer Name - Title"
	// "[Performer Name]"
	// "Performer Name | something"
	// "Performer Name (aka Other Name)"

	var names []string

	// Remove common prefixes
	cleanedTitle = strings.TrimSpace(cleanedTitle)

	// Pattern 1: Text before dash
	if strings.Contains(cleanedTitle, " - ") {
		parts := strings.Split(cleanedTitle, " - ")
		name := strings.TrimSpace(parts[0])
		// Remove brackets if present
		name = strings.Trim(name, "[]")
		if name != "" {
			names = append(names, name)
		}
	}

	// Pattern 2: Text in brackets
	bracketRegex := regexp.MustCompile(`\[(.*?)\]`)
	matches := bracketRegex.FindAllStringSubmatch(cleanedTitle, -1)
	for _, match := range matches {
		if len(match) > 1 {
			name := strings.TrimSpace(match[1])
			if name != "" && !contains(names, name) {
				names = append(names, name)
			}
		}
	}

	// Pattern 3: Text before pipe
	if strings.Contains(cleanedTitle, " | ") {
		parts := strings.Split(cleanedTitle, " | ")
		name := strings.TrimSpace(parts[0])
		name = strings.Trim(name, "[]")
		if name != "" && !contains(names, name) {
			names = append(names, name)
		}
	}

	// Pattern 4: Extract aka names
	akaRegex := regexp.MustCompile(`\(aka\s+(.*?)\)`)
	matches = akaRegex.FindAllStringSubmatch(cleanedTitle, -1)
	for _, match := range matches {
		if len(match) > 1 {
			name := strings.TrimSpace(match[1])
			if name != "" && !contains(names, name) {
				names = append(names, name)
			}
		}
	}

	return names
}

// contains checks if a string slice contains a string
func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}
