//go:build !windows

package main

func enableVTProcessing() {}

func vtAvailable() bool { return true }
