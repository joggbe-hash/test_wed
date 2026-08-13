package main

import (
	"os"
	"strings"
	"testing"
)

func TestNginxStrictLoginLimiterIncludesOwnershipVerification(t *testing.T) {
	paths := []string{
		"../nginx/templates/conf.d/default.conf.template",
		"../nginx/templates/conf.d/default.dev.conf.template",
		"../nginx/templates/conf/server.conf.template",
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read nginx template: %v", err)
			}
			if !strings.Contains(string(source), `login(?:/verify|/ownership/verify)?`) {
				t.Fatal("strict login limiter does not include ownership verification")
			}
		})
	}
}
