package handlers

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/tmjpugh/househero/internal/database"
)

type UploadHandler struct {
	db        *database.DB
	uploadDir string
}

type UploadResponse struct {
	ID        int64  `json:"id"`
	URL       string `json:"url"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Size      int64  `json:"size"`
	UploadedAt string `json:"uploaded_at"`
}

type Base64UploadRequest struct {
	Base64   string `json:"base64"`
	Name     string `json:"name"`
	FileType string `json:"file_type"` // e.g., "photo", "document", "receipt", "manual"
}

func NewUploadHandler(db *database.DB, uploadDir string) *UploadHandler {
	// Create upload directory and subdirectories
	os.MkdirAll(filepath.Join(uploadDir, "photos"), os.ModePerm)
	os.MkdirAll(filepath.Join(uploadDir, "documents"), os.ModePerm)
	os.MkdirAll(filepath.Join(uploadDir, "receipts"), os.ModePerm)
	os.MkdirAll(filepath.Join(uploadDir, "manuals"), os.ModePerm)
	
	return &UploadHandler{
		db:        db,
		uploadDir: uploadDir,
	}
}

// UploadTicketPhoto - multipart form file upload for ticket photos
func (h *UploadHandler) UploadTicketPhoto(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ticketID := vars["id"]

	h.handleMultipartUpload(w, r, ticketID, "photos", "photo", "photos")
}

// UploadTicketDocument - multipart form file upload for ticket documents
func (h *UploadHandler) UploadTicketDocument(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ticketID := vars["id"]

	h.handleMultipartUpload(w, r, ticketID, "documents", "document", "documents")
}

// UploadInventoryReceipt - multipart form file upload for inventory receipts
func (h *UploadHandler) UploadInventoryReceipt(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	itemID := vars["id"]

	h.handleMultipartUpload(w, r, itemID, "receipts", "receipt", "receipts")
}

// UploadInventoryManual - multipart form file upload for inventory manuals
func (h *UploadHandler) UploadInventoryManual(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	itemID := vars["id"]

	h.handleMultipartUpload(w, r, itemID, "manuals", "manual", "manuals")
}

// handleMultipartUpload - generic multipart form handler
func (h *UploadHandler) handleMultipartUpload(w http.ResponseWriter, r *http.Request, parentID string, subdir string, formField string, uploadType string) {
	if err := r.ParseMultipartForm(100 * 1024 * 1024); err != nil { // 100MB max
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	file, fileHeader, err := r.FormFile(formField)
	if err != nil {
		http.Error(w, "No file provided", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Validate file type based on upload type
	if !h.isValidFileType(fileHeader.Filename, uploadType) {
		http.Error(w, "Invalid file type", http.StatusBadRequest)
		return
	}

	// Generate unique filename
	filename := strconv.FormatInt(time.Now().UnixNano(), 10) + "-" + sanitizeFilename(fileHeader.Filename)
	uploadPath := filepath.Join(h.uploadDir, subdir)
	filepath := filepath.Join(uploadPath, filename)

	// Save file to disk
	dst, err := os.Create(filepath)
	if err != nil {
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	written, err := io.Copy(dst, file)
	if err != nil {
		http.Error(w, "Failed to copy file", http.StatusInternalServerError)
		return
	}

	// Get display name
	displayName := r.FormValue("name")
	if displayName == "" {
		displayName = fileHeader.Filename
	}

	// Prepare response
	response := UploadResponse{
		URL:  "/uploads/" + subdir + "/" + filename,
		Name: displayName,
		Type: uploadType,
		Size: written,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// UploadBase64 - generic base64 upload handler for photos/documents
func (h *UploadHandler) UploadBase64(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	parentID := vars["id"]
	fileType := mux.Vars(r)["type"] // "photo", "document", "receipt", "manual"

	var payload Base64UploadRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Decode base64
	data, err := base64.StdEncoding.DecodeString(payload.Base64)
	if err != nil {
		http.Error(w, "Invalid base64", http.StatusBadRequest)
		return
	}

	// Determine file extension and subdirectory
	ext := h.getFileExtension(payload.Base64, payload.Name)
	subdir := h.getSubdirectory(fileType)

	filename := strconv.FormatInt(time.Now().UnixNano(), 10) + ext
	uploadPath := filepath.Join(h.uploadDir, subdir)
	filepath := filepath.Join(uploadPath, filename)

	// Save file
	if err := os.WriteFile(filepath, data, 0644); err != nil {
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}

	// Prepare response
	response := UploadResponse{
		URL:  "/uploads/" + subdir + "/" + filename,
		Name: payload.Name,
		Type: fileType,
		Size: int64(len(data)),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// Helper functions

func (h *UploadHandler) isValidFileType(filename string, uploadType string) bool {
	ext := strings.ToLower(filepath.Ext(filename))

	validTypes := map[string][]string{
		"photo":    {".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp"},
		"document": {".pdf", ".doc", ".docx", ".txt", ".xls", ".xlsx", ".png", ".jpg", ".jpeg"},
		"receipt":  {".pdf", ".jpg", ".jpeg", ".png", ".gif"},
		"manual":   {".pdf", ".doc", ".docx", ".txt", ".png", ".jpg", ".jpeg"},
	}

	allowed, exists := validTypes[uploadType]
	if !exists {
		return false
	}

	for _, allowedExt := range allowed {
		if ext == allowedExt {
			return true
		}
	}
	return false
}

func (h *UploadHandler) getFileExtension(base64Data string, filename string) string {
	// Try to get from filename
	if filename != "" {
		ext := strings.ToLower(filepath.Ext(filename))
		if ext != "" {
			return ext
		}
	}

	// Try to detect from base64 data URI
	if strings.Contains(base64Data, "data:image/") {
		if strings.Contains(base64Data, "data:image/png") {
			return ".png"
		} else if strings.Contains(base64Data, "data:image/jpeg") {
			return ".jpg"
		} else if strings.Contains(base64Data, "data:image/webp") {
			return ".webp"
		} else if strings.Contains(base64Data, "data:image/gif") {
			return ".gif"
		}
	} else if strings.Contains(base64Data, "data:application/pdf") {
		return ".pdf"
	}

	// Default to jpg
	return ".jpg"
}

func (h *UploadHandler) getSubdirectory(fileType string) string {
	switch fileType {
	case "photo":
		return "photos"
	case "document":
		return "documents"
	case "receipt":
		return "receipts"
	case "manual":
		return "manuals"
	default:
		return "files"
	}
}

func sanitizeFilename(filename string) string {
	// Remove path separators and special characters
	filename = strings.ReplaceAll(filename, "/", "-")
	filename = strings.ReplaceAll(filename, "\\", "-")
	filename = strings.ReplaceAll(filename, ":", "-")
	filename = strings.ReplaceAll(filename, "*", "-")
	filename = strings.ReplaceAll(filename, "?", "-")
	filename = strings.ReplaceAll(filename, "\"", "-")
	filename = strings.ReplaceAll(filename, "<", "-")
	filename = strings.ReplaceAll(filename, ">", "-")
	filename = strings.ReplaceAll(filename, "|", "-")
	return filename
}

// DeleteFile - delete an uploaded file
func (h *UploadHandler) DeleteFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fileType := vars["type"] // "photo", "document", etc.
	filename := vars["filename"]

	subdir := h.getSubdirectory(fileType)
	filepath := filepath.Join(h.uploadDir, subdir, filename)

	// Verify the file is within uploadDir (security check)
	absPath, _ := filepath.Abs(filepath)
	absUploadDir, _ := filepath.Abs(h.uploadDir)
	if !strings.HasPrefix(absPath, absUploadDir) {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	if err := os.Remove(filepath); err != nil {
		http.Error(w, "Failed to delete file", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}
