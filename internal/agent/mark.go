package agent

import (
	"net/url"
	"strings"

	"github.com/istoo/owl/internal/edit"
	"github.com/istoo/owl/internal/gateway"
)

// MarkWritten has every TOML, YAML and dotenv line owl writes that
// points an agent at it carry a comment saying so and how to undo it
// (lineNote). A Codex left with openai_base_url on a gateway that no
// longer ran, after owl's folders were deleted by hand, failed every
// turn with nothing in its config naming owl (Jevik_AI on X). Set by
// main; tests of an agent's exact bytes run without it.
func MarkWritten() { edit.Mark = lineNote }

const undo = "Disconnect in owl or `owl uninstall` undoes it"

// lineNote is the note a config line gets, "" for one that isn't
// owl's: a URL on the gateway's port, and Codex's own keys that name
// owl.
func lineNote(table, key, value string) string {
	switch {
	case table == "" && key == "model_provider" && value == owlID:
		return "Codex on owl's local gateway; " + undo + ", or delete this line and the model line"
	case table == "" && key == "model_catalog_json" && strings.Contains(value, "owl"):
		return "owl's model list for Codex; " + undo + ", or delete this line"
	case !onGateway(value):
		return ""
	case table == "model_providers."+owlID:
		return "for Codex threads started on owl; unused while model_provider doesn't name owl"
	case table == "":
		return "points at owl's local gateway; " + undo + ", or delete this line"
	}
	return "points at owl's local gateway; " + undo + ", or remove this entry and pick another model"
}

// onGateway says whether v is a URL on the gateway's port: on loopback, or
// on the address a WSL distro reaches the host by.
func onGateway(v string) bool {
	if !strings.HasPrefix(v, "http://") && !strings.HasPrefix(v, "https://") {
		return false
	}
	u, err := url.Parse(v)
	if err != nil || u.Port() == "" {
		return false
	}
	for _, base := range []string{gateway.URL(), gateway.SavedURL()} {
		if b, err := url.Parse(base); err == nil && b.Port() == u.Port() {
			return true
		}
	}
	return false
}
