package guard

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/istoo/owl/internal/appdir"
)

// What the local model finds, measured against text whose entities a person
// marked by hand. The set is owl's own (2026-11-09), not the publisher's
// benchmark: it says what a request's own words are likely to get, in
// Chinese and English, the two languages AstrLink Guard is for.
type evalEntity struct {
	Text  string `json:"text"`  // the substring the model should find, once in the sentence
	Label string `json:"label"` // its label as the model names it
}

type evalCase struct {
	Text string       `json:"text"`
	Want []evalEntity `json:"want"`
}

// loadEval reads the set both this package and internal/pii measure, so a
// case is marked once.
func loadEval(t *testing.T) []evalCase {
	t.Helper()
	b, err := os.ReadFile("../testdata/eval.json")
	if err != nil {
		t.Fatal(err)
	}
	var cs []evalCase
	if err := json.Unmarshal(b, &cs); err != nil {
		t.Fatal(err)
	}
	if len(cs) < 40 {
		t.Fatalf("the set has %d cases", len(cs))
	}
	return cs
}

// evalSet: ordinary requests of the kind an agent sends — a mail, a phone,
// a card, an address, a name, a secret — and text with nothing to mask, for
// what the model calls PII in it.

// evalDir is the model's folder: OWL_PII_DIR when it is set, else the
// newest revision in owl's cache. "" when there is none, and the eval
// tests skip: CI has no model, and a test never reaches the network.
func evalDir(t *testing.T) string {
	t.Helper()
	if d := os.Getenv("OWL_PII_DIR"); d != "" {
		return d
	}
	root := filepath.Join(appdir.Cache(), "astrlink-guard")
	ds, _ := filepath.Glob(filepath.Join(root, "*"))
	var with []string
	for _, d := range ds {
		if _, err := os.Stat(filepath.Join(d, "model_int8.onnx")); err == nil {
			with = append(with, d)
		}
	}
	if len(with) == 0 {
		t.Skip("no AstrLink Guard model here (OWL_PII_DIR, or owl's cache)")
	}
	sort.Strings(with)
	return with[len(with)-1]
}

// evalResult is what one pass over the set came to, per entity and per
// byte: what it found of what was marked, and what else it called PII.
type evalResult struct {
	want, found, hit, wrongLabel int // entities
	wantBytes, hitBytes, extra   int // bytes
	missed                       []string
	falsePositives               []string
}

func (r evalResult) precision() float64 {
	if r.found == 0 {
		return 0
	}
	return float64(r.hit) / float64(r.found)
}

func (r evalResult) recall() float64 {
	if r.want == 0 {
		return 0
	}
	return float64(r.hit) / float64(r.want)
}

func (r evalResult) f1() float64 {
	p, rc := r.precision(), r.recall()
	if p+rc == 0 {
		return 0
	}
	return 2 * p * rc / (p + rc)
}

func (r evalResult) String() string {
	return fmt.Sprintf("entities %d/%d found (precision %.3f, recall %.3f, F1 %.3f; %d with another label), "+
		"bytes %d/%d masked, %d masked that wasn't marked",
		r.hit, r.want, r.precision(), r.recall(), r.f1(), r.wrongLabel, r.hitBytes, r.wantBytes, r.extra)
}

