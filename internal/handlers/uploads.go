package handlers

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
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

// UploadInventoryReceipt - multipart form file upload for inventory receipts; saves a record to documents table.
func (h *UploadHandler) UploadInventoryReceipt(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	itemID := vars["id"]

	h.uploadInventoryDocument(w, r, itemID, "receipt", "receipts")
}

// UploadInventoryManual - multipart form file upload for inventory manuals; saves a record to documents table.
func (h *UploadHandler) UploadInventoryManual(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	itemID := vars["id"]

	h.uploadInventoryDocument(w, r, itemID, "manual", "manuals")
}

// uploadInventoryDocument saves the uploaded file to disk and inserts a record into the documents table.
func (h *UploadHandler) uploadInventoryDocument(w http.ResponseWriter, r *http.Request, itemID, docType, subdir string) {
	if err := r.ParseMultipartForm(100 * 1024 * 1024); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	file, fileHeader, err := r.FormFile(docType)
	if err != nil {
		http.Error(w, "No file provided", http.StatusBadRequest)
		return
	}
	defer file.Close()

	if !h.isValidFileType(fileHeader.Filename, docType) {
		http.Error(w, "Invalid file type", http.StatusBadRequest)
		return
	}

	filename := strconv.FormatInt(time.Now().UnixNano(), 10) + "-" + sanitizeFilename(fileHeader.Filename)
	filePath := filepath.Join(h.uploadDir, subdir, filename)

	dst, err := os.Create(filePath)
	if err != nil {
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	if _, err = io.Copy(dst, file); err != nil {
		http.Error(w, "Failed to copy file", http.StatusInternalServerError)
		return
	}

	displayName := r.FormValue("name")
	if displayName == "" {
		displayName = fileHeader.Filename
	}
	url := "/uploads/" + subdir + "/" + filename

	// Persist document record in the database
	type docResponse struct {
		ID         int64     `json:"id"`
		Name       string    `json:"name"`
		URL        string    `json:"url"`
		UploadedAt time.Time `json:"uploaded_at"`
	}
	var doc docResponse
	doc.Name = displayName
	doc.URL = url

	if dbErr := h.db.QueryRow(
		"INSERT INTO documents (inventory_item_id, doc_type, name, url) VALUES ($1, $2, $3, $4) RETURNING id, uploaded_at",
		itemID, docType, displayName, url,
	).Scan(&doc.ID, &doc.UploadedAt); dbErr != nil {
		http.Error(w, "Failed to save document record", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(doc)
}

// DeleteDocument deletes a document record from the DB and its file from disk.
func (h *UploadHandler) DeleteDocument(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	docID := vars["docId"]

	var docURL string
	if err := h.db.QueryRow("SELECT url FROM documents WHERE id = $1", docID).Scan(&docURL); err != nil {
		http.Error(w, "Document not found", http.StatusNotFound)
		return
	}

	if _, err := h.db.Exec("DELETE FROM documents WHERE id = $1", docID); err != nil {
		http.Error(w, "Failed to delete document", http.StatusInternalServerError)
		return
	}

	// Best-effort delete of the physical file; log any error for operator visibility.
	if strings.HasPrefix(docURL, "/uploads/") {
		relPath := strings.TrimPrefix(docURL, "/uploads/")
		if rmErr := os.Remove(filepath.Join(h.uploadDir, relPath)); rmErr != nil {
			log.Printf("DeleteDocument: failed to remove file %s: %v", relPath, rmErr)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
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
	fileType := vars["type"] // "photo", "document", "receipt", "manual"

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
	filePath := filepath.Join(uploadPath, filename)

	// Save file
	if err := os.WriteFile(filePath, data, 0644); err != nil {
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
		"receipt":  {".pdf", ".jpg", ".jpeg", ".png", ".gif", ".webp", ".txt"},
		"manual":   {".pdf", ".jpg", ".jpeg", ".png", ".gif", ".webp", ".txt"},
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
	filePath := filepath.Join(h.uploadDir, subdir, filename)

	// Verify the file is within uploadDir (security check) - simple check without Abs
	if !strings.HasPrefix(filePath, h.uploadDir) {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	if err := os.Remove(filePath); err != nil {
		http.Error(w, "Failed to delete file", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}
