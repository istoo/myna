package provider

import (
	"bytes"
	"context"
	"strings"

	"github.com/istoo/owl/internal/proc"
)

// keychainPut adds or replaces a generic password without putting the
// secret in the process arguments, where another local user's ps could
// read it: with -w left out, security asks for it twice on stdin. A
// secret with a line break can't be typed that way, so it takes -w.
func keychainPut(ctx context.Context, service, account string, secret []byte) error {
	if account != "" && !bytes.ContainsRune(secret, '\n') {
		in := string(secret) + "\n" + string(secret) + "\n"
		cmd := proc.CommandContext(ctx, "security", "add-generic-password", "-U", "-s", service, "-a", account, "-w")
		cmd.Stdin = strings.NewReader(in)
		if err := cmd.Run(); err == nil {
			return nil
		}
	}
	args := []string{"add-generic-password", "-U", "-s", service}
	if account != "" {
		args = append(args, "-a", account)
	}
	args = append(args, "-w", string(secret))
	return proc.CommandContext(ctx, "security", args...).Run()
}