// evaluate runs the set through m and scores it. A found span is a hit when
// it covers a marked entity's bytes and names its label; a span over bytes
// nobody marked is a false positive; a marked entity no span reaches is a
// miss.
func evaluate(t *testing.T, m *Model, cases []evalCase) evalResult {
	t.Helper()
	var r evalResult
	for _, c := range cases {
		spans := m.Find(c.Text)
		marked := make([]bool, len(c.Text))
		for _, w := range c.Want {
			at := strings.Index(c.Text, w.Text)
			if at < 0 {
				t.Fatalf("the set is wrong: %q is not in %q", w.Text, c.Text)
			}
			if strings.Count(c.Text, w.Text) > 1 {
				t.Fatalf("the set is wrong: %q is twice in %q", w.Text, c.Text)
			}
			r.want++
			r.wantBytes += len(w.Text)
			found := false
			for _, s := range spans {
				if s.Start <= at && s.End >= at+len(w.Text) {
					if s.Label == w.Label {
						found = true
						r.hit++
						r.hitBytes += len(w.Text)
					} else {
						r.wrongLabel++
					}
					break
				}
			}
			if !found {
				r.missed = append(r.missed, fmt.Sprintf("%s (%s) in %q", w.Text, w.Label, c.Text))
			}
			for i := at; i < at+len(w.Text); i++ {
				marked[i] = true
			}
		}
		r.found += len(spans)
		for _, s := range spans {
			inside := false
			for i := s.Start; i < s.End && i < len(c.Text); i++ {
				if marked[i] {
					inside = true
					break
				}
			}
			if !inside {
				r.falsePositives = append(r.falsePositives, fmt.Sprintf("%q (%s %.2f)", c.Text[s.Start:s.End], s.Label, s.Score))
			}
		}
	}
	// what a span covers that nobody marked is the real over-masking count
	r.extra = 0
	for _, c := range cases {
		spans := m.Find(c.Text)
		marked := make([]bool, len(c.Text))
		for _, w := range c.Want {
			at := strings.Index(c.Text, w.Text)
			for i := at; i < at+len(w.Text); i++ {
				marked[i] = true
			}
		}
		for _, s := range spans {
			for i := s.Start; i < s.End && i < len(c.Text); i++ {
				if !marked[i] {
					r.extra++
				}
			}
		}
	}
	return r
}

// The model's own accuracy on the set above, from the CoreML program on a
// Mac and from bert.go on the CPU: the same set, both ways, so what CoreML
// costs or gains in findings is seen.
//
// The bounds are the 2026-11-09 numbers with a little room under them; a
// change that lowers them says so in its commit.
func TestEvalAstrLink(t *testing.T) {
	dir := evalDir(t)
	m, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	via, device, why := m.Backend()
	t.Logf("backend: %q on %s (%v)", via, device, why)
	cases := loadEval(t)
	ml := evaluate(t, m, cases)
	t.Logf("CoreML/bert on the device: %s", ml)

	// the same model read on the CPU, for both to be compared on one machine
	cpu := &Model{tok: m.tok, net: m.net}
	bert := evaluate(t, cpu, cases)
	t.Logf("bert.go on the CPU:        %s", bert)
	for _, m := range ml.missed {
		t.Logf("missed: %s", m)
	}
	for _, f := range ml.falsePositives {
		t.Logf("masked that wasn't marked: %s", f)
	}
	// CoreML's probabilities differ from bert.go's by up to .07 (coreml.go),
	// so a borderline entity may fall on either side: said, not failed
	if !reflect.DeepEqual(ml.missed, bert.missed) || !reflect.DeepEqual(ml.falsePositives, bert.falsePositives) {
		t.Logf("CoreML and bert.go differ in what they found")
	}
	// what one window costs, the two ways: a sentence of about 512 tokens
	long := strings.Repeat("the deployment finished at noon and the logs look clean. ", 60) + " mail alice.chen@example.com"
	for _, c := range []struct {
		name string
		m    *Model
	}{{"CoreML", m}, {"bert.go", cpu}} {
		c.m.Find(long) // its first read, and the warm-up
		start := time.Now()
		for range 5 {
			c.m.Find(long)
		}
		t.Logf("%s: %.1f ms a window of about %d tokens", c.name, float64(time.Since(start).Microseconds())/5/1000, len(c.m.tok.encode(long)))
	}
	if ml.hit < 35 {
		t.Errorf("the model found %d of %d marked entities: %v", ml.hit, ml.want, ml.missed)
	}
	if ml.f1() < 0.85 {
		t.Errorf("F1 %.3f is under the 0.85 this set held at (11-09)", ml.f1())
	}
}
