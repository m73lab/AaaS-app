package detect

import (
	"log"
	"sort"
)

// Entity is a detected sensitive value with its position in the source text.
type Entity struct {
	Type  string // category, e.g. EMAIL, CREDIT_CARD, RUT, PHONE, PERSON
	Text  string // the matched sensitive text
	Start int    // byte offset of the match start
	End   int    // byte offset of the match end
	Score float64
}

// Detector identifies sensitive entities inside a text fragment.
type Detector interface {
	Name() string
	Detect(text string) ([]Entity, error)
}

// Combined runs multiple detectors and merges their results, resolving
// overlaps by keeping the higher-score entity (structured/regex wins ties).
type Combined struct {
	detectors []Detector
}

// NewCombined builds a combined detector from the given detectors.
func NewCombined(detectors ...Detector) *Combined {
	return &Combined{detectors: detectors}
}

func (c *Combined) Name() string { return "combined" }

// Detect runs every sub-detector and returns overlap-free merged entities.
// If a detector fails (e.g. Presidio unreachable), it logs the error and
// continues with the remaining detectors so a single failure never kills the
// entire pipeline.
func (c *Combined) Detect(text string) ([]Entity, error) {
	var all []Entity
	var lastErr error
	for _, d := range c.detectors {
		es, err := d.Detect(text)
		if err != nil {
			log.Printf("detect: %s failed: %v", d.Name(), err)
			lastErr = err
			continue
		}
		all = append(all, es...)
	}
	if len(all) == 0 && lastErr != nil {
		return nil, lastErr
	}
	return MergeEntities(all), nil
}

// MergeEntities sorts by position and drops entities overlapped by a
// previously accepted (higher or equal priority) entity.
func MergeEntities(in []Entity) []Entity {
	if len(in) == 0 {
		return nil
	}
	sort.SliceStable(in, func(i, j int) bool {
		if in[i].Start != in[j].Start {
			return in[i].Start < in[j].Start
		}
		if in[i].Score != in[j].Score {
			return in[i].Score > in[j].Score
		}
		return in[i].End > in[j].End
	})
	var out []Entity
	for _, e := range in {
		overlap := false
		for _, o := range out {
			if e.Start < o.End && e.End > o.Start {
				overlap = true
				break
			}
		}
		if !overlap {
			out = append(out, e)
		}
	}
	return out
}
