package detect

import (
	"regexp"
	"strings"
)

// regexDetector matches a fixed pattern and optionally validates each match
// (e.g. Luhn checksum for credit cards, verification digit for RUT).
type regexDetector struct {
	name   string
	etype  string
	re     *regexp.Regexp
	score  float64
	verify func(string) bool
}

func (r *regexDetector) Name() string { return r.name }

func (r *regexDetector) Detect(text string) ([]Entity, error) {
	matches := r.re.FindAllStringIndex(text, -1)
	var out []Entity
	for _, m := range matches {
		val := text[m[0]:m[1]]
		if r.verify != nil && !r.verify(val) {
			continue
		}
		out = append(out, Entity{
			Type:  r.etype,
			Text:  val,
			Start: m[0],
			End:   m[1],
			Score: r.score,
		})
	}
	return out, nil
}

// NewRegexDetector builds a structured (regex-based) detector.
func NewRegexDetector(etype, pattern string, score float64, verify func(string) bool) Detector {
	return &regexDetector{
		name:   "regex:" + etype,
		etype:  etype,
		re:     regexp.MustCompile(pattern),
		score:  score,
		verify: verify,
	}
}

// --- Validators -----------------------------------------------------------

// onlyDigits strips spaces, dashes and dots and returns the digit string.
func onlyDigits(s string) string {
	return strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, s)
}

// luhnValid validates a credit-card number using the Luhn algorithm.
func luhnValid(num string) bool {
	digits := onlyDigits(num)
	if len(digits) < 13 || len(digits) > 19 {
		return false
	}
	sum := 0
	double := false
	for i := len(digits) - 1; i >= 0; i-- {
		d := int(digits[i] - '0')
		if double {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
		double = !double
	}
	return sum%10 == 0
}

// rutValid validates a Chilean RUT (with verification digit).
func rutValid(rut string) bool {
	clean := strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(rut, ".", ""), "-", ""))
	if len(clean) < 8 {
		return false
	}
	body := clean[:len(clean)-1]
	dv := clean[len(clean)-1]
	digits := onlyDigits(body)
	if len(digits) == 0 {
		return false
	}
	sum := 0
	factor := 2
	for i := len(digits) - 1; i >= 0; i-- {
		sum += int(digits[i]-'0') * factor
		factor++
		if factor > 7 {
			factor = 2
		}
	}
	mod := 11 - (sum % 11)
	var expected byte
	switch mod {
	case 11:
		expected = '0'
	case 10:
		expected = 'K'
	default:
		expected = byte('0' + mod)
	}
	return dv == expected
}

// --- Constructors for the bundled detectors -------------------------------

// StructuredDetectors returns regex detectors for the common structured PII
// categories. enablePhone enables the (noisier) phone detector.
func StructuredDetectors(enablePhone bool) []Detector {
	ds := []Detector{
		NewRegexDetector("EMAIL", `[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`, 0.95, nil),
		NewRegexDetector("CREDIT_CARD", `\b(?:\d[ -]?){13,19}\b`, 0.9, luhnValid),
		NewRegexDetector("RUT", `\b\d{1,3}(?:\.\d{3}){2,3}-[\dkK]\b|\b\d{7,8}-[\dkK]\b`, 0.9, rutValid),
		NewRegexDetector("IPV4", `\b(?:\d{1,3}\.){3}\d{1,3}\b`, 0.7, nil),
	}
	if enablePhone {
		ds = append(ds, NewRegexDetector("PHONE",
			`\+?\d[\d\s().-]{7,}\d`, 0.6, nil))
	}
	return ds
}
