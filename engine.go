package canopy

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
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
		store:      s,
		scriptsDir: scriptsDir,
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
func (e *Engine) Store() *store.Store {
	return e.store
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

// IndexFiles indexes the given file paths. For each file:
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
	fileID, err := e.store.InsertFile(&store.File{
		Path:        path,
		Language:    lang,
		Hash:        hash,
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
// Skips hidden directories, node_modules, vendor, and __pycache__.
func (e *Engine) IndexDirectory(ctx context.Context, root string) error {
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
		return fmt.Errorf("walk directory: %w", err)
	}
	return e.IndexFiles(ctx, paths)
}

// Resolve runs resolution scripts for all languages that have indexed files.
// Resolution scripts are not incremental — they process all files for a language.
// To avoid duplicates, all existing resolution data for each language is deleted
// before running its script. The blast radius (from IndexFiles) determines which
// languages need re-resolution; if nil, all languages are resolved.
func (e *Engine) Resolve(ctx context.Context) error {
	// Reset blast radius after using it.
	defer func() { e.blastRadius = nil }()

	langs, err := e.distinctLanguages()
	if err != nil {
		return fmt.Errorf("list languages: %w", err)
	}

	var errs []error
	for _, lang := range langs {
		// Delete ALL resolution data for this language's files before re-running.
		// The resolution script processes all files, so partial deletion would
		// cause duplicates for files outside the blast radius.
		langFiles, err := e.store.FilesByLanguage(lang)
		if err != nil {
			errs = append(errs, fmt.Errorf("list files for %s: %w", lang, err))
			continue
		}
		var langFileIDs []int64
		for _, f := range langFiles {
			langFileIDs = append(langFileIDs, f.ID)
		}
		if len(langFileIDs) > 0 {
			if err := e.store.DeleteResolutionDataForFiles(langFileIDs); err != nil {
				errs = append(errs, fmt.Errorf("delete resolution data for %s: %w", lang, err))
				continue
			}
		}

		scriptPath := runtime.ResolutionScriptPath(lang)
		if err := e.runtime.RunScript(ctx, scriptPath, nil); err != nil {
			errs = append(errs, fmt.Errorf("resolution script for %s: %w", lang, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("resolution had %d error(s): %w", len(errs), errs[0])
	}
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
