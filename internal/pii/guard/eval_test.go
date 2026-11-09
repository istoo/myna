package guard

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/istoo/myna/internal/appdir"
)

// What the local model finds, measured against text whose entities a person
// marked by hand. The set is myna's own (2026-11-09), not the publisher's
// benchmark: it says what a request's own words are likely to get, in
// Chinese and English, the two languages AstrLink Guard is for.
type evalEntity struct {
	text  string // the substring the model should find, once in the sentence
	label string // its label as the model names it
}

type evalCase struct {
	text string
	want []evalEntity
}

// evalSet: ordinary requests of the kind an agent sends — a mail, a phone,
// a card, an address, a name, a secret — and text with nothing to mask, for
// what the model calls PII in it.
var evalSet = []evalCase{
	// English: one entity each
	{"mail me at alice.chen@example.com before the standup", []evalEntity{{"alice.chen@example.com", "email"}}},
	{"dial (415) 555-0132 when you land", []evalEntity{{"(415) 555-0132", "phone"}}},
	{"the card on file is 4242 4242 4242 4242", []evalEntity{{"4242 4242 4242 4242", "payment_card"}}},
	{"prod-db answers at 192.168.10.24 on 5432", []evalEntity{{"192.168.10.24", "ip_address"}}},
	{"the runbook is at https://intranet.example.com/ops/runbook", []evalEntity{{"https://intranet.example.com/ops/runbook", "url"}}},
	{"set the password to Hunter2-Prod! and restart", []evalEntity{{"Hunter2-Prod!", "common_secret"}}},
	{"ship it to 1188 Mission St, San Francisco", []evalEntity{{"1188 Mission St, San Francisco", "private_address"}}},
	{"her birthday is 1987-04-12, remember it", []evalEntity{{"1987-04-12", "private_date"}}},
	{"Dana Whitfield is the new lead", []evalEntity{{"Dana Whitfield", "private_person"}}},
	{"user jdoe1985 signed in from a new laptop", []evalEntity{{"jdoe1985", "account"}}},
	// English: two entities
	{"write to maria.gomez@corp.example and cc ada@corp.example", []evalEntity{{"maria.gomez@corp.example", "email"}, {"ada@corp.example", "email"}}},
	{"call (415) 555-0143 or mail ops@relay.example", []evalEntity{{"(415) 555-0143", "phone"}, {"ops@relay.example", "email"}}},
	{"the box at 10.20.30.40 serves files.example.dev", []evalEntity{{"10.20.30.40", "ip_address"}}},
	{"token 550e8400-e29b-41d4-a716-446655440000 goes in the header", []evalEntity{{"550e8400-e29b-41d4-a716-446655440000", "common_secret"}}},
	{"Dana Whitfield's number is (628) 555-0177", []evalEntity{{"Dana Whitfield", "private_person"}, {"(628) 555-0177", "phone"}}},
	// English: nothing to mask
	{"the build failed on the third attempt, so I retried it", nil},
	{"OpenAI released a new model with a longer context", nil},
	{"increase the timeout to 30 seconds and run it again", nil},
	{"the weather in Denver was fine for the demo", nil},
	{"rename the function to parse_request and update callers", nil},
	{"password is correct, retry the request", nil},
	// English: forms that are common and easy to miss
	{"contact +1 415 555 0148 for the shipment", []evalEntity{{"+1 415 555 0148", "phone"}}},
	{"ssh into 10.1.2.3 as ops and rotate the key", []evalEntity{{"10.1.2.3", "ip_address"}}},
	{"the api key sk-live-9f8a7b6c5d4e3f2a1b0c goes in .env", []evalEntity{{"sk-live-9f8a7b6c5d4e3f2a1b0c", "common_secret"}}},
	{"Dana Whitfield and Priya Raman joined the call", []evalEntity{{"Dana Whitfield", "private_person"}, {"Priya Raman", "private_person"}}},
	// Chinese: one entity each
	{"我的邮箱是 zhang.wei@example.cn，请抄送给他", []evalEntity{{"zhang.wei@example.cn", "email"}}},
	{"手机号 13800138000，下午联系", []evalEntity{{"13800138000", "phone"}}},
	{"身份证号 11010519491231002X，帮我核对", []evalEntity{{"11010519491231002X", "account"}}},
	{"银行卡 6222020200112233445 已绑定", []evalEntity{{"6222020200112233445", "payment_card"}}},
	{"服务器 10.0.0.8 拒绝连接", []evalEntity{{"10.0.0.8", "ip_address"}}},
	{"文档在 http://intra.example.cn/wiki/deploy", []evalEntity{{"http://intra.example.cn/wiki/deploy", "url"}}},
	{"密码改成 Xk9mQ2vP 后重启", []evalEntity{{"Xk9mQ2vP", "common_secret"}}},
	{"家住北京市朝阳区建国路88号", []evalEntity{{"北京市朝阳区建国路88号", "private_address"}}},
	{"出生日期 1990-08-16", []evalEntity{{"1990-08-16", "private_date"}}},
	{"联系人李伟，明天到公司", []evalEntity{{"李伟", "private_person"}}},
	// Chinese: two entities
	{"联系 zhang.wei@example.cn 或打 13900139000", []evalEntity{{"zhang.wei@example.cn", "email"}, {"13900139000", "phone"}}},
	{"李伟的地址是上海市浦东新区世纪大道100号", []evalEntity{{"李伟", "private_person"}, {"上海市浦东新区世纪大道100号", "private_address"}}},
	{"账号 admin_ops 密码 Xk9mQ2vP", []evalEntity{{"admin_ops", "account"}, {"Xk9mQ2vP", "common_secret"}}},
	// Chinese: nothing to mask
	{"构建在第三次尝试时失败，我重新跑了一遍", nil},
	{"把超时改成 30 秒再试一次", nil},
	{"版本 1.2.3 修复了三个问题", nil},
	{"请更新 README 里的安装步骤", nil},
	{"这个函数的返回值需要排序", nil},
	// Chinese: forms that are common and easy to miss
	{"联系电话 13700137000，尽快回电", []evalEntity{{"13700137000", "phone"}}},
	{"把 192.168.1.1 改成 10.0.0.1 再试", []evalEntity{{"192.168.1.1", "ip_address"}, {"10.0.0.1", "ip_address"}}},
	{"11 月 3 日的会议改到下午", nil},
}

