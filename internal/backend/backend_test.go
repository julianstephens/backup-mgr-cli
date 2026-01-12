package backend_test

import (
	"os"
	"testing"

	"github.com/julianstephens/warden/internal/backend"
	"github.com/julianstephens/warden/internal/backend/common"
)

func TestNewBackend(t *testing.T) {
	testDir := "/tmp/local"

	_, err := backend.NewBackend(common.LocalStorage, common.LocalStorageParams{Location: testDir})
	if err != nil {
		t.Fatal(err)
	}

	_, err = backend.NewBackend(1000, common.LocalStorageParams{Location: ""})
	if err == nil {
		t.Fatalf("expected invalid backend err, got nil")
	}

	os.RemoveAll(testDir)
}
