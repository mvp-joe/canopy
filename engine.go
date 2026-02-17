package canopy

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jward/canopy/internal/runtime"
	"github.com/jward/canopy/internal/store"
)

// Engine orchestrates the canopy pipeline: file discovery, change detection,
// extraction via Risor scripts, resolution, and query access.
type Engine struct {
	store      *store.Store
	runtime    *runtime.Runtime
	scriptsDir string
	scriptsFS  fs.FS
	languages  map[string]bool // nil means all languages

	// blastRadius accumulates file IDs that need re-resolution after indexing.
	// nil means "resolve everything" (first run or full reindex).
	blastRadius map[int64]bool

	// useParallel enables the parallel extraction pipeline.
	useParallel bool
}

// Option configures an Engine.
type Option func(*Engine)

// WithLanguages restricts which languages the Engine will process.
func WithLanguages(languages ...string) Option {
	return func(e *Engine) {
		e.languages = make(map[string]bool, len(languages))
		for _, lang := range languages {
			e.languages[lang] = true
		}
	}
}

// WithParallel controls parallel extraction. When true (default), IndexFiles
// uses a worker pool for parsing and script execution, with a single writer
// goroutine committing batches to SQLite. Set to false for serial mode.
func WithParallel(parallel bool) Option {
	return func(e *Engine) {
		e.useParallel = parallel
	}
}

// WithScriptsFS configures the Engine to load Risor scripts from the given
// filesystem instead of from the scriptsDir path on disk. This enables
// embedding scripts via go:embed. When set, scriptsDir is ignored for
// script loading but may still be used as a label in error messages.
func WithScriptsFS(fsys fs.FS) Option {
	return func(e *Engine) {
		e.scriptsFS = fsys
	}
}

// New creates an Engine backed by a SQLite database at dbPath.
// Script loading priority:
//  1. If WithScriptsFS is set, use the provided fs.FS
//  2. Otherwise, use scriptsDir on disk
//
// The scriptsDir parameter may be empty when WithScriptsFS is used.
func New(dbPath string, scriptsDir string, opts ...Option) (*Engine, error) {
	s, err := store.NewStore(dbPath)
	if err != nil {
		return nil, fmt.Errorf("canopy: create store: %w", err)
	}
	if err := s.Migrate(); err != nil {
		s.Close()
		return nil, fmt.Errorf("canopy: migrate: %w", err)
	}

	// Apply options to a temporary Engine to collect configuration before
	// creating the Runtime, since the Runtime needs to know about fs.FS.
	e := &Engine{
		store:       s,
		scriptsDir:  scriptsDir,
		useParallel: true, // default to parallel extraction
	}
	for _, opt := range opts {
		opt(e)
	}

	// Build Runtime with the appropriate script source.
	var rtOpts []runtime.RuntimeOption
	if e.scriptsFS != nil {
		rtOpts = append(rtOpts, runtime.WithRuntimeFS(e.scriptsFS))
	}
	e.runtime = runtime.NewRuntime(s, scriptsDir, rtOpts...)

	return e, nil
}

// Close releases the Engine's database resources.
func (e *Engine) Close() error {
	return e.store.Close()
}

// Store returns the underlying Store for direct access.
func (e *Engine) Store() *Store {
	return e.store
}

