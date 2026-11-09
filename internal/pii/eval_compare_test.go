package pii

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/istoo/myna/internal/redact"
)

// The two local models on one set of hand-marked requests (testdata/eval.json,
// the same set internal/pii/guard measures AstrLink Guard on), so Settings ›
// Privacy's choice is a measured one: who finds more of what a person marked,
// and what each calls PII in text that has none.
type evalCase struct {
	Text string `json:"text"`
	Want []struct {
		Text  string `json:"text"`
		Label string `json:"label"`
	} `json:"want"`
}

func loadEval(t *testing.T) []evalCase {
	t.Helper()
	b, err := os.ReadFile("testdata/eval.json")
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

type score struct {
	want, found, hit, wrong int
	wantBytes, hitBytes     int
	overMasked              int
	ms                      int64
}

func (s score) precision() float64 {
	if s.found == 0 {
		return 0
	}
	return float64(s.hit) / float64(s.found)
}
func (s score) recall() float64 {
	if s.want == 0 {
		return 0
	}
	return float64(s.hit) / float64(s.want)
}
func (s score) f1() float64 {
	p, r := s.precision(), s.recall()
	if p+r == 0 {
		return 0
	}
	return 2 * p * r / (p + r)
}

// sameKind says two labels name the same kind of thing: the guard's own
// (email, phone, account) and the names masking uses for them
// (private_email, private_phone, account_number — guardLabels), so the two
// models can be scored on one set.
func sameKind(a, b string) bool {
	kind := func(l string) string {
		switch l {
		case "email", "private_email":
			return "email"
		case "phone", "private_phone":
			return "phone"
		case "account", "account_number", "payment_card":
			return "account"
		case "common_secret", "secret":
			return "secret"
		case "url", "private_url":
			return "url"
		case "ip_address":
			return "ip_address"
		case "private_person":
			return "person"
		case "private_address":
			return "address"
		case "private_date":
			return "date"
		}
		return l
	}
	return kind(a) == kind(b)
}

// measure runs the set through f once, every text at once, and scores it the
// way guard's test does: a found span covering a marked entity's bytes and
// naming its label is a hit, a span over bytes nobody marked is over-masking.
func measure(t *testing.T, f func([]string) [][]redact.Found, cs []evalCase) score {
	t.Helper()
	texts := make([]string, len(cs))
	for i, c := range cs {
		texts[i] = c.Text
	}
	start := time.Now()
	got := f(texts)
	var s score
	s.ms = time.Since(start).Milliseconds()
	if len(got) != len(cs) {
		t.Fatalf("found %d answers for %d texts", len(got), len(cs))
	}
	for i, c := range cs {
		marked := make([]bool, len(c.Text))
		for _, w := range c.Want {
			at := strings.Index(c.Text, w.Text)
			if at < 0 {
				t.Fatalf("the set is wrong: %q is not in %q", w.Text, c.Text)
			}
			s.want++
			s.wantBytes += len(w.Text)
			hit := false
			for _, f := range got[i] {
				if f.Start <= at && f.End >= at+len(w.Text) {
					if sameKind(f.Label, w.Label) {
						hit = true
						s.hit++
						s.hitBytes += len(w.Text)
					} else {
						s.wrong++
					}
					break
				}
			}
			for j := at; j < at+len(w.Text); j++ {
				marked[j] = true
			}
			_ = hit
		}
		s.found += len(got[i])
		for _, f := range got[i] {
			for j := f.Start; j < f.End && j < len(c.Text); j++ {
				if !marked[j] {
					s.overMasked++
				}
			}
		}
	}
	return s
}

// TestEvalBothModels is the comparison a user asks for when choosing in
// Settings: it skips a model that isn't on this machine (CI has neither).
func TestEvalBothModels(t *testing.T) {
	// the models live in the real home's cache, not the test's own home:
	// this one-off comparison runs with MYNA_EVAL_HOME=$HOME (CI has no
	// models, and skips)
	if h := os.Getenv("MYNA_EVAL_HOME"); h != "" {
		os.Setenv("HOME", h)
		os.Setenv("XDG_CACHE_HOME", filepath.Join(h, ".cache"))
	}
	cs := loadEval(t)
	type row struct {
		name string
		s    score
	}
	var rows []row
	for _, m := range Models() {
		if !m.Supported() || !m.Installed() {
			t.Logf("%s: not installed here, skipped", m.ID())
			continue
		}
		m.Warm()
		s := measure(t, func(texts []string) [][]redact.Found {
			got, err := m.Find(context.Background(), texts)
			if err != nil {
				t.Fatalf("%s: %v", m.ID(), err)
			}
			return got
		}, cs)
		rows = append(rows, row{m.ID(), s})
		t.Logf("%s: entities %d/%d (precision %.3f, recall %.3f, F1 %.3f; %d another label), "+
			"bytes %d/%d masked, %d over-masked, %d ms for %d requests",
			m.ID(), s.hit, s.want, s.precision(), s.recall(), s.f1(), s.wrong,
			s.hitBytes, s.wantBytes, s.overMasked, s.ms, len(cs))
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].s.f1() > rows[j].s.f1() })
	for _, r := range rows {
		fmt.Printf("%-16s F1 %.3f  P %.3f  R %.3f  over-masked %d bytes\n", r.name, r.s.f1(), r.s.precision(), r.s.recall(), r.s.overMasked)
	}
	if len(rows) == 0 {
		t.Skip("neither local model is on this machine")
	}
}
