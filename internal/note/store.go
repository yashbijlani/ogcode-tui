package note

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/prasenjeet-symon/ogcode/internal/db"
	"github.com/prasenjeet-symon/ogcode/internal/id"
)

type Store struct {
	db *db.DB
}

func NewStore(database *db.DB) *Store {
	return &Store{db: database}
}

func (s *Store) Create(note *Note) error {
	src := note.Source
	if src == "" {
		src = SourceAI
	}
	_, err := s.db.Exec(
		`INSERT INTO note (id, directory, title, query, content, session_id, status, source, version, time_created, time_updated)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		note.ID, note.Directory, note.Title, note.Query, note.Content, note.SessionID, note.Status, src, note.Version, note.CreatedAt, note.UpdatedAt,
	)
	if err != nil {
		return err
	}
	note.Source = src
	return s.writeFile(note)
}

func (s *Store) Get(id string) (*Note, error) {
	row := s.db.QueryRow(
		`SELECT id, directory, title, query, content, session_id, status, source, version, time_created, time_updated
		 FROM note WHERE id = ?`, id,
	)
	return scanNote(row)
}

func (s *Store) GetBySessionID(sessionID string) (*Note, error) {
	row := s.db.QueryRow(
		`SELECT id, directory, title, query, content, session_id, status, source, version, time_created, time_updated
		 FROM note WHERE session_id = ?`, sessionID,
	)
	return scanNote(row)
}

func (s *Store) List(directory string) ([]*Note, error) {
	rows, err := s.db.Query(
		`SELECT id, directory, title, query, content, session_id, status, source, version, time_created, time_updated
		 FROM note WHERE directory = ? ORDER BY time_updated DESC`, directory,
	)
	if err != nil {
		return nil, fmt.Errorf("list notes: %w", err)
	}
	defer rows.Close()

	var notes []*Note
	for rows.Next() {
		n, err := scanNoteRow(rows)
		if err != nil {
			return nil, err
		}
		notes = append(notes, n)
	}
	return notes, nil
}

func (s *Store) UpdateContent(id, title, content, status string, version int) error {
	now := Now()
	_, err := s.db.Exec(
		`UPDATE note SET title = ?, content = ?, status = ?, version = ?, time_updated = ? WHERE id = ?`,
		title, content, status, version, now, id,
	)
	if err != nil {
		return err
	}
	n, err := s.Get(id)
	if err != nil || n == nil {
		return err
	}
	return s.writeFile(n)
}

// FinalizeBySession is called when the note agent loop completes. It increments
// the version, saves the new content to the note table, and records a version
// entry so the full history is preserved.
func (s *Store) FinalizeBySession(sessionID, content, exitReason string) error {
	note, err := s.GetBySessionID(sessionID)
	if err != nil || note == nil {
		return err
	}
	status := StatusDone
	if exitReason == "error" {
		status = StatusError
	}
	title := extractTitle(content, note.Query)
	nextVersion := note.Version + 1

	if err := s.UpdateContent(note.ID, title, content, status, nextVersion); err != nil {
		return err
	}

	// Record version snapshot
	v := &NoteVersion{
		ID:        id.NewNoteVersionID(),
		NoteID:    note.ID,
		Version:   nextVersion,
		Content:   content,
		CreatedAt: Now(),
	}
	return s.createVersion(v)
}

func (s *Store) createVersion(v *NoteVersion) error {
	_, err := s.db.Exec(
		`INSERT INTO note_version (id, note_id, version, content, created_at) VALUES (?, ?, ?, ?, ?)`,
		v.ID, v.NoteID, v.Version, v.Content, v.CreatedAt,
	)
	return err
}

func (s *Store) ListVersions(noteID string) ([]*NoteVersion, error) {
	rows, err := s.db.Query(
		`SELECT id, note_id, version, content, created_at FROM note_version WHERE note_id = ? ORDER BY version DESC`,
		noteID,
	)
	if err != nil {
		return nil, fmt.Errorf("list note versions: %w", err)
	}
	defer rows.Close()

	var versions []*NoteVersion
	for rows.Next() {
		var v NoteVersion
		if err := rows.Scan(&v.ID, &v.NoteID, &v.Version, &v.Content, &v.CreatedAt); err != nil {
			return nil, err
		}
		versions = append(versions, &v)
	}
	return versions, nil
}

// SaveContent updates a note's title and content, increments version, and records a version snapshot.
func (s *Store) SaveContent(noteID, title, content string) error {
	n, err := s.Get(noteID)
	if err != nil || n == nil {
		return fmt.Errorf("note not found: %w", err)
	}
	nextVersion := n.Version + 1
	if err := s.UpdateContent(noteID, title, content, StatusDone, nextVersion); err != nil {
		return err
	}
	v := &NoteVersion{
		ID:        id.NewNoteVersionID(),
		NoteID:    noteID,
		Version:   nextVersion,
		Content:   content,
		CreatedAt: Now(),
	}
	return s.createVersion(v)
}

// RecoverStuckNotes finds all notes still in "generating" status and marks
// them as "error". This runs on server startup to handle notes whose agent
// loop was interrupted by a server restart or crash.
func (s *Store) RecoverStuckNotes() ([]*Note, error) {
	rows, err := s.db.Query(
		`SELECT id, directory, title, query, content, session_id, status, source, version, time_created, time_updated
		 FROM note WHERE status = ?`, StatusGenerating,
	)
	if err != nil {
		return nil, fmt.Errorf("recover stuck notes: %w", err)
	}
	defer rows.Close()

	var stuck []*Note
	for rows.Next() {
		n, err := scanNoteRow(rows)
		if err != nil {
			return nil, err
		}
		stuck = append(stuck, n)
	}

	now := Now()
	for _, n := range stuck {
		title := n.Title
		if title == "" || strings.HasPrefix(title, "Note: ") {
			title = n.Query
			if len(title) > 60 {
				title = title[:60] + "…"
			}
		}
		content := n.Content
		if content == "" {
			content = "# " + title + "\n\n> Note generation was interrupted.\n\n**Query:** " + n.Query + "\n"
		}
		_, err := s.db.Exec(
			`UPDATE note SET status = ?, title = ?, content = ?, time_updated = ? WHERE id = ?`,
			StatusError, title, content, now, n.ID,
		)
		if err != nil {
			slog.Warn("recover stuck note: failed to update", "note", n.ID, "err", err)
			continue
		}
		n.Status = StatusError
		n.Title = title
		n.Content = content
		n.UpdatedAt = now
		_ = s.writeFile(n)
	}

	return stuck, nil
}

func (s *Store) Delete(id string) error {
	n, err := s.Get(id)
	if err == nil && n != nil {
		_ = s.removeFile(n)
	}
	_, err = s.db.Exec(`DELETE FROM note WHERE id = ?`, id)
	return err
}

func (s *Store) writeFile(n *Note) error {
	dir := filepath.Join(n.Directory, ".ogcode", "notes")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create notes dir: %w", err)
	}
	path := filepath.Join(dir, n.ID+".md")
	content := n.Content
	if content == "" {
		content = "# " + n.Title + "\n\n> Generating...\n\n**Query:** " + n.Query + "\n"
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func (s *Store) removeFile(n *Note) error {
	path := filepath.Join(n.Directory, ".ogcode", "notes", n.ID+".md")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func scanNote(row *sql.Row) (*Note, error) {
	var n Note
	var sessionID sql.NullString
	err := row.Scan(&n.ID, &n.Directory, &n.Title, &n.Query, &n.Content, &sessionID, &n.Status, &n.Source, &n.Version, &n.CreatedAt, &n.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get note: %w", err)
	}
	n.SessionID = sessionID.String
	if n.Source == "" {
		n.Source = SourceAI
	}
	return &n, nil
}

func scanNoteRow(rows *sql.Rows) (*Note, error) {
	var n Note
	var sessionID sql.NullString
	err := rows.Scan(&n.ID, &n.Directory, &n.Title, &n.Query, &n.Content, &sessionID, &n.Status, &n.Source, &n.Version, &n.CreatedAt, &n.UpdatedAt)
	if err != nil {
		return nil, err
	}
	n.SessionID = sessionID.String
	if n.Source == "" {
		n.Source = SourceAI
	}
	return &n, nil
}

func extractTitle(content, query string) string {
	for _, line := range strings.SplitN(content, "\n", 20) {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimPrefix(line, "# ")
		}
	}
	if len(query) > 60 {
		return query[:60] + "…"
	}
	return query
}
