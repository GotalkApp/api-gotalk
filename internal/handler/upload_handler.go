package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/quocanhngo/gotalk/internal/model"
	"github.com/quocanhngo/gotalk/pkg/storage"
)

// Max upload size: 50MB
const maxUploadSize = 50 << 20

// Allowed MIME types
var allowedImageTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/gif":  true,
	"image/webp": true,
}

var allowedVideoTypes = map[string]bool{
	"video/mp4":       true,
	"video/webm":      true,
	"video/quicktime": true,
}

var allowedFileTypes = map[string]bool{
	"application/pdf":    true,
	"application/msword": true,
	"application/zip":    true,
	"audio/mpeg":         true,
	"audio/ogg":          true,
	"audio/wav":          true,
}

// UploadHandler handles file upload endpoints
type UploadHandler struct {
	storage *storage.MinIOStorage
}

// NewUploadHandler creates a new upload handler
func NewUploadHandler(storage *storage.MinIOStorage) *UploadHandler {
	return &UploadHandler{storage: storage}
}

// UploadFile godoc
// @Summary Upload a file (image, video, or document)
// @Description Upload a file to storage. Returns the public URL. Supports images (jpg, png, gif, webp), videos (mp4, webm, mov), and documents (pdf, doc, zip).
// @Tags Upload
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param file formance file true "File to upload"
// @Param type formData string false "File type hint: image, video, file" Enums(image, video, file)
// @Success 200 {object} model.UploadResponse
// @Failure 400 {object} model.ErrorResponse
// @Failure 413 {object} model.ErrorResponse
// @Router /upload [post]
func (h *UploadHandler) UploadFile(c *gin.Context) {
	// Limit request body size
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadSize)

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		if err.Error() == "http: request body too large" {
			c.JSON(http.StatusRequestEntityTooLarge, model.ErrorResponse{Error: "File too large (max 50MB)"})
			return
		}
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "File is required", Message: err.Error()})
		return
	}
	defer file.Close()

	// Detect and validate content type
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "Unable to detect file type"})
		return
	}

	// Determine folder based on content type
	folder := determineFolder(contentType)
	if folder == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error:   "Unsupported file type",
			Message: "Allowed: jpg, png, gif, webp, mp4, webm, mov, pdf, doc, zip, mp3, ogg, wav",
		})
		return
	}

	// Upload to MinIO
	result, err := h.storage.Upload(c.Request.Context(), file, header, folder)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: "Failed to upload file", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.UploadResponse{
		URL:      result.URL,
		FileName: result.FileName,
		FileSize: result.FileSize,
		MimeType: result.MimeType,
	})
}

// UploadMultiple godoc
// @Summary Upload multiple files
// @Description Upload up to 10 files at once. Returns array of URLs.
// @Tags Upload
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param files formData file true "Files to upload (max 10)"
// @Success 200 {array} model.UploadResponse
// @Failure 400 {object} model.ErrorResponse
// @Router /upload/multiple [post]
func (h *UploadHandler) UploadMultiple(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadSize)

	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "Invalid form data", Message: err.Error()})
		return
	}

	files := form.File["files"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "No files provided"})
		return
	}

	if len(files) > 10 {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "Maximum 10 files allowed"})
		return
	}

	results := []model.UploadResponse{}
	for _, header := range files {
		file, err := header.Open()
		if err != nil {
			continue
		}

		contentType := header.Header.Get("Content-Type")
		folder := determineFolder(contentType)
		if folder == "" {
			file.Close()
			continue // Skip unsupported files
		}

		result, err := h.storage.Upload(c.Request.Context(), file, header, folder)
		file.Close()
		if err != nil {
			continue // Skip failed uploads
		}

		results = append(results, model.UploadResponse{
			URL:      result.URL,
			FileName: result.FileName,
			FileSize: result.FileSize,
			MimeType: result.MimeType,
		})
	}

	c.JSON(http.StatusOK, results)
}

// determineFolder returns the storage folder based on content type
func determineFolder(contentType string) string {
	ct := strings.ToLower(contentType)

	if allowedImageTypes[ct] {
		return "images"
	}
	if allowedVideoTypes[ct] {
		return "videos"
	}
	if allowedFileTypes[ct] {
		if strings.HasPrefix(ct, "audio/") {
			return "audio"
		}
		return "files"
	}
	return "" // unsupported
}
