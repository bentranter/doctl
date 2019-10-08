package e2e

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"text/template"
)

// binpath is the path to the binary's package.
const binpath string = "github.com/digitalocean/doctl/cmd/doctl"

// A TestMode is the mode in which to run the test suite.
type TestMode string

// Test modes.
const (
	// TestModeIntegration is the test mode for running integration tests.
	TestModeIntegration TestMode = "integration"

	// TestModeE2E is the test mode for running end to end tests.
	TestModeE2E TestMode = "end_to_end"
)

// Runner contains the information needed to run the test. suite
type Runner struct {
	// Bin is the name of the binary we're executing.
	Bin string

	// Tests are the tests we're going to execute.
	Tests []*Test

	// Mode is the mode to run the suite in.
	Mode TestMode
}

// NewRunner creates a new instance of a test suite.
//
// In will call the doctl binary by the given name, and start the test suite
// in the given test mode.
func NewRunner(bin string, mode TestMode) *Runner {
	return &Runner{Bin: bin, Mode: mode}
}

// A Test is an individual test case.
type Test struct {
	name string

	cmdtpl  string
	cmdargs []string

	in  []byte
	out string
}

// GenerateTestsFromPath generates test cases by walking the directory
// at the given path.
func (r *Runner) GenerateTestsFromPath(dir string) error {
	currentTest := &Test{}

	return filepath.Walk(dir, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if fi == nil || fi.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}

		var ext string
		if strings.Index(rel, ".") != -1 {
			ext = filepath.Ext(rel)
		}

		name := (rel[0 : len(rel)-len(ext)])

		switch ext {
		case ".json":
			in, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}
			currentTest.in = in

		case ".sh":
			cmdtpl, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}
			currentTest.cmdtpl = strings.TrimSpace(string(cmdtpl))

		case ".txt":
			out, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}
			currentTest.out = string(out)

			// Write out and reset the current test.
			currentTest.name = name
			r.Tests = append(r.Tests, currentTest)
			currentTest = &Test{}

		case ".go":
			// ok

		default:
			return fmt.Errorf(
				`files within the e2e directory can ONLY end in ".json", ".sh", ".txt", but found "%s" at %s/%s`,
				ext, rel, name)
		}

		return nil
	})
}

// TransformToExecutable transforms the tests with the runner to be executable
// test cases for the runner's current test mode.
func (r *Runner) TransformToExecutable() error {
	for i, test := range r.Tests {
		tpl, err := template.New("").Parse(test.cmdtpl)
		if err != nil {
			return err
		}

		buf := &bytes.Buffer{}
		if err := tpl.Execute(buf, map[string]string{
			"Bin": r.Bin,
		}); err != nil {
			return err
		}

		cmdstr := buf.String()
		cmdargs := strings.Split(cmdstr, " ")
		if len(cmdargs) == 0 {
			return errors.New("cmd requires at least one argument")
		}

		r.Tests[i].cmdargs = cmdargs
	}

	return nil
}

// Run executes the test suite.
func (r *Runner) Run(t *testing.T) {
	for _, test := range r.Tests {
		t.Run(test.name, func(t *testing.T) {
			if r.Mode == TestModeIntegration {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Write(test.in)
				}))
				defer server.Close()

				test.cmdargs = append(test.cmdargs, "-u", server.URL)
			}

			cmd := exec.Command(test.cmdargs[0], test.cmdargs[1:]...)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatal(err)
			}

			expected := strings.TrimSpace(test.out)
			actual := strings.TrimSpace(string(output))

			if expected != actual {
				t.Fatalf("expected %s but got %s", expected, actual)
			}
		})
	}
}
