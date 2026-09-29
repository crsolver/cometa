package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestDescribeCrashTranslatesIndexError(t *testing.T) {
	trace := "panic: runtime error: index out of range [5] with length 3\n\ngoroutine 1 [running]:\nmain.Main()\n\tC:/x/juego.cometa:12 +0x1d\n"
	got := describeCrash(trace)
	for _, want := range []string{"el índice 5 está fuera de rango (la lista tiene 3 elementos)", "juego.cometa, línea 12"} {
		if !strings.Contains(got, want) {
			t.Errorf("falta %q en %q", want, got)
		}
	}
}

func TestDescribeCrashTranslatesDivision(t *testing.T) {
	got := describeCrash("panic: runtime error: integer divide by zero\n[signal]\n")
	if !strings.Contains(got, "división entre cero") || !strings.Contains(got, "no se pudo determinar la línea") {
		t.Errorf("mensaje inesperado: %q", got)
	}
}

func TestCrashFilterForwardsOrdinaryStderr(t *testing.T) {
	var out bytes.Buffer
	filter := &crashFilter{out: &out}
	filter.Write([]byte("aviso\npanic: runtime er"))
	filter.Write([]byte("ror: integer divide by zero\n"))
	filter.Flush()
	if out.String() != "aviso\n" {
		t.Errorf("stderr reenviado = %q", out.String())
	}
	if !filter.crashed {
		t.Error("no se detectó el pánico")
	}
}

func TestBuildFailureExplainsNetworkAndGeneratedErrors(t *testing.T) {
	network := buildFailure("dial tcp: lookup proxy.golang.org: no such host", nil)
	if !strings.Contains(network.Error(), "conexión a internet") {
		t.Errorf("mensaje de red inesperado: %v", network)
	}
	internal := buildFailure("./x.cometa:3:1: undefined: foo", nil)
	if !strings.Contains(internal.Error(), "error interno del compilador") {
		t.Errorf("mensaje interno inesperado: %v", internal)
	}
}