// evalDir is the model's folder: MYNA_PII_DIR when it is set, else the
// newest revision in myna's cache. "" when there is none, and the eval
// tests skip: CI has no model, and a test never reaches the network.
func evalDir(t *testing.T) string {
	t.Helper()
	if d := os.Getenv("MYNA_PII_DIR"); d != "" {
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
		t.Skip("no AstrLink Guard model here (MYNA_PII_DIR, or myna's cache)")
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
func evaluate(t *testing.T, m *Model) evalResult {
	t.Helper()
	var r evalResult
	for _, c := range evalSet {
		spans := m.Find(c.text)
		marked := make([]bool, len(c.text))
		for _, w := range c.want {
			at := strings.Index(c.text, w.text)
			if at < 0 {
				t.Fatalf("the set is wrong: %q is not in %q", w.text, c.text)
			}
			if strings.Count(c.text, w.text) > 1 {
				t.Fatalf("the set is wrong: %q is twice in %q", w.text, c.text)
			}
			r.want++
			r.wantBytes += len(w.text)
			found := false
			for _, s := range spans {
				if s.Start <= at && s.End >= at+len(w.text) {
					if s.Label == w.label {
						found = true
						r.hit++
						r.hitBytes += len(w.text)
					} else {
						r.wrongLabel++
					}
					break
				}
			}
			if !found {
				r.missed = append(r.missed, fmt.Sprintf("%s (%s) in %q", w.text, w.label, c.text))
			}
			for i := at; i < at+len(w.text); i++ {
				marked[i] = true
			}
		}
		r.found += len(spans)
		for _, s := range spans {
			inside := false
			for i := s.Start; i < s.End && i < len(c.text); i++ {
				if marked[i] {
					inside = true
					break
				}
			}
			if !inside {
				r.falsePositives = append(r.falsePositives, fmt.Sprintf("%q (%s %.2f)", c.text[s.Start:s.End], s.Label, s.Score))
			}
		}
	}
	// what a span covers that nobody marked is the real over-masking count
	r.extra = 0
	for _, c := range evalSet {
		spans := m.Find(c.text)
		marked := make([]bool, len(c.text))
		for _, w := range c.want {
			at := strings.Index(c.text, w.text)
			for i := at; i < at+len(w.text); i++ {
				marked[i] = true
			}
		}
		for _, s := range spans {
			for i := s.Start; i < s.End && i < len(c.text); i++ {
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
	ml := evaluate(t, m)
	t.Logf("CoreML/bert on the device: %s", ml)

	// the same model read on the CPU, for both to be compared on one machine
	cpu := &Model{tok: m.tok, net: m.net}
	bert := evaluate(t, cpu)
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
