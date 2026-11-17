// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"
	"strings"
	"testing"
)

// TestConnectProjectValidationBeforeSetup points RI_HOST at a closed port:
// if a tool connected before validating its input, the error would be a
// connection failure instead of the validation message.
func TestConnectProjectValidationBeforeSetup(t *testing.T) {
	t.Setenv("RI_HOST", "http://127.0.0.1:1")
	t.Setenv("RI_USER", mockUsername)
	t.Setenv("RI_PASSWORD", mockPassword)

	checks := []struct {
		name        string
		call        func() error
		errContains string
	}{
		{"create, blank name", func() error {
			_, _, err := CreateConnectProject(context.Background(), newReq(), CreateConnectProjectInput{Name: " "})
			return err
		}, "name is required"},
		{"update, blank name", func() error {
			_, _, err := UpdateConnectProject(context.Background(), newReq(), UpdateConnectProjectInput{Name: " ", Description: strPtr("x")})
			return err
		}, "name is required"},
		{"update, nothing to change", func() error {
			_, _, err := UpdateConnectProject(context.Background(), newReq(), UpdateConnectProjectInput{Name: "zzz1"})
			return err
		}, "nothing to update"},
		{"delete, blank name", func() error {
			_, _, err := DeleteConnectProject(context.Background(), newReq(), DeleteConnectProjectInput{Name: " "})
			return err
		}, "name is required"},
	}

	for _, c := range checks {
		t.Run(c.name, func(t *testing.T) {
			err := c.call()
			if err == nil || !strings.Contains(err.Error(), c.errContains) {
				t.Fatalf("expected an error containing %q, got %v", c.errContains, err)
			}
		})
	}
}
