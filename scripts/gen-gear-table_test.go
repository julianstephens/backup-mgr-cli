package scripts_test

import (
	"testing"

	"github.com/julianstephens/warden/scripts"
)

func TestRun(t *testing.T) {
	res := scripts.Run()

	if len(res) != 256 {
		t.Fatalf("expected table len 256, got %d", len(res))
	}
}
