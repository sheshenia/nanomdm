//go:build integration
// +build integration

package pgsql

import (
	"flag"
	"testing"

	"github.com/micromdm/nanomdm/storage/gensql"
	"github.com/micromdm/nanomdm/storage/internal/test"

	_ "github.com/lib/pq"
)

var flDSN = flag.String("dsn", "", "DSN of test PostgreSQL instance")

func TestQueue(t *testing.T) {
	if *flDSN == "" {
		t.Fatal("PostgreSQL DSN flag not provided to test")
	}

	storage, err := New(gensql.WithDSN(*flDSN), gensql.WithDeleteCommands())
	if err != nil {
		t.Fatal(err)
	}

	err = test.EnrollTestDevice(storage)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("WithDeleteCommands()", func(t *testing.T) {
		test.TestQueue(t, test.DeviceUDID, storage)
	})

	storage, err = New(gensql.WithDSN(*flDSN))
	if err != nil {
		t.Fatal(err)
	}

	t.Run("normal", func(t *testing.T) {
		test.TestQueue(t, test.DeviceUDID, storage)
	})
}
