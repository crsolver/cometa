package codegen

import (
	"os"
	"testing"
)

// Every program compiled by these tests also has its generated Go
// type-checked (see codegen.SelfCheck).
func TestMain(m *testing.M) {
	SelfCheck = true
	os.Exit(m.Run())
}
