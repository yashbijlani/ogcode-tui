package note

import "time"

const (
	StatusGenerating = "generating"
	StatusDone       = "done"
	StatusError      = "error"

	SourceAI     = "ai"
	SourceManual = "manual"
)

type Note struct {
	ID        string `json:"id"`
	Directory string `json:"directory"`
	Title     string `json:"title"`
	Query     string `json:"query"`
	Content   string `json:"content"`
	SessionID string `json:"sessionId,omitempty"`
	Status    string `json:"status"`
	Source    string `json:"source"`
	Version   int    `json:"version"`
	CreatedAt int64  `json:"createdAt"`
	UpdatedAt int64  `json:"updatedAt"`
}

type NoteVersion struct {
	ID        string `json:"id"`
	NoteID    string `json:"noteId"`
	Version   int    `json:"version"`
	Content   string `json:"content"`
	CreatedAt int64  `json:"createdAt"`
}

func Now() int64 {
	return time.Now().UnixMilli()
}
