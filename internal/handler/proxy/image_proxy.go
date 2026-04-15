package proxy

import (
	"io"
	"net/http"
)

// @Description		Get image via proxy
// @ID				image-proxy
// @Summary			Get image
// @Tags			proxy
// @Accept			json
// @Param			image_url	query	string	true	"image url"
// @Success			200
// @Failure			400	{object}	dto.CommonErrorResponse
// @Failure			502	{object}	dto.CommonErrorResponse
// @Router			/image-proxy [get]
func ImageProxy(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	if url == "" {
		http.Error(w, "missing url", http.StatusBadRequest)
		return
	}

	resp, err := http.Get(url)
	if err != nil {
		http.Error(w, "failed to fetch image", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.WriteHeader(http.StatusOK)
	io.Copy(w, resp.Body)
}
