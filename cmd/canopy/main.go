package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jward/canopy"
	"github.com/jward/canopy/scripts"
	"github.com/spf13/cobra"
)

var (
	flagDB     string
	flagFormat string
)

// errorHandled is set by outputError so main() doesn't double-print.
var errorHandled bool

func main() {
	if err := rootCmd.Execute(); err != nil {
		if !errorHandled {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		}
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:           "canopy",
	Short:         "Deterministic, scope-aware semantic code analysis",
	Long:          "Canopy indexes source code using tree-sitter and Risor scripts, producing a SQLite database for semantic queries.",
	SilenceErrors: true,
	SilenceUsage:  true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return validateFormat(flagFormat)
	},
	// No Run — prints help by default.
}

func init() {
	rootCmd.PersistentFlags().StringVar(&flagDB, "db", "", "database path (default: .canopy/index.db relative to repo root)")
	rootCmd.PersistentFlags().StringVar(&flagFormat, "format", "json", "output format: json|text")

	rootCmd.AddCommand(indexCmd)
	rootCmd.AddCommand(queryCmd)
}

var (
	flagForce      bool
	flagLanguages  string
	flagScriptsDir string
)

var indexCmd = &cobra.Command{
	Use:   "index [path]",
	Short: "Index a repository for semantic analysis",
	Long:  "Parses source files with tree-sitter, runs extraction and resolution scripts, and writes results to the SQLite database.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runIndex,
}

func init() {
	indexCmd.Flags().BoolVar(&flagForce, "force", false, "delete database and reindex from scratch")
	indexCmd.Flags().StringVar(&flagLanguages, "languages", "", "comma-separated language filter (e.g. go,typescript)")
	indexCmd.Flags().StringVar(&flagScriptsDir, "scripts-dir", "", "load scripts from disk path instead of embedded")
}

func runIndex(cmd *cobra.Command, args []string) error {
	start := time.Now()

	// Determine the target directory.
	targetDir, err := resolveTargetDir(args)
	if err != nil {
		return err
	}

	// Resolve repo root and DB path.
	repoRoot := findRepoRoot(targetDir)
	dbPath := resolveDBPath(repoRoot)

	// Ensure .canopy/ directory exists.
	canopyDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(canopyDir, 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", canopyDir, err)
	}

	// Handle --force: delete the DB file entirely.
	if flagForce {
		if err := os.Remove(dbPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("removing database for --force: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Cleared database: %s\n", dbPath)
	}

	// Build engine options.
	var opts []canopy.Option
	if flagLanguages != "" {
		langs := strings.Split(flagLanguages, ",")
		for i := range langs {
			langs[i] = strings.TrimSpace(langs[i])
		}
		opts = append(opts, canopy.WithLanguages(langs...))
	}

	// Script source: --scripts-dir overrides embedded FS.
	scriptsDir := flagScriptsDir
	if scriptsDir == "" {
		opts = append(opts, canopy.WithScriptsFS(scripts.FS))
	}

	engine, err := canopy.New(dbPath, scriptsDir, opts...)
	if err != nil {
		return fmt.Errorf("creating engine: %w", err)
	}
	defer engine.Close()

	ctx := context.Background()

	// Run extraction.
	extractStart := time.Now()
	if err := engine.IndexDirectory(ctx, targetDir); err != nil {
		return fmt.Errorf("indexing: %w", err)
	}
	extractDuration := time.Since(extractStart)

	// Run resolution.
	resolveStart := time.Now()
	if err := engine.Resolve(ctx); err != nil {
		return fmt.Errorf("resolving: %w", err)
	}
	resolveDuration := time.Since(resolveStart)

	totalDuration := time.Since(start)

	// Print timing summary to stderr.
	fmt.Fprintf(os.Stderr, "Indexed %s in %s (extract: %s, resolve: %s)\n",
		targetDir,
		totalDuration.Round(time.Millisecond),
		extractDuration.Round(time.Millisecond),
		resolveDuration.Round(time.Millisecond),
	)
	fmt.Fprintf(os.Stderr, "Database: %s\n", dbPath)

	return nil
}

// resolveTargetDir returns the absolute path of the directory to index.
func resolveTargetDir(args []string) (string, error) {
	dir := "."
	if len(args) > 0 {
		dir = args[0]
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("resolving path %q: %w", dir, err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("directory not found: %s", abs)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("not a directory: %s", abs)
	}
	return abs, nil
}

// findRepoRoot walks up from startDir looking for a .git directory.
// Returns the directory containing .git, or startDir if not found.
func findRepoRoot(startDir string) string {
	dir := startDir
	for {
		if info, err := os.Stat(filepath.Join(dir, ".git")); err == nil && info.IsDir() {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root without finding .git.
			return startDir
		}
		dir = parent
	}
}

// resolveDBPath returns the database path from the --db flag or the default.
func resolveDBPath(repoRoot string) string {
	if flagDB != "" {
		if filepath.IsAbs(flagDB) {
			return flagDB
		}
		return filepath.Join(repoRoot, flagDB)
	}
	return filepath.Join(repoRoot, ".canopy", "index.db")
}
