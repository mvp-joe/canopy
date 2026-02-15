package canopy

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// findModuleRootB is the benchmark equivalent of findModuleRoot.
func findModuleRootB(b *testing.B) string {
	b.Helper()
	dir, err := os.Getwd()
	if err != nil {
		b.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			b.Fatal("could not find module root")
		}
		dir = parent
	}
}

// benchGoSource is a realistic ~100-line Go file with functions, structs,
// interfaces, and method calls for exercising the full extraction pipeline.
const benchGoSource = `package bench

import (
	"fmt"
	"strings"
)

// Logger defines a logging interface.
type Logger interface {
	Log(msg string)
	Logf(format string, args ...interface{})
}

// Config holds application configuration.
type Config struct {
	Name    string
	Debug   bool
	MaxRetry int
	Tags    []string
}

// Validate checks the config for correctness.
func (c *Config) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("name is required")
	}
	if c.MaxRetry < 0 {
		return fmt.Errorf("max_retry must be non-negative")
	}
	return nil
}

// String returns a human-readable representation.
func (c *Config) String() string {
	return fmt.Sprintf("Config{Name: %s, Debug: %v}", c.Name, c.Debug)
}

// HasTag reports whether the config includes the given tag.
func (c *Config) HasTag(tag string) bool {
	for _, t := range c.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// StdoutLogger implements Logger by writing to stdout.
type StdoutLogger struct {
	Prefix string
}

// Log writes a plain message.
func (l *StdoutLogger) Log(msg string) {
	fmt.Printf("[%s] %s\n", l.Prefix, msg)
}

// Logf writes a formatted message.
func (l *StdoutLogger) Logf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	l.Log(msg)
}

// NewApp creates and returns an initialized App.
func NewApp(cfg *Config, log Logger) *App {
	return &App{config: cfg, logger: log}
}

// App is the main application struct.
type App struct {
	config *Config
	logger Logger
}

// Run starts the application.
func (a *App) Run() error {
	if err := a.config.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}
	a.logger.Logf("starting %s", a.config.Name)
	a.process()
	return nil
}

// process does the main work.
func (a *App) process() {
	tags := strings.Join(a.config.Tags, ", ")
	a.logger.Logf("processing with tags: %s", tags)
}

// BuildGreeting constructs a greeting string.
func BuildGreeting(name string) string {
	return fmt.Sprintf("Hello, %s!", name)
}

// CountWords returns the number of words in s.
func CountWords(s string) int {
	return len(strings.Fields(s))
}
`

// setupBenchEngine creates an Engine and a Go source file, returning the engine,
// file path, and temp dir. Caller must close the engine.
func setupBenchEngine(b *testing.B) (*Engine, string) {
	b.Helper()
	dir := b.TempDir()
	dbPath := filepath.Join(dir, "bench.db")
	modRoot := findModuleRootB(b)
	scriptsDir := filepath.Join(modRoot, "scripts")

	e, err := New(dbPath, scriptsDir, WithLanguages("go"))
	if err != nil {
		b.Fatal(err)
	}

	srcPath := filepath.Join(dir, "bench.go")
	if err := os.WriteFile(srcPath, []byte(benchGoSource), 0644); err != nil {
		e.Close()
		b.Fatal(err)
	}

	return e, srcPath
}

// BenchmarkIndexFiles_Go measures the time to extract semantic data from a
// realistic Go source file (~100 lines with structs, interfaces, methods).
func BenchmarkIndexFiles_Go(b *testing.B) {
	ctx := context.Background()
	modRoot := findModuleRootB(b)
	scriptsDir := filepath.Join(modRoot, "scripts")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		dir := b.TempDir()
		dbPath := filepath.Join(dir, "bench.db")
		e, err := New(dbPath, scriptsDir, WithLanguages("go"))
		if err != nil {
			b.Fatal(err)
		}
		srcPath := filepath.Join(dir, "bench.go")
		if err := os.WriteFile(srcPath, []byte(benchGoSource), 0644); err != nil {
			e.Close()
			b.Fatal(err)
		}
		b.StartTimer()

		if err := e.IndexFiles(ctx, []string{srcPath}); err != nil {
			e.Close()
			b.Fatal(err)
		}

		b.StopTimer()
		e.Close()
		b.StartTimer()
	}
}

// BenchmarkResolve_Go measures the time to resolve cross-file references
// after extraction, using a pre-indexed Go source file.
func BenchmarkResolve_Go(b *testing.B) {
	e, srcPath := setupBenchEngine(b)
	defer e.Close()
	ctx := context.Background()

	// Index once as setup.
	if err := e.IndexFiles(ctx, []string{srcPath}); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := e.Resolve(ctx); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkQueryDefinitionAt measures the time to perform a DefinitionAt
// query after a full index+resolve cycle. This benchmarks the query path only.
func BenchmarkQueryDefinitionAt(b *testing.B) {
	e, srcPath := setupBenchEngine(b)
	defer e.Close()
	ctx := context.Background()

	if err := e.IndexFiles(ctx, []string{srcPath}); err != nil {
		b.Fatal(err)
	}
	if err := e.Resolve(ctx); err != nil {
		b.Fatal(err)
	}

	q := e.Query()

	// Query the "Validate" method call inside App.Run — line 89 in benchGoSource:
	//   if err := a.config.Validate(); err != nil {
	// "Validate" starts at column ~23 (after "a.config.").
	// Use a reference we know exists: the Logger interface usage.
	// Line 73: func NewApp(cfg *Config, log Logger) *App {
	// "Config" reference at col ~22, "Logger" at col ~35.

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Query DefinitionAt for "Logger" on the NewApp line.
		// Even if the exact position doesn't find a resolved reference,
		// the benchmark exercises the full query path (file lookup, reference scan, resolution).
		_, err := q.DefinitionAt(srcPath, 73, 35)
		if err != nil {
			b.Fatal(err)
		}
	}
}
