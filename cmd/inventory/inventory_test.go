package inventory

import (
	"flag"
	"io"
	"os"
	"testing"

	"github.com/urfave/cli/v2"
)

// jsonlContext returns a CLI context whose --jsonl flag is set as given.
func jsonlContext(jsonl bool) *cli.Context {
	set := flag.NewFlagSet("test", flag.ContinueOnError)
	set.Bool("jsonl", jsonl, "")

	return cli.NewContext(cli.NewApp(), set, nil)
}

// captureOutput runs f and returns what it wrote to stdout and stderr.
func captureOutput(t *testing.T, f func()) (string, string) {
	t.Helper()

	outR, outW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	errR, errW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	stdout, stderr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = outW, errW

	f()

	os.Stdout, os.Stderr = stdout, stderr
	outW.Close()
	errW.Close()

	out, _ := io.ReadAll(outR)
	errOut, _ := io.ReadAll(errR)

	return string(out), string(errOut)
}

func TestReportMissing(t *testing.T) {
	missing := []string{"gid://shopify/InventoryTransfer/1"}
	argForGID := map[string]string{
		"gid://shopify/InventoryTransfer/1": "1",
		"gid://shopify/InventoryTransfer/2": "gid://shopify/InventoryTransfer/2",
	}

	t.Run("no missing ids", func(t *testing.T) {
		stdout, stderr := captureOutput(t, func() {
			reportMissing(jsonlContext(false), "Inventory Transfer", nil, argForGID)
		})

		if stdout != "" || stderr != "" {
			t.Errorf("expected no output, got stdout=%q stderr=%q", stdout, stderr)
		}
	})

	t.Run("plain prints the given argument", func(t *testing.T) {
		stdout, stderr := captureOutput(t, func() {
			reportMissing(jsonlContext(false), "Inventory Transfer", missing, argForGID)
		})

		if want := "Inventory Transfer(s) not found: 1\n"; stdout != want {
			t.Errorf("stdout = %q, want %q", stdout, want)
		}
		if stderr != "" {
			t.Errorf("stderr = %q, want empty", stderr)
		}
	})

	t.Run("jsonl prints to stderr", func(t *testing.T) {
		stdout, stderr := captureOutput(t, func() {
			reportMissing(jsonlContext(true), "Inventory Transfer", []string{"gid://shopify/InventoryTransfer/2"}, argForGID)
		})

		if stdout != "" {
			t.Errorf("stdout = %q, want empty", stdout)
		}
		if want := "Inventory Transfer(s) not found: gid://shopify/InventoryTransfer/2\n"; stderr != want {
			t.Errorf("stderr = %q, want %q", stderr, want)
		}
	})
}
