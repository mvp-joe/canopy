package store

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

// Store is the SQLite data access layer for canopy's 16 tables.
type Store struct {
	db *sql.DB
}

// NewStore opens a SQLite database at dbPath with WAL mode enabled.
func NewStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_foreign_keys=ON&_busy_timeout=30000")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return &Store{db: db}, nil
}

// Close closes the underlying database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// DB returns the underlying *sql.DB for use in transactions.
func (s *Store) DB() *sql.DB {
	return s.db
}

// Migrate creates all 16 tables and indexes. Idempotent.
func (s *Store) Migrate() error {
	_, err := s.db.Exec(schemaDDL)
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	return nil
}

const schemaDDL = `
-- Extraction tables

CREATE TABLE IF NOT EXISTS files (
  id              INTEGER PRIMARY KEY,
  path            TEXT NOT NULL UNIQUE,
  language        TEXT NOT NULL,
  hash            TEXT,
  last_indexed    TIMESTAMP
);

CREATE TABLE IF NOT EXISTS symbols (
  id              INTEGER PRIMARY KEY,
  file_id         INTEGER REFERENCES files(id),
  name            TEXT NOT NULL,
  kind            TEXT NOT NULL,
  visibility      TEXT,
  modifiers       TEXT,
  signature_hash  TEXT,
  start_line      INTEGER,
  start_col       INTEGER,
  end_line        INTEGER,
  end_col         INTEGER,
  parent_symbol_id INTEGER REFERENCES symbols(id)
);

CREATE TABLE IF NOT EXISTS symbol_fragments (
  id              INTEGER PRIMARY KEY,
  symbol_id       INTEGER NOT NULL REFERENCES symbols(id),
  file_id         INTEGER NOT NULL REFERENCES files(id),
  start_line      INTEGER,
  start_col       INTEGER,
  end_line        INTEGER,
  end_col         INTEGER,
  is_primary      BOOLEAN DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS scopes (
  id              INTEGER PRIMARY KEY,
  file_id         INTEGER NOT NULL REFERENCES files(id),
  symbol_id       INTEGER REFERENCES symbols(id),
  kind            TEXT NOT NULL,
  start_line      INTEGER,
  start_col       INTEGER,
  end_line        INTEGER,
  end_col         INTEGER,
  parent_scope_id INTEGER REFERENCES scopes(id)
);

CREATE TABLE IF NOT EXISTS references_ (
  id              INTEGER PRIMARY KEY,
  file_id         INTEGER NOT NULL REFERENCES files(id),
  scope_id        INTEGER REFERENCES scopes(id),
  name            TEXT NOT NULL,
  start_line      INTEGER,
  start_col       INTEGER,
  end_line        INTEGER,
  end_col         INTEGER,
  context         TEXT
);

CREATE TABLE IF NOT EXISTS imports (
  id              INTEGER PRIMARY KEY,
  file_id         INTEGER NOT NULL REFERENCES files(id),
  source          TEXT NOT NULL,
  imported_name   TEXT,
  local_alias     TEXT,
  kind            TEXT DEFAULT 'module',
  scope           TEXT DEFAULT 'file'
);

CREATE TABLE IF NOT EXISTS type_members (
  id              INTEGER PRIMARY KEY,
  symbol_id       INTEGER NOT NULL REFERENCES symbols(id),
  name            TEXT NOT NULL,
  kind            TEXT NOT NULL,
  type_expr       TEXT,
  visibility      TEXT
);

CREATE TABLE IF NOT EXISTS function_parameters (
  id              INTEGER PRIMARY KEY,
  symbol_id       INTEGER NOT NULL REFERENCES symbols(id),
  name            TEXT,
  ordinal         INTEGER NOT NULL,
  type_expr       TEXT,
  is_receiver     BOOLEAN DEFAULT FALSE,
  is_return       BOOLEAN DEFAULT FALSE,
  has_default     BOOLEAN DEFAULT FALSE,
  default_expr    TEXT
);

CREATE TABLE IF NOT EXISTS type_parameters (
  id              INTEGER PRIMARY KEY,
  symbol_id       INTEGER NOT NULL REFERENCES symbols(id),
  name            TEXT NOT NULL,
  ordinal         INTEGER NOT NULL,
  variance        TEXT,
  param_kind      TEXT DEFAULT 'type',
  constraints     TEXT
);

CREATE TABLE IF NOT EXISTS annotations (
  id              INTEGER PRIMARY KEY,
  target_symbol_id INTEGER NOT NULL REFERENCES symbols(id),
  name            TEXT NOT NULL,
  resolved_symbol_id INTEGER REFERENCES symbols(id),
  arguments       TEXT,
  file_id         INTEGER REFERENCES files(id),
  line            INTEGER,
  col             INTEGER
);

-- Resolution tables

CREATE TABLE IF NOT EXISTS resolved_references (
  id              INTEGER PRIMARY KEY,
  reference_id    INTEGER NOT NULL REFERENCES references_(id),
  target_symbol_id INTEGER NOT NULL REFERENCES symbols(id),
  confidence      REAL DEFAULT 1.0,
  resolution_kind TEXT
);

CREATE TABLE IF NOT EXISTS implementations (
  id              INTEGER PRIMARY KEY,
  type_symbol_id  INTEGER NOT NULL REFERENCES symbols(id),
  interface_symbol_id INTEGER NOT NULL REFERENCES symbols(id),
  kind            TEXT,
  file_id         INTEGER REFERENCES files(id),
  declaring_module TEXT
);

CREATE TABLE IF NOT EXISTS call_graph (
  id              INTEGER PRIMARY KEY,
  caller_symbol_id INTEGER NOT NULL REFERENCES symbols(id),
  callee_symbol_id INTEGER NOT NULL REFERENCES symbols(id),
  file_id         INTEGER REFERENCES files(id),
  line            INTEGER,
  col             INTEGER
);

CREATE TABLE IF NOT EXISTS reexports (
  id              INTEGER PRIMARY KEY,
  file_id         INTEGER NOT NULL REFERENCES files(id),
  original_symbol_id INTEGER NOT NULL REFERENCES symbols(id),
  exported_name   TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS extension_bindings (
  id              INTEGER PRIMARY KEY,
  member_symbol_id INTEGER NOT NULL REFERENCES symbols(id),
  extended_type_expr TEXT NOT NULL,
  extended_type_symbol_id INTEGER REFERENCES symbols(id),
  kind            TEXT DEFAULT 'method',
  constraints     TEXT,
  is_default_impl BOOLEAN DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS type_compositions (
  id              INTEGER PRIMARY KEY,
  composite_symbol_id INTEGER NOT NULL REFERENCES symbols(id),
  component_symbol_id INTEGER NOT NULL REFERENCES symbols(id),
  composition_kind TEXT NOT NULL
);

-- Indexes

CREATE INDEX IF NOT EXISTS idx_files_language ON files(language);
CREATE INDEX IF NOT EXISTS idx_symbols_file ON symbols(file_id);
CREATE INDEX IF NOT EXISTS idx_symbols_name ON symbols(name);
CREATE INDEX IF NOT EXISTS idx_symbols_kind ON symbols(kind);
CREATE INDEX IF NOT EXISTS idx_symbols_parent ON symbols(parent_symbol_id);
CREATE INDEX IF NOT EXISTS idx_symbols_hash ON symbols(signature_hash);
CREATE INDEX IF NOT EXISTS idx_scopes_file ON scopes(file_id);
CREATE INDEX IF NOT EXISTS idx_scopes_parent ON scopes(parent_scope_id);
CREATE INDEX IF NOT EXISTS idx_references_file ON references_(file_id);
CREATE INDEX IF NOT EXISTS idx_references_name ON references_(name);
CREATE INDEX IF NOT EXISTS idx_references_scope ON references_(scope_id);
CREATE INDEX IF NOT EXISTS idx_imports_file ON imports(file_id);
CREATE INDEX IF NOT EXISTS idx_imports_source ON imports(source);
CREATE INDEX IF NOT EXISTS idx_type_members_symbol ON type_members(symbol_id);
CREATE INDEX IF NOT EXISTS idx_function_params_symbol ON function_parameters(symbol_id);
CREATE INDEX IF NOT EXISTS idx_type_params_symbol ON type_parameters(symbol_id);
CREATE INDEX IF NOT EXISTS idx_annotations_target ON annotations(target_symbol_id);
CREATE INDEX IF NOT EXISTS idx_symbol_fragments_symbol ON symbol_fragments(symbol_id);
CREATE INDEX IF NOT EXISTS idx_resolved_refs_reference ON resolved_references(reference_id);
CREATE INDEX IF NOT EXISTS idx_resolved_refs_target ON resolved_references(target_symbol_id);
CREATE INDEX IF NOT EXISTS idx_implementations_type ON implementations(type_symbol_id);
CREATE INDEX IF NOT EXISTS idx_implementations_interface ON implementations(interface_symbol_id);
CREATE INDEX IF NOT EXISTS idx_call_graph_caller ON call_graph(caller_symbol_id);
CREATE INDEX IF NOT EXISTS idx_call_graph_callee ON call_graph(callee_symbol_id);
CREATE INDEX IF NOT EXISTS idx_reexports_file ON reexports(file_id);
CREATE INDEX IF NOT EXISTS idx_reexports_original ON reexports(original_symbol_id);
CREATE INDEX IF NOT EXISTS idx_extension_bindings_member ON extension_bindings(member_symbol_id);
CREATE INDEX IF NOT EXISTS idx_extension_bindings_type ON extension_bindings(extended_type_symbol_id);
CREATE INDEX IF NOT EXISTS idx_type_compositions_composite ON type_compositions(composite_symbol_id);
CREATE INDEX IF NOT EXISTS idx_type_compositions_component ON type_compositions(component_symbol_id);
`

