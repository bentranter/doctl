package e2e_test

import (
	"log"
	"os"
	"testing"

	"github.com/digitalocean/doctl/e2e"
)

func TestE2E(t *testing.T) {
	mode := e2e.TestModeIntegration
	if testmode := os.Getenv("TESTMODE"); testmode != "" {
		switch testmode {
		case "e2e", "end-to-end", string(e2e.TestModeE2E):
			mode = e2e.TestModeE2E
		}
	}

	log.Println("Started in test mode", mode)

	// TODO Use binary like in current integration tests.
	runner := e2e.NewRunner("doctl", mode)
	if err := runner.GenerateTestsFromPath(
		"/Users/ben/go/src/github.com/digitalocean/doctl/e2e"); err != nil {
		t.Fatal(err)
	}

	if err := runner.TransformToExecutable(); err != nil {
		t.Fatal(err)
	}

	runner.Run(t)
}
