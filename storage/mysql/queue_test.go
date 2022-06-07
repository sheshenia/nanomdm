//go:build integration
// +build integration

package mysql

import (
	"flag"
	"testing"

	"github.com/micromdm/nanomdm/storage/gensql"
	"github.com/micromdm/nanomdm/storage/internal/test"

	_ "github.com/go-sql-driver/mysql"
)

var flDSN = flag.String("dsn", "", "DSN of test MySQL instance")

func TestQueue(t *testing.T) {
	if *flDSN == "" {
		t.Fatal("MySQL DSN flag not provided to test")
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
