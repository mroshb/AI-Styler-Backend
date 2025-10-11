package worker

import (
	"context"
	"fmt"
	"log"
)

// FileValidator handles file validation and corruption detection
type FileValidator struct {
	maxFileSize    int64
	supportedTypes []string
	checksumAlgo   string
}

// FileValidationResult represents the result of file validation
type FileValidationResult struct {
	Valid    bool
	Error    error
	FileSize int64
	MimeType string
	Checksum string
	Warnings []string
}

// NewFileValidator creates a new file validator
func NewFileValidator(maxFileSize int64, supportedTypes []string) *FileValidator {
	if maxFileSize == 0 {
		maxFileSize = 10 * 1024 * 1024 // 10MB default
	}
	if len(supportedTypes) == 0 {
		supportedTypes = []string{"image/jpeg", "image/png", "image/webp"}
	}

	return &FileValidator{
		maxFileSize:    maxFileSize,
		supportedTypes: supportedTypes,
		checksumAlgo:   "sha256",
	}
}

// ValidateFile validates a file for corruption and format
func (fv *FileValidator) ValidateFile(ctx context.Context, fileData []byte, filename string) (*FileValidationResult, error) {
	result := &FileValidationResult{
		Valid:    true,
		FileSize: int64(len(fileData)),
		Warnings: make([]string, 0),
	}

	// Check file size
	if result.FileSize == 0 {
		result.Valid = false
		result.Error = fmt.Errorf("file is empty")
		return result, result.Error
	}

	if result.FileSize > fv.maxFileSize {
		result.Valid = false
		result.Error = fmt.Errorf("file too large: %d bytes (max: %d)", result.FileSize, fv.maxFileSize)
		return result, result.Error
	}

	// Detect MIME type
	mimeType, err := fv.detectMimeType(fileData)
	if err != nil {
		result.Valid = false
		result.Error = fmt.Errorf("failed to detect MIME type: %w", err)
		return result, result.Error
	}
	result.MimeType = mimeType

	// Check if MIME type is supported
	if !fv.isSupportedType(mimeType) {
		result.Valid = false
		result.Error = fmt.Errorf("unsupported file type: %s", mimeType)
		return result, result.Error
	}

	// Validate file format
	if err := fv.validateFileFormat(fileData, mimeType); err != nil {
		result.Valid = false
		result.Error = fmt.Errorf("file format validation failed: %w", err)
		return result, err
	}

	// Calculate checksum
	checksum, err := fv.calculateChecksum(fileData)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("failed to calculate checksum: %v", err))
	} else {
		result.Checksum = checksum
	}

	// Additional validations
	fv.performAdditionalValidations(fileData, mimeType, result)

	return result, nil
}

// detectMimeType detects the MIME type of the file
func (fv *FileValidator) detectMimeType(fileData []byte) (string, error) {
	// Check magic bytes for common image formats
	if len(fileData) < 4 {
		return "", fmt.Errorf("file too small to determine type")
	}

	// JPEG
	if fileData[0] == 0xFF && fileData[1] == 0xD8 {
		return "image/jpeg", nil
	}

	// PNG
	if fileData[0] == 0x89 && fileData[1] == 0x50 && fileData[2] == 0x4E && fileData[3] == 0x47 {
		return "image/png", nil
	}

	// WebP
	if len(fileData) >= 12 &&
		fileData[0] == 0x52 && fileData[1] == 0x49 && fileData[2] == 0x46 && fileData[3] == 0x46 &&
		fileData[8] == 0x57 && fileData[9] == 0x45 && fileData[10] == 0x42 && fileData[11] == 0x50 {
		return "image/webp", nil
	}

	// GIF
	if len(fileData) >= 6 &&
		fileData[0] == 0x47 && fileData[1] == 0x49 && fileData[2] == 0x46 {
		return "image/gif", nil
	}

	return "application/octet-stream", fmt.Errorf("unknown file type")
}

// isSupportedType checks if the MIME type is supported
func (fv *FileValidator) isSupportedType(mimeType string) bool {
	for _, supportedType := range fv.supportedTypes {
		if mimeType == supportedType {
			return true
		}
	}
	return false
}

