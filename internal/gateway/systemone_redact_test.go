package gateway

import (
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/istoo/myna/internal/provider"
	"github.com/istoo/myna/internal/settings"
)

// System One is asked the user's own text (state.message,
// previous_user_message), and it doesn't go through serve: with masking
// on, what reaches the decision model is masked as any other request's
// text is (#4130).
func TestSystemOneRedacted(t *testing.T) {
	const key = "sk-proj-abcdEFGH1234ijklMNOP5678qrst"
	var got []byte
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"model":"jev-1","answers":{},"usage":{"input_tokens":1,"output_tokens":0}}`)
	}))
	defer up.Close()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	if err := provider.Save(provider.Provider{ID: "jv", Name: "Jev", Key: "kj", Decide: up.URL}); err != nil {
		t.Fatal(err)
	}
	if err := settings.Save(settings.Settings{Redact: true}); err != nil {
		t.Fatal(err)
	}
	body := `{"model":"jv/jev-latest","state":{"message":"my key is ` + key + `"},"questions":{"intent":{"type":"choice","criteria":{"bug":"a bug"}}}}`
	code, out := post(t, "/v1/systemone", body)
	if code != 200 {
		t.Fatalf("%d %s", code, out)
	}
	placeholder := regexp.MustCompile(`\{\{(?:API_KEY|SECRET)_[a-z2-7]{8}\}\}`)
	if strings.Contains(string(got), key) || !placeholder.Match(got) {
		t.Fatalf("the decision model was sent %s", got)
	}
	if !strings.Contains(string(got), "my key is {{") {
		t.Fatalf("the rest of the message was lost: %s", got)
	}
}
