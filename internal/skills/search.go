package skills

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var tokensPattern = regexp.MustCompile(`[a-z0-9][a-z0-9_-]{1,}`)
var stopWords = map[string]bool{"when": true, "use": true, "with": true, "from": true, "for": true, "that": true, "this": true, "have": true, "needs": true, "analyze": true, "testing": true, "security": true}

func (i *Index) Search(_ context.Context, query Query) ([]Candidate, error) {
	terms := tokenize(query.Objective)
	for _, observation := range query.Observations {
		terms = append(terms, tokenize(observation.Summary)...)
	}
	for _, hypothesis := range query.Hypotheses {
		terms = append(terms, tokenize(hypothesis.Claim)...)
	}
	termSet := set(terms)
	i.mu.RLock()
	documents := make([]Document, 0, len(i.documents))
	for _, document := range i.documents {
		documents = append(documents, document)
	}
	i.mu.RUnlock()
	var candidates []Candidate
	for _, document := range documents {
		metadata := document.Metadata
		weighted := map[string]float64{}
		for _, token := range tokenize(metadata.Name) {
			weighted[token] += 3
		}
		for _, token := range tokenize(metadata.Description) {
			weighted[token] += 1
		}
		for _, token := range metadata.Domains {
			weighted[strings.ToLower(token)] += 2
		}
		for _, token := range metadata.Intents {
			weighted[strings.ToLower(token)] += 2
		}
		score := 0.0
		var matched []string
		for term := range termSet {
			if weight := weighted[term]; weight > 0 {
				score += weight
				matched = append(matched, term)
			}
		}
		if score == 0 {
			continue
		}
		sort.Strings(matched)
		candidates = append(candidates, Candidate{Metadata: metadata, Score: score, Reason: fmt.Sprintf("matched: %s", strings.Join(matched, ", "))})
	}
	sort.Slice(candidates, func(a, b int) bool {
		if candidates[a].Score == candidates[b].Score {
			return candidates[a].Metadata.Name < candidates[b].Metadata.Name
		}
		return candidates[a].Score > candidates[b].Score
	})
	limit := query.Limit
	if limit <= 0 {
		limit = 8
	}
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}
	return candidates, nil
}

func tokenize(value string) []string {
	raw := tokensPattern.FindAllString(strings.ToLower(value), -1)
	result := make([]string, 0, len(raw))
	for _, token := range raw {
		for _, part := range strings.FieldsFunc(token, func(r rune) bool { return r == '-' || r == '_' }) {
			if len(part) > 1 && !stopWords[part] {
				result = append(result, part)
			}
		}
	}
	return result
}
func set(values []string) map[string]bool {
	result := map[string]bool{}
	for _, value := range values {
		result[value] = true
	}
	return result
}
