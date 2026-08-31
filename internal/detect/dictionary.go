package detect

import (
	"regexp"
	"sort"
	"strings"
)

// dictionaryDetector matches a fixed, configurable list of sensitive terms
// (e.g. employee names, internal project codenames, company legal names).
// It stands in for model-based NER for free-text proper nouns in the MVP and
// is updated at runtime from the policy configuration.
type dictionaryDetector struct {
	name  string
	etype string
	re    *regexp.Regexp
	score float64
}

// NewDictionaryDetector builds a case-insensitive whole-word detector for the
// supplied terms, tagged with the given entity type.
func NewDictionaryDetector(etype string, words []string, score float64) Detector {
	clean := make([]string, 0, len(words))
	for _, w := range words {
		w = strings.TrimSpace(w)
		if w != "" {
			clean = append(clean, regexp.QuoteMeta(w))
		}
	}
	if len(clean) == 0 {
		return &dictionaryDetector{name: "dict:" + etype, etype: etype, score: score}
	}
	sort.Strings(clean)
	pattern := `(?i)\b(?:` + strings.Join(clean, "|") + `)\b`
	return &dictionaryDetector{
		name:  "dict:" + etype,
		etype: etype,
		re:    regexp.MustCompile(pattern),
		score: score,
	}
}

func (d *dictionaryDetector) Name() string { return d.name }

func (d *dictionaryDetector) Detect(text string) ([]Entity, error) {
	if d.re == nil {
		return nil, nil
	}
	matches := d.re.FindAllStringIndex(text, -1)
	var out []Entity
	for _, m := range matches {
		out = append(out, Entity{
			Type:  d.etype,
			Text:  text[m[0]:m[1]],
			Start: m[0],
			End:   m[1],
			Score: d.score,
		})
	}
	return out, nil
}
