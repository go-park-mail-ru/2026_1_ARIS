package service

import "mime/multipart"

type SaveFileInput struct {
	Name      string
	Size      int64
	File      multipart.File
	FileFor   string
	ProfileID int64
}

type SavedFile struct {
	ID    int64
	URL   string
	Index int
}
