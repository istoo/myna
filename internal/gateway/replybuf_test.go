package gateway

import "testing"

// TestReplyBufSizeKeepsSmallRepliesSmall is the per-request allocation: a
// 170-byte JSON reply read through a 32 KB buffer is 32 KB allocated for
// every request answered whole.
func TestReplyBufSizeKeepsSmallRepliesSmall(t *testing.T) {
	for _, tc := range []struct {
		n    int64
		want int
	}{
		{-1, 32 << 10}, // unknown: a stream
		{0, 4 << 10},   // a reply that says it is empty
		{170, 4 << 10},
		{4 << 10, 4 << 10},
		{16 << 10, 16 << 10},
		{32 << 10, 32 << 10},
		{1 << 20, 32 << 10},
	} {
		if got := replyBufSize(tc.n); got != tc.want {
			t.Errorf("replyBufSize(%d) = %d, want %d", tc.n, got, tc.want)
		}
	}
}
