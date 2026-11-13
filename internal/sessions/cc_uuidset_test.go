package sessions

import (
	"fmt"
	"runtime"
	"testing"
)

// A file's message uuids answer for exactly the strings put in: a uuid in
// another case or form is not one of them, and one not in canonical form
// is kept as written.
func TestCCUUIDSetIsExact(t *testing.T) {
	s := newCCUUIDSet()
	s.add("fb0bf5a9-e23f-4c29-b71c-a184c38bb005")
	s.add("msg_01XYZ")
	s.add("FB0BF5A9-E23F-4C29-B71C-A184C38BB006")
	for v, want := range map[string]bool{
		"fb0bf5a9-e23f-4c29-b71c-a184c38bb005": true,
		"FB0BF5A9-E23F-4C29-B71C-A184C38BB005": false,
		"fb0bf5a9-e23f-4c29-b71c-a184c38bb004": false,
		"fb0bf5a9e23f-4c29-b71c-a184c38bb005-": false,
		"msg_01XYZ":                            true,
		"msg_01xyz":                            false,
		"FB0BF5A9-E23F-4C29-B71C-A184C38BB006": true,
		"fb0bf5a9-e23f-4c29-b71c-a184c38bb006": false,
		"":                                     false,
	} {
		if got := s.has(v); got != want {
			t.Errorf("has(%q) = %v, want %v", v, got, want)
		}
	}
	var none *ccUUIDSet
	if none.has("fb0bf5a9-e23f-4c29-b71c-a184c38bb005") {
		t.Error("a file never read holds no uuid")
	}
}

// The sets are kept for every file of a pair that start alike, one entry
// per message: 0.2.18 held ~600k of them as strings, 51 MB. Each is now
// its 16 bytes in the map's slot, with no string beside it.
func TestCCUUIDSetHoldsNoStrings(t *testing.T) {
	const n = 200_000
	ids := make([]string, n)
	for i := range ids {
		ids[i] = fmt.Sprintf("%08x-e23f-4c29-b71c-%012x", i, i*7919)
	}
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	s := newCCUUIDSet()
	for _, v := range ids {
		s.add(v)
	}
	runtime.GC()
	runtime.ReadMemStats(&after)
	per := float64(int64(after.HeapAlloc)-int64(before.HeapAlloc)) / n
	if !s.has(ids[n-1]) {
		t.Fatal("the last uuid is in the set")
	}
	// a 16-byte slot at a swiss table's load (7/16 to 7/8) is 19 to 39
	// bytes; a 48-byte string apiece beside a string's slot is over 67
	if per > 45 {
		t.Errorf("%.1f bytes held per uuid, want at most 45", per)
	}
	t.Logf("%.1f bytes held per uuid", per)
	runtime.KeepAlive(ids)
}
