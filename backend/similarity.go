package main

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var nonAlphanumRE = regexp.MustCompile(`[^a-z0-9]+`)

// defaultMatchThreshold is the minimum composite score for a candidate to be
// shown. Configurable via MATCH_SCORE_THRESHOLD environment variable.
const defaultMatchThreshold = 0.15

// rankSimilarDocuments scores each candidate against query, applies threshold
// filtering, and returns up to 5 results sorted by descending score.
//
// Candidates must already be filtered to the same org (caller's responsibility).
// query itself is excluded. Pass threshold=0 to disable filtering.
//
// Scoring weights:
//   - Filename token overlap (Jaccard): 0.30
//   - Customer part number overlap (Jaccard): 0.50
//   - Manufacturer part number overlap (Jaccard): 0.20
func rankSimilarDocuments(query *Document, candidates []*Document, threshold float64) []SimilarDocument {
	if len(candidates) == 0 {
		return []SimilarDocument{}
	}

	queryTokens := filenameTokens(query.Filename)
	queryCPNs := cpnSet(query)
	queryMPNs := mpnSet(query)

	var results []SimilarDocument
	for _, doc := range candidates {
		if doc.ID == query.ID {
			continue
		}
		if doc.Status != StatusDone || len(doc.BOMRows) == 0 {
			continue
		}

		score, breakdown, reasons := computeSimilarity(queryTokens, queryCPNs, queryMPNs, doc)
		if score <= 0 {
			continue
		}
		if score < threshold {
			continue
		}
		results = append(results, SimilarDocument{
			ID:             doc.ID,
			Filename:       doc.Filename,
			UploadedAt:     doc.UploadedAt,
			Score:          score,
			ScoreBreakdown: breakdown,
			MatchReasons:   reasons,
			BOMRowCount:    len(doc.BOMRows),
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
	if len(results) > 5 {
		results = results[:5]
	}
	return results
}

func computeSimilarity(queryTokens, queryCPNs, queryMPNs map[string]bool, doc *Document) (float64, ScoreBreakdown, []string) {
	var bd ScoreBreakdown
	var score float64
	var reasons []string

	// Filename token overlap (weight 0.30).
	docTokens := filenameTokens(doc.Filename)
	bd.Filename = jaccardSimilarity(queryTokens, docTokens)
	if bd.Filename > 0 {
		score += 0.30 * bd.Filename
		reasons = append(reasons, fmt.Sprintf("Filename match (%.0f%%)", bd.Filename*100))
	}

	// Customer part number overlap (weight 0.50).
	docCPNs := cpnSet(doc)
	bd.CPN = jaccardSimilarity(queryCPNs, docCPNs)
	if bd.CPN > 0 {
		score += 0.50 * bd.CPN
		shared := sharedItems(queryCPNs, docCPNs)
		if len(shared) <= 3 {
			reasons = append(reasons, "Shared parts: "+strings.Join(shared, ", "))
		} else {
			reasons = append(reasons, fmt.Sprintf("%d shared part numbers", len(shared)))
		}
	}

	// Manufacturer part number overlap (weight 0.20).
	// Only scored when CPN overlap didn't already match, to avoid double-counting
	// cases where CPN and MPN refer to the same part.
	docMPNs := mpnSet(doc)
	if bd.CPN == 0 {
		docMPNJacc := jaccardSimilarity(queryMPNs, docMPNs)
		bd.MPN = docMPNJacc
		if docMPNJacc > 0 {
			score += 0.20 * docMPNJacc
			shared := sharedItems(queryMPNs, docMPNs)
			if len(shared) <= 3 {
				reasons = append(reasons, "Shared manufacturer parts: "+strings.Join(shared, ", "))
			} else {
				reasons = append(reasons, fmt.Sprintf("%d shared manufacturer part numbers", len(shared)))
			}
		}
	}

	return score, bd, reasons
}

// filenameTokens splits a filename into lowercase alphanumeric tokens of length >= 2.
func filenameTokens(name string) map[string]bool {
	name = strings.ToLower(strings.TrimSuffix(strings.ToLower(name), ".pdf"))
	parts := nonAlphanumRE.Split(name, -1)
	tokens := make(map[string]bool)
	for _, p := range parts {
		if len(p) >= 2 {
			tokens[p] = true
		}
	}
	return tokens
}

// cpnSet returns the uppercased set of non-empty CustomerPartNumbers in doc.
func cpnSet(doc *Document) map[string]bool {
	s := make(map[string]bool)
	for _, row := range doc.BOMRows {
		if cpn := strings.TrimSpace(row.CustomerPartNumber); cpn != "" {
			s[strings.ToUpper(cpn)] = true
		}
	}
	return s
}

// mpnSet returns the uppercased set of non-empty ManufacturerPartNumbers in doc.
func mpnSet(doc *Document) map[string]bool {
	s := make(map[string]bool)
	for _, row := range doc.BOMRows {
		if mpn := strings.TrimSpace(row.ManufacturerPartNumber); mpn != "" {
			s[strings.ToUpper(mpn)] = true
		}
	}
	return s
}

// jaccardSimilarity returns |A ∩ B| / |A ∪ B|. Returns 0 when both sets are empty.
func jaccardSimilarity(a, b map[string]bool) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 0
	}
	intersection := 0
	for k := range a {
		if b[k] {
			intersection++
		}
	}
	union := len(a) + len(b) - intersection
	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}

// sharedItems returns sorted items present in both a and b.
func sharedItems(a, b map[string]bool) []string {
	var out []string
	for k := range a {
		if b[k] {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}