// validateFileFormat validates the file format
func (fv *FileValidator) validateFileFormat(fileData []byte, mimeType string) error {
	switch mimeType {
	case "image/jpeg":
		return fv.validateJPEG(fileData)
	case "image/png":
		return fv.validatePNG(fileData)
	case "image/webp":
		return fv.validateWebP(fileData)
	case "image/gif":
		return fv.validateGIF(fileData)
	default:
		return fmt.Errorf("validation not implemented for type: %s", mimeType)
	}
}

// validateJPEG validates JPEG format
func (fv *FileValidator) validateJPEG(fileData []byte) error {
	if len(fileData) < 4 {
		return fmt.Errorf("JPEG file too small")
	}

	// Check for JPEG header
	if fileData[0] != 0xFF || fileData[1] != 0xD8 {
		return fmt.Errorf("invalid JPEG header")
	}

	// Check for JPEG footer
	if len(fileData) < 2 || fileData[len(fileData)-2] != 0xFF || fileData[len(fileData)-1] != 0xD9 {
		return fmt.Errorf("invalid JPEG footer - file may be corrupted")
	}

	// Basic structure validation
	return fv.validateJPEGStructure(fileData)
}

// validatePNG validates PNG format
func (fv *FileValidator) validatePNG(fileData []byte) error {
	if len(fileData) < 8 {
		return fmt.Errorf("PNG file too small")
	}

	// Check PNG signature
	pngSignature := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	for i, b := range pngSignature {
		if fileData[i] != b {
			return fmt.Errorf("invalid PNG signature")
		}
	}

	// Basic chunk validation
	return fv.validatePNGChunks(fileData)
}

// validateWebP validates WebP format
func (fv *FileValidator) validateWebP(fileData []byte) error {
	if len(fileData) < 12 {
		return fmt.Errorf("WebP file too small")
	}

	// Check RIFF header
	if fileData[0] != 0x52 || fileData[1] != 0x49 || fileData[2] != 0x46 || fileData[3] != 0x46 {
		return fmt.Errorf("invalid WebP RIFF header")
	}

	// Check WebP signature
	if fileData[8] != 0x57 || fileData[9] != 0x45 || fileData[10] != 0x42 || fileData[11] != 0x50 {
		return fmt.Errorf("invalid WebP signature")
	}

	return nil
}

// validateGIF validates GIF format
func (fv *FileValidator) validateGIF(fileData []byte) error {
	if len(fileData) < 6 {
		return fmt.Errorf("GIF file too small")
	}

	// Check GIF signature
	if fileData[0] != 0x47 || fileData[1] != 0x49 || fileData[2] != 0x46 {
		return fmt.Errorf("invalid GIF signature")
	}

	return nil
}

// validateJPEGStructure validates JPEG structure
func (fv *FileValidator) validateJPEGStructure(fileData []byte) error {
	// Look for required JPEG markers
	requiredMarkers := []byte{0xFF, 0xD8} // SOI
	optionalMarkers := []byte{0xFF, 0xD9} // EOI

	// Check for SOI marker
	if len(fileData) < 2 || fileData[0] != requiredMarkers[0] || fileData[1] != requiredMarkers[1] {
		return fmt.Errorf("missing JPEG SOI marker")
	}

	// Check for EOI marker
	if len(fileData) < 2 || fileData[len(fileData)-2] != optionalMarkers[0] || fileData[len(fileData)-1] != optionalMarkers[1] {
		return fmt.Errorf("missing JPEG EOI marker")
	}

	return nil
}

// validatePNGChunks validates PNG chunks
func (fv *FileValidator) validatePNGChunks(fileData []byte) error {
	// Basic chunk validation - look for IHDR and IEND chunks
	hasIHDR := false
	hasIEND := false

	for i := 8; i < len(fileData)-8; i++ {
		if i+8 < len(fileData) {
			chunkType := string(fileData[i+4 : i+8])
			if chunkType == "IHDR" {
				hasIHDR = true
			}
			if chunkType == "IEND" {
				hasIEND = true
				break
			}
		}
	}

	if !hasIHDR {
		return fmt.Errorf("missing PNG IHDR chunk")
	}
	if !hasIEND {
		return fmt.Errorf("missing PNG IEND chunk")
	}

	return nil
}

