package api

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const maxPhotoBytes = 5 << 20 // 5 MB; the Mini App shrinks photos before upload

// Only raster images are accepted. SVG is refused because it can carry scripts.
var photoExt = map[string]string{"image/jpeg": ".jpg", "image/png": ".png", "image/webp": ".webp", "image/gif": ".gif"}

// handleOwnerProductPhoto stores a new photo for one of the owner's products.
func (s *Server) handleOwnerProductPhoto(w http.ResponseWriter, r *http.Request) {
	owner := ownerFrom(r)
	product, err := s.Store.GetProduct(idParam(r, "id"), "en")
	if err == nil && product.ShopID != owner.ShopID {
		err = errNotYours
	}
	if err != nil {
		writeError(w, 404, "not found")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxPhotoBytes+64<<10)
	file, _, err := r.FormFile("photo")
	if err != nil {
		writeJSON(w, 400, apiError{Error: "send the photo in the form field \"photo\" (up to 5 MB)", Code: "BAD_PHOTO"})
		return
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxPhotoBytes+1))
	if err != nil || len(data) > maxPhotoBytes {
		writeJSON(w, 413, apiError{Error: "the photo is larger than 5 MB", Code: "PHOTO_TOO_BIG"})
		return
	}
	// Trust the file's bytes, not its name or the browser's content type.
	ext, ok := photoExt[http.DetectContentType(data)]
	if !ok {
		writeJSON(w, 415, apiError{Error: "use a JPEG, PNG or WebP photo", Code: "BAD_PHOTO_TYPE"})
		return
	}

	random := make([]byte, 6)
	_, _ = rand.Read(random)
	name := fmt.Sprintf("p%d-%s%s", product.ID, hex.EncodeToString(random), ext)
	if err := os.WriteFile(filepath.Join(s.Cfg.UploadDir, name), data, 0o644); err != nil {
		writeStoreError(w, err)
		return
	}
	if err := s.Store.SetProductPhoto(owner.ShopID, product.ID, "/uploads/"+name, ""); err != nil {
		writeStoreError(w, err)
		return
	}
	// The old uploaded file is not needed any more (seeded photos are part of the app and stay).
	if old, ok := strings.CutPrefix(product.Photo, "/uploads/"); ok {
		_ = os.Remove(filepath.Join(s.Cfg.UploadDir, filepath.Base(old)))
	}
	writeJSON(w, 200, map[string]string{"photo": "/uploads/" + name})
}

// serveUpload serves an uploaded photo. filepath.Base keeps requests inside the upload folder.
func (s *Server) serveUpload(w http.ResponseWriter, r *http.Request) {
	path := filepath.Join(s.Cfg.UploadDir, filepath.Base(r.URL.Path))
	if info, err := os.Stat(path); err != nil || info.IsDir() {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	http.ServeFile(w, r, path)
}
