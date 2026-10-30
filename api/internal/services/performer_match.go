package services

import (
	"path/filepath"
	"regexp"
	"strings"
)

// performer_match.go — ONE shared filename→performer scorer.
//
// Used by AIService.calculateMatch (existing auto-link) and the AI Companion
// tools (resolve_performer_from_filename / link_performers_from_filenames).
// The aggressive part the Companion asked for: filenames are NORMALIZED first
// ("Crystal.Rush.2024.1080p.mp4" → "crystal rush 2024 1080p") so dotted and
// underscored names actually match, and a Levenshtein fallback catches typos.
// Matching is word-boundary based — "ann" can no longer match inside "annual".

var nonAlnumRe = regexp.MustCompile(`[^a-z0-9]+`)

// videoExts are stripped before matching. Only known media extensions —
// stripping blindly would eat years like ".2024" from "scene.2024.mp4".
var videoExts = map[string]bool{
	".mp4": true, ".mkv": true, ".avi": true, ".wmv": true, ".mov": true,
	".m4v": true, ".mpg": true, ".mpeg": true, ".ts": true, ".webm": true,
	".flv": true, ".rmvb": true, ".iso": true,
}

// normalizeForMatch lowercases, strips a known video extension, and collapses
// all separators (. _ - spaces) to single spaces for token matching.
func normalizeForMatch(s string) string {
	s = strings.ToLower(s)
	if ext := filepath.Ext(s); videoExts[ext] {
		s = strings.TrimSuffix(s, ext)
	}
	return strings.TrimSpace(nonAlnumRe.ReplaceAllString(s, " "))
}

// tokenize splits a normalized string into word tokens.
func tokenize(s string) []string {
	return strings.Fields(s)
}

// scorePerformerMatch scores how confidently a filename contains performerName.
// Returns 0.0–1.0 plus a match type: exact | partial | fuzzy | levenshtein | none.
//
// Scoring:
//
//	1.00  all performer tokens present as whole words (normalized)
//	0.95  first AND last name present
//	0.75  last name only (len > 4)   | 0.65 short last name
//	0.70  first name only (len > 4)
//	0.65  any long token (len > 5) present
//	0.60  Levenshtein similarity ≥ 0.85 against a filename token (typo tolerance)
func scorePerformerMatch(filename, performerName string) (float64, string) {
	fnNorm := normalizeForMatch(filename)
	if fnNorm == "" || strings.TrimSpace(performerName) == "" {
		return 0.0, "none"
	}
	fnTokens := tokenize(fnNorm)
	if len(fnTokens) == 0 {
		return 0.0, "none"
	}
	fnSet := make(map[string]bool, len(fnTokens))
	for _, t := range fnTokens {
		fnSet[t] = true
	}

	pNorm := normalizeForMatch(performerName)
	pTokens := tokenize(pNorm)
	if len(pTokens) == 0 {
		return 0.0, "none"
	}

	// 1.00 — every performer token present as a whole word
	allPresent := true
	for _, t := range pTokens {
		if !fnSet[t] {
			allPresent = false
			break
		}
	}
	if allPresent {
		return 1.0, "exact"
	}

	first := pTokens[0]
	last := pTokens[len(pTokens)-1]
	hasFirst := fnSet[first]
	hasLast := fnSet[last]

	if hasFirst && hasLast {
		return 0.95, "partial"
	}
	if hasLast {
		if len(last) > 4 {
			return 0.75, "partial"
		}
		return 0.65, "partial"
	}
	if hasFirst && len(first) > 4 {
		return 0.70, "partial"
	}

	// Any long performer token present as a whole word
	for _, t := range pTokens {
		if len(t) > 5 && fnSet[t] {
			return 0.65, "fuzzy"
		}
	}

	// Levenshtein fallback — typo tolerance ("Rushh" vs "rush")
	for _, pt := range pTokens {
		if len(pt) < 5 {
			continue
		}
		for _, ft := range fnTokens {
			if len(ft) < 4 {
				continue
			}
			if calculateLevenshteinSimilarity(pt, ft) >= 0.85 {
				return 0.60, "levenshtein"
			}
		}
	}

	return 0.0, "none"
}

// performerCandidate is one scored match for a filename.
type performerCandidate struct {
	PerformerID int64   `json:"performer_id"`
	Name        string  `json:"name"`
	Score       float64 `json:"score"`
	MatchType   string  `json:"match_type"`
}

// rankPerformersForFilename scores every performer against one filename and
// returns candidates sorted best-first (only those above minScore).
func rankPerformersForFilename(filename string, performers []pathEntity, minScore float64) []performerCandidate {
	out := []performerCandidate{}
	for _, p := range performers {
		score, matchType := scorePerformerMatch(filename, p.name)
		if score >= minScore {
			out = append(out, performerCandidate{
				PerformerID: p.id, Name: p.name, Score: score, MatchType: matchType,
			})
		}
	}
	// insertion sort — performer counts are small (tens to low hundreds)
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].Score > out[j-1].Score; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

// isAmbiguous reports whether the top two candidates are too close to call —
// the caller must NOT auto-link in that case.
func isAmbiguous(cands []performerCandidate) bool {
	return len(cands) >= 2 && (cands[0].Score-cands[1].Score) < 0.05
}