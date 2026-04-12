package main

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/ledongthuc/pdf"
)

// extractText reads selectable text from a PDF content stream.
//
// Returns ("", nil) for image-only/scanned PDFs — they have no text layer.
// Returns an error for corrupt or encrypted files.
//
// TODO: OCR fallback for scanned drawings:
//   out, err := exec.Command("tesseract", filePath, "stdout", "--psm", "6").Output()
//   Tesseract must be installed separately; treat absence as a soft failure.
func extractText(filePath string) (string, error) {
	f, r, err := pdf.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("open: %w", err)
	}
	defer f.Close()

	plain, err := r.GetPlainText()
	if err != nil {
		return "", fmt.Errorf("read text layer: %w", err)
	}

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(plain); err != nil {
		return "", fmt.Errorf("buffer: %w", err)
	}

	return strings.TrimSpace(buf.String()), nil
}