// performAdditionalValidations performs additional file validations
func (fv *FileValidator) performAdditionalValidations(fileData []byte, mimeType string, result *FileValidationResult) {
	// Check for suspicious patterns that might indicate corruption
	if fv.hasSuspiciousPatterns(fileData) {
		result.Warnings = append(result.Warnings, "file contains suspicious patterns")
	}

	// Check file size vs content ratio
	if fv.hasUnusualSizeRatio(fileData, mimeType) {
		result.Warnings = append(result.Warnings, "unusual file size ratio detected")
	}

	// Check for embedded data
	if fv.hasEmbeddedData(fileData) {
		result.Warnings = append(result.Warnings, "file may contain embedded data")
	}
}

// hasSuspiciousPatterns checks for suspicious patterns in the file
func (fv *FileValidator) hasSuspiciousPatterns(fileData []byte) bool {
	// Look for patterns that might indicate corruption
	nullCount := 0
	for _, b := range fileData {
		if b == 0 {
			nullCount++
		}
	}

	// If more than 10% of the file is null bytes, it's suspicious
	return float64(nullCount)/float64(len(fileData)) > 0.1
}

// hasUnusualSizeRatio checks for unusual size ratios
func (fv *FileValidator) hasUnusualSizeRatio(fileData []byte, mimeType string) bool {
	// Basic heuristic: very small files for their type might be suspicious
	switch mimeType {
	case "image/jpeg":
		return len(fileData) < 1000 // Less than 1KB for JPEG is unusual
	case "image/png":
		return len(fileData) < 500 // Less than 500B for PNG is unusual
	case "image/webp":
		return len(fileData) < 500 // Less than 500B for WebP is unusual
	}
	return false
}

// hasEmbeddedData checks for embedded data in the file
func (fv *FileValidator) hasEmbeddedData(fileData []byte) bool {
	// Look for common embedded data signatures
	embeddedSignatures := [][]byte{
		{0x50, 0x4B},             // ZIP/Office documents
		{0x25, 0x50, 0x44, 0x46}, // PDF
		{0x4D, 0x5A},             // Windows executable
	}

	for _, sig := range embeddedSignatures {
		if fv.findSignature(fileData, sig) {
			return true
		}
	}
	return false
}

// findSignature searches for a signature in the file
func (fv *FileValidator) findSignature(fileData []byte, signature []byte) bool {
	if len(signature) > len(fileData) {
		return false
	}

	for i := 0; i <= len(fileData)-len(signature); i++ {
		match := true
		for j, b := range signature {
			if fileData[i+j] != b {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// calculateChecksum calculates the checksum of the file
func (fv *FileValidator) calculateChecksum(fileData []byte) (string, error) {
	// Simple implementation - in production, use crypto/sha256
	hash := uint32(0)
	for _, b := range fileData {
		hash = hash*31 + uint32(b)
	}
	return fmt.Sprintf("%x", hash), nil
}

// FileCorruptionDetector detects file corruption
type FileCorruptionDetector struct {
	validator *FileValidator
}

// NewFileCorruptionDetector creates a new file corruption detector
func NewFileCorruptionDetector(validator *FileValidator) *FileCorruptionDetector {
	return &FileCorruptionDetector{
		validator: validator,
	}
}

// DetectCorruption detects file corruption
func (fcd *FileCorruptionDetector) DetectCorruption(ctx context.Context, fileData []byte, filename string) error {
	result, err := fcd.validator.ValidateFile(ctx, fileData, filename)
	if err != nil {
		return fmt.Errorf("file validation failed: %w", err)
	}

	if !result.Valid {
		return fmt.Errorf("file is corrupted: %v", result.Error)
	}

	// Check for warnings that might indicate corruption
	if len(result.Warnings) > 0 {
		log.Printf("File validation warnings for %s: %v", filename, result.Warnings)
	}

	return nil
}

// IsFileCorrupted checks if a file is corrupted
func (fcd *FileCorruptionDetector) IsFileCorrupted(ctx context.Context, fileData []byte, filename string) bool {
	err := fcd.DetectCorruption(ctx, fileData, filename)
	return err != nil
}

// GetFileInfo returns information about the file
func (fcd *FileCorruptionDetector) GetFileInfo(ctx context.Context, fileData []byte, filename string) (*FileValidationResult, error) {
	return fcd.validator.ValidateFile(ctx, fileData, filename)
}
