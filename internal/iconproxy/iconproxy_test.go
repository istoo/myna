package iconproxy

import (
	"net/url"
	"testing"
)

func TestURLCarriesThePictureWholeThroughAProxy(t *testing.T) {
	proxy := "https://icons.example/fetch"
	t.Setenv("MYNA_ICON_PROXY", proxy)
	pic := "https://github.com/magpie-community.png?size=96&x=a b#c"
	got := URL(pic)
	u, err := url.Parse(got)
	if err != nil {
		t.Fatal(err)
	}
	if u.Scheme+"://"+u.Host+u.Path != proxy {
		t.Fatalf("URL(%q) = %q, not on %s", pic, got, proxy)
	}
	if q := u.Query(); len(q) != 1 || q.Get("url") != pic {
		t.Fatalf("URL(%q) = %q: query %v", pic, got, q)
	}
}

func TestURLAsksTheHostWithoutAProxy(t *testing.T) {
	t.Setenv("MYNA_ICON_PROXY", "")
	for _, u := range []string{
		"https://github.com/istoo/myna/partners/x.png",
		"https://example.com/x.png?a=1&b=2",
	} {
		if got := URL(u); got != u {
			t.Errorf("URL(%q) = %q, want it unchanged", u, got)
		}
	}
}