// scriptsHash computes a SHA-256 hash of all Risor scripts (extract, resolve, lib).
// Walks the scriptsFS or scriptsDir to find all .risor files, sorts them by path,
// and hashes their concatenated contents. Returns hex-encoded hash string.
func (e *Engine) scriptsHash() string {
	var paths []string

	if e.scriptsFS != nil {
		fs.WalkDir(e.scriptsFS, ".", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if !d.IsDir() && strings.HasSuffix(path, ".risor") {
				paths = append(paths, path)
			}
			return nil
		})
	} else if e.scriptsDir != "" {
		filepath.WalkDir(e.scriptsDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if !d.IsDir() && strings.HasSuffix(path, ".risor") {
				rel, _ := filepath.Rel(e.scriptsDir, path)
				paths = append(paths, rel)
			}
			return nil
		})
	}

	sort.Strings(paths)

	h := sha256.New()
	for _, p := range paths {
		src, err := e.runtime.LoadScript(p)
		if err != nil {
			continue
		}
		h.Write([]byte(p))
		h.Write([]byte(src))
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

// ScriptsChanged reports whether the embedded scripts differ from what was
// used to build the current database. Returns true if the DB has no stored
// hash (first run) or if the hash doesn't match. When true, the caller
// should delete the DB and reindex from scratch.
func (e *Engine) ScriptsChanged() bool {
	current := e.scriptsHash()
	stored, err := e.store.GetMetadata("scripts_hash")
	if err != nil || stored == "" {
		return true
	}
	return current != stored
}

// storeScriptsHash persists the current scripts hash to the database.
func (e *Engine) storeScriptsHash() {
	_ = e.store.SetMetadata("scripts_hash", e.scriptsHash())
}

// Query returns a new QueryBuilder wrapping the Store.
func (e *Engine) Query() *QueryBuilder {
	return &QueryBuilder{store: e.store}
}

// symbolKey uniquely identifies a symbol by (name, kind, parent_symbol_id).
type symbolKey struct {
	Name           string
	Kind           string
	ParentSymbolID int64 // 0 means no parent
}

// capturedSymbol holds a symbol's identity and hash for blast radius comparison.
type capturedSymbol struct {
	ID            int64
	Key           symbolKey
	SignatureHash string
}

// captureSymbols captures the current symbols for a file, including their computed
// signature hashes.
func (e *Engine) captureSymbols(fileID int64) ([]capturedSymbol, error) {
	syms, err := e.store.SymbolsByFile(fileID)
	if err != nil {
		return nil, err
	}

	var captured []capturedSymbol
	for _, sym := range syms {
		members, err := e.store.TypeMembers(sym.ID)
		if err != nil {
			return nil, err
		}
		params, err := e.store.FunctionParams(sym.ID)
		if err != nil {
			return nil, err
		}
		typeParams, err := e.store.TypeParams(sym.ID)
		if err != nil {
			return nil, err
		}
		hash := store.ComputeSignatureHash(
			sym.Name, sym.Kind, sym.Visibility, sym.Modifiers,
			members, params, typeParams,
		)

		var parentID int64
		if sym.ParentSymbolID != nil {
			parentID = *sym.ParentSymbolID
		}

		captured = append(captured, capturedSymbol{
			ID: sym.ID,
			Key: symbolKey{
				Name:           sym.Name,
				Kind:           sym.Kind,
				ParentSymbolID: parentID,
			},
			SignatureHash: hash,
		})
	}
	return captured, nil
}

// IndexFiles indexes the given file paths. When WithParallel is enabled,
// uses a worker pool for concurrent extraction with batched SQLite writes.
// Otherwise falls back to the serial path.
//
// For each file:
// 1. Detect language from extension
// 2. Skip unsupported or filtered-out languages
// 3. Skip unchanged files (same content hash)
// 4. Capture old symbols (for blast radius)
// 5. Delete stale data, insert/update file record
// 6. Run the language's extraction script
// 7. Capture new symbols, compute blast radius
//
// Errors on individual files are logged and skipped; processing continues.
func (e *Engine) IndexFiles(ctx context.Context, paths []string) error {
	if e.useParallel {
		return e.IndexFilesParallel(ctx, paths)
	}
	return e.indexFilesSerial(ctx, paths)
}

func (e *Engine) indexFilesSerial(ctx context.Context, paths []string) error {
	// Initialize blast radius so Resolve() can distinguish "no changes"
	// (non-nil empty map) from "first run" (nil).
	if e.blastRadius == nil {
		e.blastRadius = make(map[int64]bool)
	}
	var errs []error
	for _, path := range paths {
		if err := e.indexFile(ctx, path); err != nil {
			errs = append(errs, fmt.Errorf("index %s: %w", path, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("indexing had %d error(s): %w", len(errs), errs[0])
	}
	return nil
}

func (e *Engine) indexFile(ctx context.Context, path string) error {
	lang, ok := runtime.LanguageForFile(path)
	if !ok {
		return nil // unsupported extension
	}
	if e.languages != nil && !e.languages[lang] {
		return nil // filtered out
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}
	hash := fmt.Sprintf("%x", sha256.Sum256(content))

	existing, err := e.store.FileByPath(path)
	if err != nil {
		return fmt.Errorf("lookup file: %w", err)
	}
	if existing != nil && existing.Hash == hash {
		return nil // unchanged
	}

	// Step 1: Capture old symbols before deletion (for blast radius).
	var oldSymbols []capturedSymbol
	if existing != nil {
		oldSymbols, err = e.captureSymbols(existing.ID)
		if err != nil {
			return fmt.Errorf("capture old symbols: %w", err)
		}
	}

	// Step 2: Clean up old data if the file was previously indexed.
	if existing != nil {
		if err := e.store.DeleteFileData(existing.ID); err != nil {
			return fmt.Errorf("delete old data: %w", err)
		}
		if _, err := e.store.DB().Exec("DELETE FROM files WHERE id = ?", existing.ID); err != nil {
			return fmt.Errorf("delete file record: %w", err)
		}
	}

	// Step 3: Insert new file record and run extraction.
	lineCount := bytes.Count(content, []byte{'\n'}) + 1
	fileID, err := e.store.InsertFile(&store.File{
		Path:        path,
		Language:    lang,
		Hash:        hash,
		LineCount:   lineCount,
		LastIndexed: time.Now(),
	})
	if err != nil {
		return fmt.Errorf("insert file: %w", err)
	}

	scriptPath := runtime.ExtractionScriptPath(lang)
	extras := map[string]any{
		"file_path": path,
		"file_id":   fileID,
	}
	if err := e.runtime.RunScript(ctx, scriptPath, extras); err != nil {
		return fmt.Errorf("extraction script: %w", err)
	}

	// Step 4: Capture new symbols and compute blast radius.
	newSymbols, err := e.captureSymbols(fileID)
	if err != nil {
		return fmt.Errorf("capture new symbols: %w", err)
	}

	blastFileIDs := e.computeBlastRadius(fileID, oldSymbols, newSymbols)

	// Add to accumulated blast radius.
	if e.blastRadius == nil {
		e.blastRadius = make(map[int64]bool)
	}
	for _, fid := range blastFileIDs {
		e.blastRadius[fid] = true
	}

	return nil
}

// computeBlastRadius compares old vs new symbols and returns file IDs that need
// re-resolution.
func (e *Engine) computeBlastRadius(fileID int64, oldSyms, newSyms []capturedSymbol) []int64 {
	// Always include the changed file itself.
	result := map[int64]bool{fileID: true}

	// Build maps by key.
	oldByKey := make(map[symbolKey]capturedSymbol, len(oldSyms))
	for _, s := range oldSyms {
		oldByKey[s.Key] = s
	}
	newByKey := make(map[symbolKey]capturedSymbol, len(newSyms))
	for _, s := range newSyms {
		newByKey[s.Key] = s
	}

	// Classify symbols.
	var removedIDs, changedOldIDs []int64
	hasAdded := false
	hasRemoved := false

	for key, oldSym := range oldByKey {
		if newSym, ok := newByKey[key]; ok {
			if oldSym.SignatureHash != newSym.SignatureHash {
				// Changed: signature differs.
				changedOldIDs = append(changedOldIDs, oldSym.ID)
			}
			// else: unchanged
		} else {
			// Removed: no matching new symbol.
			removedIDs = append(removedIDs, oldSym.ID)
			hasRemoved = true
		}
	}
	for key := range newByKey {
		if _, ok := oldByKey[key]; !ok {
			hasAdded = true
		}
	}

	// Add files with resolved_references targeting removed/changed symbols.
	affectedIDs := append(removedIDs, changedOldIDs...)
	if len(affectedIDs) > 0 {
		fileIDs, err := e.store.FilesReferencingSymbols(affectedIDs)
		if err == nil {
			for _, fid := range fileIDs {
				result[fid] = true
			}
		}
	}

	// If symbols were added or removed, add files that import this file's module/package.
	if hasAdded || hasRemoved {
		// Find all imports that reference this file's package.
		// We search by both the bare package name (for same-directory imports)
		// and by import paths ending with the package name (for full paths).
		for _, s := range newSyms {
			if s.Key.Kind == "package" {
				pkgName := s.Key.Name
				// Direct match (bare name, e.g., tests or same-dir).
				importers, err := e.store.FilesImportingSource(pkgName)
				if err == nil {
					for _, fid := range importers {
						result[fid] = true
					}
				}
				// Also match imports whose source ends with /pkgName.
				rows, err := e.store.DB().Query(
					`SELECT DISTINCT file_id FROM imports WHERE source LIKE ?`,
					"%/"+pkgName,
				)
				if err == nil {
					for rows.Next() {
						var fid int64
						if err := rows.Scan(&fid); err == nil {
							result[fid] = true
						}
					}
					rows.Close()
				}
				break
			}
		}
	}

	// Delete stale resolution data for removed symbols.
	if len(removedIDs) > 0 {
		_ = e.store.DeleteResolutionDataForSymbols(removedIDs)
	}

	var fileIDs []int64
	for fid := range result {
		fileIDs = append(fileIDs, fid)
	}
	return fileIDs
}

// skipDir returns true for directories that should be excluded from indexing.
var skipDirs = map[string]bool{
	"node_modules": true,
	"vendor":       true,
	"__pycache__":  true,
}

// IndexDirectory walks root and indexes all files with supported extensions.
// If root is inside a git repository, uses git ls-files to respect .gitignore.
// Falls back to filesystem walk (skipping hidden dirs, node_modules, vendor,
// __pycache__) if git is unavailable.
func (e *Engine) IndexDirectory(ctx context.Context, root string) error {
	paths, err := e.gitListFiles(root)
	if err != nil {
		// Not a git repo or git not available — fall back to walk.
		paths, err = e.walkListFiles(root)
		if err != nil {
			return err
		}
	}
	return e.IndexFiles(ctx, paths)
}

// gitListFiles uses git ls-files to discover tracked and untracked (but not
// ignored) files under root, filtered to supported languages.
func (e *Engine) gitListFiles(root string) ([]string, error) {
	// --cached: tracked files, --others: untracked files,
	// --exclude-standard: respect .gitignore, .git/info/exclude, global excludes.
	cmd := exec.Command("git", "ls-files", "--cached", "--others", "--exclude-standard")
	cmd.Dir = root
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git ls-files: %w", err)
	}

	var paths []string
	for _, line := range strings.Split(stdout.String(), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		absPath := filepath.Join(root, line)
		if _, ok := runtime.LanguageForFile(absPath); ok {
			paths = append(paths, absPath)
		}
	}
	return paths, nil
}

// walkListFiles discovers files by walking the filesystem, used as a fallback
// when git is not available. Skips hidden directories, node_modules, vendor,
// and __pycache__.
func (e *Engine) walkListFiles(root string) ([]string, error) {
	var paths []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if strings.HasPrefix(name, ".") || skipDirs[name] {
				return filepath.SkipDir
			}
			return nil
		}
		if _, ok := runtime.LanguageForFile(path); ok {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk directory: %w", err)
	}
	return paths, nil
}

// Resolve runs resolution scripts for all languages that have indexed files.
// Resolution scripts are not incremental — they process all files for a language.
// Resolve runs language-specific resolution scripts against extraction data.
//
// Resolution scripts receive a files_to_resolve function that returns only
// the files needing resolution, while files_by_language continues to return
// all files (needed for cross-file lookup caches).
func (e *Engine) Resolve(ctx context.Context) error {
	defer func() { e.blastRadius = nil }()

	// Non-nil empty blast radius means no files changed — skip resolution.
	if e.blastRadius != nil && len(e.blastRadius) == 0 {
		return nil
	}

	langs, err := e.distinctLanguages()
	if err != nil {
		return fmt.Errorf("list languages: %w", err)
	}

	// Delete resolution data for affected files before re-running scripts.
	if e.blastRadius != nil {
		// Incremental: only delete resolution data for blast radius files.
		var blastIDs []int64
		for fid := range e.blastRadius {
			blastIDs = append(blastIDs, fid)
		}
		if len(blastIDs) > 0 {
			if err := e.store.DeleteResolutionDataForFiles(blastIDs); err != nil {
				return fmt.Errorf("delete resolution data: %w", err)
			}
		}
	} else {
		// Full resolve: delete all resolution data for every language.
		for _, lang := range langs {
			langFiles, err := e.store.FilesByLanguage(lang)
			if err != nil {
				return fmt.Errorf("list files for %s: %w", lang, err)
			}
			var langFileIDs []int64
			for _, f := range langFiles {
				langFileIDs = append(langFileIDs, f.ID)
			}
			if len(langFileIDs) > 0 {
				if err := e.store.DeleteResolutionDataForFiles(langFileIDs); err != nil {
					return fmt.Errorf("delete resolution data for %s: %w", lang, err)
				}
			}
		}
	}

	// Pass files_to_resolve as an extra global. It filters FilesByLanguage
	// to only files in the blast radius (or returns all files on full resolve).
	extras := map[string]any{
		"files_to_resolve": runtime.MakeFilesToResolveFn(e.store, e.blastRadius),
	}

	// Run resolution scripts in parallel (one per language).
	var (
		mu   sync.Mutex
		wg   sync.WaitGroup
		errs []error
	)
	for _, lang := range langs {
		wg.Add(1)
		go func(l string) {
			defer wg.Done()
			scriptPath := runtime.ResolutionScriptPath(l)
			if err := e.runtime.RunScript(ctx, scriptPath, extras); err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("resolution script for %s: %w", l, err))
				mu.Unlock()
			}
		}(lang)
	}
	wg.Wait()

	if len(errs) > 0 {
		return fmt.Errorf("resolution had %d error(s): %w", len(errs), errs[0])
	}

	// Store the current scripts hash so future runs can detect changes.
	e.storeScriptsHash()

	return nil
}

// distinctLanguages returns all languages that have at least one file in the Store.
func (e *Engine) distinctLanguages() ([]string, error) {
	rows, err := e.store.DB().Query("SELECT DISTINCT language FROM files")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var langs []string
	for rows.Next() {
		var lang string
		if err := rows.Scan(&lang); err != nil {
			return nil, err
		}
		langs = append(langs, lang)
	}
	return langs, rows.Err()
}
