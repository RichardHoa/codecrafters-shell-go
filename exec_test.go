package main

import (
	"os"
	"os/exec"
	"testing"

	"golang.org/x/sys/unix"
)

// Path to a common binary on most Unix systems
const testPath = "/bin/ls"
const testCmd = "ls"

// Benchmark unix.Access (The "Raw" Syscall)
func BenchmarkUnixAccess(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = unix.Access(testPath, unix.X_OK)
	}
}

func BenchmarkOsStat(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = os.Stat(testPath)
	}
}

// Benchmark exec.LookPath with an Absolute Path
// This skips the $PATH search but still does internal Go overhead
func BenchmarkLookPathAbs(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = exec.LookPath(testPath)
	}
}

// Benchmark exec.LookPath with a Command Name
// This forces a full $PATH crawl
func BenchmarkLookPathSearch(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = exec.LookPath(testCmd)
	}
}
