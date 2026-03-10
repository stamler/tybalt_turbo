// Command testseed maintains the text fixture set used by tests.
//
// This command is intentionally separate from the application's top-level
// main.go. The root app/main.go starts the PocketBase server, while
// app/cmd/testseed/main.go is a small maintenance CLI for:
//   - dumping the current canonical test data to text fixtures
//   - loading a fresh PocketBase data directory from those fixtures
//   - verifying that the fixture set round-trips correctly
//
// Typical usage is via `go run ./cmd/testseed <subcommand>` from the app/
// directory.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"tybalt/internal/testseed"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "dump":
		if err := runDump(os.Args[2:]); err != nil {
			fatal(err)
		}
	case "load":
		if err := runLoad(os.Args[2:]); err != nil {
			fatal(err)
		}
	case "verify":
		if err := runVerify(os.Args[2:]); err != nil {
			fatal(err)
		}
	default:
		usage()
		os.Exit(1)
	}
}

// runDump exports the migrated runtime state of a PocketBase test data
// directory into the canonical text seed directory.
func runDump(args []string) error {
	fs := flag.NewFlagSet("dump", flag.ContinueOnError)
	dataDir := fs.String("data-dir", defaultDataDir(), "path to source pocketbase data directory")
	seedDir := fs.String("out", testseed.DefaultSeedDir(), "directory to write text seed data")
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return err
	}

	return testseed.DumpSeedDataFromTestApp(*dataDir, *seedDir)
}

// runLoad rebuilds a PocketBase data directory from migrations plus the text
// fixtures in the requested seed directory.
func runLoad(args []string) error {
	fs := flag.NewFlagSet("load", flag.ContinueOnError)
	dataDir := fs.String("out", "", "directory to build seeded pocketbase data into")
	seedDir := fs.String("seed-dir", testseed.DefaultSeedDir(), "directory containing text seed data")
	profile := fs.String("profile", testseed.TestFullProfile, "seed profile to load")
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *dataDir == "" {
		return fmt.Errorf("missing required --out")
	}

	if err := testseed.ValidateProfile(*profile); err != nil {
		return err
	}

	if err := resetGeneratedSQLiteFiles(*dataDir); err != nil {
		return err
	}

	return testseed.BuildSeededDataDirForProfile(*dataDir, *seedDir, *profile)
}

// runVerify confirms that the text fixture set reproduces the same migrated
// runtime data currently obtained from the source test data directory for the
// canonical test-full fixture profile.
func runVerify(args []string) error {
	fs := flag.NewFlagSet("verify", flag.ContinueOnError)
	dataDir := fs.String("data-dir", defaultDataDir(), "path to source pocketbase data directory")
	seedDir := fs.String("seed-dir", testseed.DefaultSeedDir(), "directory containing text seed data")
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return err
	}

	return testseed.VerifySeedDataAgainstTestApp(*dataDir, *seedDir)
}

// defaultDataDir returns the conventional local scratch PocketBase data
// directory used for manual app runs and dump/verify workflows.
func defaultDataDir() string {
	return filepath.Join(testseed.PackageRoot(), "test_pb_data")
}

// resetGeneratedSQLiteFiles removes the generated PocketBase SQLite artifacts
// from dataDir while preserving any non-generated files in the directory, such
// as checked-in documentation next to the local scratch DB.
func resetGeneratedSQLiteFiles(dataDir string) error {
	for _, name := range []string{
		"data.db",
		"data.db-shm",
		"data.db-wal",
		"auxiliary.db",
		"auxiliary.db-shm",
		"auxiliary.db-wal",
	} {
		if err := os.Remove(filepath.Join(dataDir, name)); err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	return nil
}

// usage prints the command synopsis for the testseed maintenance CLI.
func usage() {
	fmt.Fprintf(os.Stderr, "usage: %s <dump|load|verify> [flags]\n", filepath.Base(os.Args[0]))
}

// fatal prints err and exits with a non-zero status.
func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
