package main

// Export is the top-level AI Pro chat export document.
type Export struct {
	Version int            `json:"version"`
	History []Conversation `json:"history"`
	Folders []Folder       `json:"folders"`
	Files   []FileBlob     `json:"files"`
}

// Conversation is one chat thread with its messages and attachment references.
// FolderID is any because the export uses null for unfiled conversations.
type Conversation struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Messages []Message `json:"messages"`
	Files    []FileRef `json:"files"`
	FolderID any       `json:"folderId"`
}

// Folder maps an export folder ID to its display name.
type Folder struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// Message is a single chat turn and the attachments referenced by it.
type Message struct {
	Role    string    `json:"role"`
	Content string    `json:"content"`
	Files   []FileRef `json:"files"`
}

// FileRef points at an attachment blob by key. FileSize is any because the
// export mixes human-readable strings and numbers.
type FileRef struct {
	FileKey  string `json:"fileKey"`
	FileName string `json:"fileName"`
	FileSize any    `json:"fileSize"`
}

// FileBlob is a top-level attachment payload, stored either as text Content
// or as an ImageBase64 string (optionally a data URI).
type FileBlob struct {
	FileKey        string `json:"fileKey"`
	FileName       string `json:"fileName"`
	FileSize       any    `json:"fileSize"`
	ConversationID string `json:"conversationId"`
	Content        string `json:"content"`
	FileType       any    `json:"fileType"`
	ImageBase64    any    `json:"imageBase64"`
}
