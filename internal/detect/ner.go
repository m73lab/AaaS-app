package detect

import (
	"regexp"
	"strings"
)

// heuristicNER is an interim, dependency-free NER that raises recall for
// free-text proper nouns (names, orgs, places) by detecting runs of
// capitalized words. It is conservative (low score, stop-listed) and exists
// so the product is not blind to out-of-dictionary names before a real model
// (ONNX, see ner_onnx.go) is wired in. It tags matches as PERSON; in
// production prefer the ONNX NER for typed entities (PER/ORG/LOC).
type heuristicNER struct {
	name   string
	etype  string
	score  float64
	stop   map[string]struct{}
	minLen int
}

var capWordRe = regexp.MustCompile(`\b([A-ZÁÉÍÓÚÑ][a-záéíóúñ]*)\b`)

// common capitalized words that are not proper nouns (sentence starts, etc.)
var nerStopwords = map[string]struct{}{
	"i": {}, "we": {}, "the": {}, "this": {}, "that": {}, "these": {}, "those": {},
	"hello": {}, "hi": {}, "hey": {}, "how": {}, "what": {}, "when": {}, "where": {},
	"who": {}, "why": {}, "please": {}, "thanks": {}, "thank": {}, "yes": {}, "no": {},
	"ok": {}, "okay": {}, "hola": {}, "como": {}, "qué": {}, "cómo": {}, "dónde": {},
	"cuándo": {}, "quién": {}, "por": {}, "para": {}, "los": {}, "las": {}, "el": {},
	"la": {}, "una": {}, "un": {}, "yo": {}, "nosotros": {}, "este": {}, "esta": {},
}

// NewHeuristicNER returns a capitalized-word NER. When enabled is false it is
// a no-op (the real ONNX NER should be used instead).
func NewHeuristicNER(enabled bool, etype string, score float64) Detector {
	if !enabled {
		return &nerStub{name: "heuristic:disabled"}
	}
	return &heuristicNER{
		name:   "heuristic:" + etype,
		etype:  etype,
		score:  score,
		stop:   nerStopwords,
		minLen: 2,
	}
}

func (h *heuristicNER) Name() string { return h.name }

func (h *heuristicNER) Detect(text string) ([]Entity, error) {
	matches := capWordRe.FindAllStringIndex(text, -1)
	if len(matches) == 0 {
		return nil, nil
	}
	type span struct {
		start, end int
		word       string
	}
	var spans []span
	for _, m := range matches {
		w := text[m[0]:m[1]]
		if len([]rune(w)) < h.minLen {
			continue
		}
		if _, ok := h.stop[strings.ToLower(w)]; ok {
			continue
		}
		spans = append(spans, span{start: m[0], end: m[1], word: w})
	}
	if len(spans) == 0 {
		return nil, nil
	}
	var out []Entity
	i := 0
	for i < len(spans) {
		s, e := spans[i].start, spans[i].end
		j := i + 1
		// merge consecutive capitalized words with a small gap (<=1 space)
		for j < len(spans) {
			gap := spans[j].start - e
			if gap <= 1 {
				e = spans[j].end
				j++
				continue
			}
			break
		}
		out = append(out, Entity{
			Type:  h.etype,
			Text:  text[s:e],
			Start: s,
			End:   e,
			Score: h.score,
		})
		i = j
	}
	return out, nil
}

// nerStub is a disabled detector that returns no entities.
type nerStub struct{ name string }

func (n *nerStub) Name() string { return n.name }
func (n *nerStub) Detect(text string) ([]Entity, error) {
	return nil, nil
}