// DeleteFileData transactionally removes all data for a file across all 16 tables.
// Deletes in reverse-dependency order to respect FK constraints.
func (s *Store) DeleteFileData(fileID int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get symbol IDs for this file (needed for child table cleanup).
	rows, err := tx.Query("SELECT id FROM symbols WHERE file_id = ?", fileID)
	if err != nil {
		return fmt.Errorf("query symbols: %w", err)
	}
	var symbolIDs []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return fmt.Errorf("scan symbol id: %w", err)
		}
		symbolIDs = append(symbolIDs, id)
	}
	rows.Close()

	// Get reference IDs for this file (needed for resolved_references cleanup).
	rows, err = tx.Query("SELECT id FROM references_ WHERE file_id = ?", fileID)
	if err != nil {
		return fmt.Errorf("query references: %w", err)
	}
	var refIDs []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return fmt.Errorf("scan reference id: %w", err)
		}
		refIDs = append(refIDs, id)
	}
	rows.Close()

	// Delete resolution tables referencing this file's symbols.
	if len(symbolIDs) > 0 {
		placeholders := placeholderList(len(symbolIDs))
		args := int64sToArgs(symbolIDs)

		for _, q := range []string{
			"DELETE FROM type_compositions WHERE composite_symbol_id IN (" + placeholders + ") OR component_symbol_id IN (" + placeholders + ")",
			"DELETE FROM extension_bindings WHERE member_symbol_id IN (" + placeholders + ") OR extended_type_symbol_id IN (" + placeholders + ")",
			"DELETE FROM reexports WHERE original_symbol_id IN (" + placeholders + ")",
			"DELETE FROM call_graph WHERE caller_symbol_id IN (" + placeholders + ") OR callee_symbol_id IN (" + placeholders + ")",
			"DELETE FROM implementations WHERE type_symbol_id IN (" + placeholders + ") OR interface_symbol_id IN (" + placeholders + ")",
			"DELETE FROM resolved_references WHERE target_symbol_id IN (" + placeholders + ")",
		} {
			// Count how many placeholder groups the query needs.
			expandedArgs := args
			count := countSubstring(q, "("+placeholders+")")
			if count > 1 {
				expandedArgs = repeatArgs(args, count)
			}
			if _, err := tx.Exec(q, expandedArgs...); err != nil {
				return fmt.Errorf("delete resolution data for symbols: %w", err)
			}
		}
	}

	// Delete resolved_references referencing this file's references.
	if len(refIDs) > 0 {
		placeholders := placeholderList(len(refIDs))
		args := int64sToArgs(refIDs)
		if _, err := tx.Exec("DELETE FROM resolved_references WHERE reference_id IN ("+placeholders+")", args...); err != nil {
			return fmt.Errorf("delete resolved references by ref: %w", err)
		}
	}

	// Delete resolution tables referencing this file directly.
	for _, q := range []string{
		"DELETE FROM reexports WHERE file_id = ?",
		"DELETE FROM call_graph WHERE file_id = ?",
		"DELETE FROM implementations WHERE file_id = ?",
	} {
		if _, err := tx.Exec(q, fileID); err != nil {
			return fmt.Errorf("delete resolution data for file: %w", err)
		}
	}

	// Delete extraction child tables for this file's symbols.
	if len(symbolIDs) > 0 {
		placeholders := placeholderList(len(symbolIDs))
		args := int64sToArgs(symbolIDs)
		for _, q := range []string{
			"DELETE FROM annotations WHERE target_symbol_id IN (" + placeholders + ")",
			"DELETE FROM type_parameters WHERE symbol_id IN (" + placeholders + ")",
			"DELETE FROM function_parameters WHERE symbol_id IN (" + placeholders + ")",
			"DELETE FROM type_members WHERE symbol_id IN (" + placeholders + ")",
			"DELETE FROM symbol_fragments WHERE symbol_id IN (" + placeholders + ")",
		} {
			if _, err := tx.Exec(q, args...); err != nil {
				return fmt.Errorf("delete extraction child data: %w", err)
			}
		}
	}

	// Delete symbol_fragments by file_id (for fragments from other symbols located in this file).
	if _, err := tx.Exec("DELETE FROM symbol_fragments WHERE file_id = ?", fileID); err != nil {
		return fmt.Errorf("delete symbol fragments by file: %w", err)
	}

	// Delete extraction tables for this file.
	for _, q := range []string{
		"DELETE FROM references_ WHERE file_id = ?",
		"DELETE FROM scopes WHERE file_id = ?",
		"DELETE FROM imports WHERE file_id = ?",
		"DELETE FROM symbols WHERE file_id = ?",
		"DELETE FROM annotations WHERE file_id = ?",
	} {
		if _, err := tx.Exec(q, fileID); err != nil {
			return fmt.Errorf("delete extraction data: %w", err)
		}
	}

	return tx.Commit()
}
