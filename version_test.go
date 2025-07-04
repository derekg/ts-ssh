package main

import (
	"fmt"
	"testing"
)

func TestVersionVariable(t *testing.T) {
	fmt.Printf("Version variable contains: '%s'\n", version)
	if version == "" {
		t.Error("Version variable should not be empty")
	}
}
