package http

type mediaResponse struct {
	Index    int    `json:"index"`
	MediaID  int64  `json:"mediaID"`
	MediaURL string `json:"mediaURL"`
}

type FileError struct {
	Index int    `json:"index"`
	Error string `json:"error"`
}

type fileResponse struct {
	Files       []mediaResponse `json:"media"`
	FilesErrors []FileError     `json:"errors"`
}

type urlResponse struct {
	MediaID  int64  `json:"mediaID"`
	MediaURL string `json:"mediaURL"`
}

type errorResponse struct {
	Error string `json:"error"`
}
