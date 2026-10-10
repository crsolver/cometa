package main

import (
	"os"
	"testing"

	"github.com/crsolver/cometa/internal/codegen"
)

// Every program compiled by these tests also has its generated Go
// type-checked (see codegen.SelfCheck).
func TestMain(m *testing.M) {
	codegen.SelfCheck = true
	os.Exit(m.Run())
}
