package runtime

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/risor-io/risor"
	"github.com/risor-io/risor/importer"
	"github.com/risor-io/risor/object"

	"github.com/jward/canopy/internal/store"
)

// Runtime embeds a Risor VM and provides tree-sitter host functions
// and Store access to extraction and resolution scripts.
type Runtime struct {
	store      store.DataStore
	scriptsDir string
	fsys       fs.FS
	sources    *sourceStore
}

// RuntimeOption configures a Runtime.
type RuntimeOption func(*Runtime)

// WithRuntimeFS configures the Runtime to load scripts from an fs.FS
// instead of from disk. Also configures the Risor importer to use
// FSImporter for import statement resolution.
func WithRuntimeFS(fsys fs.FS) RuntimeOption {
	return func(r *Runtime) {
		r.fsys = fsys
	}
}

// NewRuntime creates a Runtime wired to the given DataStore and scripts directory.
// Accepts optional RuntimeOptions for configuration such as fs.FS-based script loading.
func NewRuntime(s store.DataStore, scriptsDir string, opts ...RuntimeOption) *Runtime {
	r := &Runtime{
		store:      s,
		scriptsDir: scriptsDir,
		sources:    newSourceStore(),
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// RunScript loads and executes a Risor script with all standard globals
// plus any extra globals provided by the caller.
func (r *Runtime) RunScript(ctx context.Context, scriptPath string, extraGlobals map[string]any) error {
	src, err := r.LoadScript(scriptPath)
	if err != nil {
		return err
	}
	return r.eval(ctx, src, scriptPath, extraGlobals)
}

// RunSource executes Risor source code directly with all standard globals
// plus any extra globals. Useful for testing without script files.
func (r *Runtime) RunSource(ctx context.Context, source string, extraGlobals map[string]any) error {
	return r.eval(ctx, source, "<inline>", extraGlobals)
}

func (r *Runtime) eval(ctx context.Context, source, label string, extraGlobals map[string]any) error {
	globals := r.buildGlobals(extraGlobals)

	var opts []risor.Option
	for name, val := range globals {
		opts = append(opts, risor.WithGlobal(name, val))
	}

	// Wire importer so Risor import statements resolve correctly.
	if imp := r.buildImporter(globals); imp != nil {
		opts = append(opts, risor.WithImporter(imp))
	}

	_, err := risor.Eval(ctx, source, opts...)
	if err != nil {
		return fmt.Errorf("runtime: script %s: %w", label, err)
	}
	return nil
}

// buildImporter returns a Risor importer configured for the Runtime's script source.
// Returns nil if neither fs.FS nor scriptsDir is configured.
func (r *Runtime) buildImporter(globals map[string]any) importer.Importer {
	// Combine our custom global names with Risor's default globals (builtins
	// like len, string, print, and modules like strings, math, etc.) so that
	// imported modules can reference them without "undefined variable" errors.
	defaultNames := risor.NewConfig().GlobalNames()
	nameSet := make(map[string]bool, len(globals)+len(defaultNames))
	for name := range globals {
		nameSet[name] = true
	}
	for _, name := range defaultNames {
		nameSet[name] = true
	}
	globalNames := make([]string, 0, len(nameSet))
	for name := range nameSet {
		globalNames = append(globalNames, name)
	}

	if r.fsys != nil {
		return importer.NewFSImporter(importer.FSImporterOptions{
			GlobalNames: globalNames,
			SourceFS:    r.fsys,
			Extensions:  []string{".risor"},
		})
	}
	if r.scriptsDir != "" {
		return importer.NewLocalImporter(importer.LocalImporterOptions{
			GlobalNames: globalNames,
			SourceDir:   r.scriptsDir,
			Extensions:  []string{".risor"},
		})
	}
	return nil
}

// LoadScript reads a .risor file and returns its source code.
// When an fs.FS is configured, uses fs.ReadFile on the embedded filesystem.
// Otherwise, uses os.ReadFile with scriptsDir as the base directory.
func (r *Runtime) LoadScript(path string) (string, error) {
	if r.fsys != nil {
		// For fs.FS, strip any leading path separator so the path is
		// relative within the FS (e.g., "/extract/go.risor" -> "extract/go.risor").
		fsPath := strings.TrimPrefix(filepath.ToSlash(path), "/")
		data, err := fs.ReadFile(r.fsys, fsPath)
		if err != nil {
			return "", fmt.Errorf("runtime: loading script %s from fs: %w", fsPath, err)
		}
		return string(data), nil
	}

	fullPath := path
	if !filepath.IsAbs(path) {
		fullPath = filepath.Join(r.scriptsDir, path)
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("runtime: loading script %s: %w", fullPath, err)
	}
	return string(data), nil
}

// ExtractionScriptPath returns the path to a language's extraction script.
func ExtractionScriptPath(language string) string {
	return filepath.Join("extract", language+".risor")
}

// ResolutionScriptPath returns the path to a language's resolution script.
func ResolutionScriptPath(language string) string {
	return filepath.Join("resolve", language+".risor")
}

// buildGlobals constructs the full set of globals exposed to Risor scripts.
// Extraction globals work with any DataStore (including BatchedStore).
// Resolution globals require *store.Store and are only added when the
// underlying store is a real Store.
func (r *Runtime) buildGlobals(extra map[string]any) map[string]any {
	globals := map[string]any{
		"parse":      makeParseFn(r.sources),
		"parse_src":  makeParseSrcFn(r.sources),
		"node_text":  makeNodeTextFn(r.sources),
		"node_child": makeNodeChildFn(),
		"query":      makeQueryFn(r.sources),
		"log":        mustProxy(&logObject{prefix: "canopy"}),
	}

	// Expose extraction globals — these work with any DataStore.
	if r.store != nil {
		// Extraction insert functions
		globals["insert_symbol"] = makeInsertSymbolFn(r.store)
		globals["insert_scope"] = makeInsertScopeFn(r.store)
		globals["insert_reference"] = makeInsertReferenceFn(r.store)
		globals["insert_import"] = makeInsertImportFn(r.store)
		globals["insert_type_member"] = makeInsertTypeMemberFn(r.store)
		globals["insert_function_param"] = makeInsertFunctionParamFn(r.store)
		globals["insert_type_param"] = makeInsertTypeParamFn(r.store)
		globals["insert_annotation"] = makeInsertAnnotationFn(r.store)

		// Extraction query functions
		globals["symbols_by_name"] = makeSymbolsByNameFn(r.store)
		globals["symbols_by_file"] = makeSymbolsByFileFn(r.store)

		// Resolution globals require *store.Store (DB access, queries, etc.)
		if realStore, ok := r.store.(*store.Store); ok {
			globals["db"] = mustProxy(realStore)
			globals["update_annotation_resolved"] = makeUpdateAnnotationResolvedFn(realStore)

			// Resolution insert functions
			globals["insert_resolved_reference"] = makeInsertResolvedReferenceFn(realStore)
			globals["insert_implementation"] = makeInsertImplementationFn(realStore)
			globals["insert_call_edge"] = makeInsertCallEdgeFn(realStore)
			globals["insert_extension_binding"] = makeInsertExtensionBindingFn(realStore)

			// Resolution query functions
			globals["references_by_file"] = makeReferencesByFileFn(realStore)
			globals["scopes_by_file"] = makeScopesByFileFn(realStore)
			globals["imports_by_file"] = makeImportsByFileFn(realStore)
			globals["type_members"] = makeTypeMembersFn(realStore)
			globals["files_by_language"] = makeFilesByLanguageFn(realStore)
			globals["symbols_by_kind"] = makeSymbolsByKindFn(realStore)
			globals["scope_chain"] = makeScopeChainFn(realStore)
			globals["batch_scope_chains"] = makeBatchScopeChainsFn(realStore)
			globals["function_params"] = makeFunctionParamsFn(realStore)
			globals["db_query"] = makeDBQueryFn(realStore)
		}
	}

	for k, v := range extra {
		globals[k] = v
	}
	return globals
}

func mustProxy(v any) object.Object {
	p, err := object.NewProxy(v)
	if err != nil {
		panic(fmt.Sprintf("runtime: proxy error: %v", err))
	}
	return p
}
