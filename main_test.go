package main

import "testing"

func TestRun_PropagatesCLIErrors(t *testing.T) {
	err := run([]string{"hcl-forge", "unknown"})
	if err == nil {
		t.Fatalf("expected run to return cli error")
	}
}
