package usecase

import "mime/multipart"

type SaveFileInput struct {
	Name      string
	Size      int64
	File      multipart.File
	FileFor   string
	ProfileID int64
}

type SaveFilesInput struct {
	FileHeaders   []*multipart.FileHeader
	FileFor       string
	UserAccountID int64
}

type SavedFile struct {
	ID    int64
	URL   string
	Index int
}

type FileError struct {
	Index int
	Error string
}
