package canopy

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"

	canopyrt "github.com/jward/canopy/internal/runtime"
	"github.com/jward/canopy/internal/store"
)

// workItem holds everything a parallel extraction worker needs.
type workItem struct {
	path   string
	lang   string
	fileID int64
	batch  *store.BatchedStore

	// Pre-captured old symbols for blast radius computation after commit.
	oldSymbols []capturedSymbol
}

// IndexFilesParallel indexes files using a three-phase parallel pipeline:
//
//	Phase A (serial):  Hash check, delete old data, prepare file records.
//	Phase B (parallel): Parse and extract via worker pool (each with own Runtime).
//	Phase C (serial):  Commit batches to SQLite, compute blast radius.
func (e *Engine) IndexFilesParallel(ctx context.Context, paths []string) error {
	if e.blastRadius == nil {
		e.blastRadius = make(map[int64]bool)
	}

	// ---- Phase A: Serial file preparation ----
	var items []workItem
	for _, path := range paths {
		item, skip, err := e.prepareFile(ctx, path)
		if err != nil {
			return fmt.Errorf("prepare %s: %w", path, err)
		}
		if skip {
			continue
		}
		items = append(items, item)
	}

	if len(items) == 0 {
		return nil
	}

	// ---- Phase B: Parallel extraction ----
	numWorkers := min(runtime.NumCPU(), len(items))
	if numWorkers < 1 {
		numWorkers = 1
	}

	workCh := make(chan workItem, len(items))
	for _, item := range items {
		workCh <- item
	}
	close(workCh)

	type result struct {
		item workItem
		err  error
	}
	resultCh := make(chan result, len(items))

	var wg sync.WaitGroup
	for range numWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Each worker gets its own Runtime with a fresh sourceStore.
			// The BatchedStore per item handles write isolation.
			for item := range workCh {
				err := e.extractFile(ctx, item)
				resultCh <- result{item: item, err: err}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// ---- Phase C: Serial commit ----
	var errs []error
	for res := range resultCh {
		if res.err != nil {
			errs = append(errs, fmt.Errorf("extract %s: %w", res.item.path, res.err))
			continue
		}

		if err := e.store.CommitBatch(res.item.batch); err != nil {
			errs = append(errs, fmt.Errorf("commit %s: %w", res.item.path, err))
			continue
		}

		// Capture new symbols and compute blast radius (now that data is committed).
		newSymbols, err := e.captureSymbols(res.item.fileID)
		if err != nil {
			errs = append(errs, fmt.Errorf("capture new symbols %s: %w", res.item.path, err))
			continue
		}

		blastFileIDs := e.computeBlastRadius(res.item.fileID, res.item.oldSymbols, newSymbols)
		for _, fid := range blastFileIDs {
			e.blastRadius[fid] = true
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("parallel indexing had %d error(s): %w", len(errs), errs[0])
	}
	return nil
}

// prepareFile does Phase A work for a single file: hash check, cleanup, file record.
// Returns (item, skip, error). skip=true means the file is unchanged or unsupported.
func (e *Engine) prepareFile(_ context.Context, path string) (workItem, bool, error) {
	lang, ok := canopyrt.LanguageForFile(path)
	if !ok {
		return workItem{}, true, nil
	}
	if e.languages != nil && !e.languages[lang] {
		return workItem{}, true, nil
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return workItem{}, false, fmt.Errorf("read file: %w", err)
	}
	hash := fmt.Sprintf("%x", sha256.Sum256(content))

	existing, err := e.store.FileByPath(path)
	if err != nil {
		return workItem{}, false, fmt.Errorf("lookup file: %w", err)
	}
	if existing != nil && existing.Hash == hash {
		return workItem{}, true, nil // unchanged
	}

	// Capture old symbols before deletion (for blast radius).
	var oldSymbols []capturedSymbol
	if existing != nil {
		oldSymbols, err = e.captureSymbols(existing.ID)
		if err != nil {
			return workItem{}, false, fmt.Errorf("capture old symbols: %w", err)
		}
	}

	// Clean up old data.
	if existing != nil {
		if err := e.store.DeleteFileData(existing.ID); err != nil {
			return workItem{}, false, fmt.Errorf("delete old data: %w", err)
		}
		if _, err := e.store.DB().Exec("DELETE FROM files WHERE id = ?", existing.ID); err != nil {
			return workItem{}, false, fmt.Errorf("delete file record: %w", err)
		}
	}

	// Insert new file record (real ID assigned by SQLite).
	fileID, err := e.store.InsertFile(&store.File{
		Path:        path,
		Language:    lang,
		Hash:        hash,
		LastIndexed: time.Now(),
	})
	if err != nil {
		return workItem{}, false, fmt.Errorf("insert file: %w", err)
	}

	batch := store.NewBatchedStore(e.store)
	return workItem{
		path:       path,
		lang:       lang,
		fileID:     fileID,
		batch:      batch,
		oldSymbols: oldSymbols,
	}, false, nil
}

// extractFile runs the extraction script for a single file using a BatchedStore.
// Each call creates its own Runtime so tree-sitter parsing is goroutine-safe.
func (e *Engine) extractFile(ctx context.Context, item workItem) error {
	// Create a per-worker Runtime backed by the BatchedStore.
	var rtOpts []canopyrt.RuntimeOption
	if e.scriptsFS != nil {
		rtOpts = append(rtOpts, canopyrt.WithRuntimeFS(e.scriptsFS))
	}
	rt := canopyrt.NewRuntime(item.batch, e.scriptsDir, rtOpts...)

	scriptPath := canopyrt.ExtractionScriptPath(item.lang)
	extras := map[string]any{
		"file_path": item.path,
		"file_id":   item.fileID,
	}
	if err := rt.RunScript(ctx, scriptPath, extras); err != nil {
		return fmt.Errorf("extraction script: %w", err)
	}
	return nil
}
