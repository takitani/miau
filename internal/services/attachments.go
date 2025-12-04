package services

import (
	"database/sql"
	"fmt"

	"github.com/opik/miau/internal/imap"
	"github.com/opik/miau/internal/storage"
)

// AttachmentService handles attachment-related operations
type AttachmentService struct{}

// NewAttachmentService creates a new AttachmentService
func NewAttachmentService() *AttachmentService {
	return &AttachmentService{}
}

// SyncAttachmentMetadata fetches attachment metadata from IMAP and stores in DB
// Returns the list of attachments and whether the email has attachments
func (s *AttachmentService) SyncAttachmentMetadata(client *imap.Client, emailID, accountID int64, uid uint32) ([]storage.AttachmentSummary, bool, error) {
	// Fetch attachment metadata from IMAP
	var attachments, hasAttachments, err = client.FetchAttachmentMetadata(uid)
	if err != nil {
		return nil, false, fmt.Errorf("failed to fetch attachment metadata: %w", err)
	}

	if !hasAttachments || len(attachments) == 0 {
		return nil, false, nil
	}

	// Store each attachment in database
	for _, att := range attachments {
		var attachment = &storage.Attachment{
			EmailID:     emailID,
			AccountID:   accountID,
			Filename:    att.Filename,
			ContentType: att.ContentType,
			Size:        att.Size,
			IsInline:    att.IsInline,
			PartNumber:  sql.NullString{String: att.PartNumber, Valid: att.PartNumber != ""},
			Encoding:    sql.NullString{String: att.Encoding, Valid: att.Encoding != ""},
			Charset:     sql.NullString{String: att.Charset, Valid: att.Charset != ""},
		}

		// Handle ContentID (remove angle brackets if present)
		if att.ContentID != "" {
			var contentID = att.ContentID
			if len(contentID) > 2 && contentID[0] == '<' && contentID[len(contentID)-1] == '>' {
				contentID = contentID[1 : len(contentID)-1]
			}
			attachment.ContentID = sql.NullString{String: contentID, Valid: true}
		}

		// Set disposition
		if att.IsInline {
			attachment.ContentDisposition = sql.NullString{String: "inline", Valid: true}
		} else {
			attachment.ContentDisposition = sql.NullString{String: "attachment", Valid: true}
		}

		if _, err := storage.UpsertAttachment(attachment); err != nil {
			// Log but don't fail - continue with other attachments
			continue
		}
	}

	// Return summaries from database
	var summaries, err2 = storage.GetAttachmentSummariesByEmail(emailID)
	return summaries, true, err2
}

// GetAttachmentContent fetches the actual attachment content from IMAP
func (s *AttachmentService) GetAttachmentContent(client *imap.Client, uid uint32, attachment *storage.Attachment) ([]byte, error) {
	// Get part number
	var partNumber = ""
	if attachment.PartNumber.Valid {
		partNumber = attachment.PartNumber.String
	}

	// Fetch the body part
	var rawData, err = client.FetchAttachmentPart(uid, partNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch attachment part: %w", err)
	}

	// Decode based on encoding
	var encoding = ""
	if attachment.Encoding.Valid {
		encoding = attachment.Encoding.String
	}

	var decoded, err2 = imap.DecodeAttachmentContent(rawData, encoding)
	if err2 != nil {
		return nil, fmt.Errorf("failed to decode attachment: %w", err2)
	}

	return decoded, nil
}

// CacheAttachment downloads and caches an attachment
func (s *AttachmentService) CacheAttachment(client *imap.Client, uid uint32, attachment *storage.Attachment) error {
	// Fetch content
	var content, err = s.GetAttachmentContent(client, uid, attachment)
	if err != nil {
		return err
	}

	// Cache in database (no compression for now)
	if err := storage.CacheAttachmentContent(attachment.ID, content, false); err != nil {
		return fmt.Errorf("failed to cache attachment: %w", err)
	}

	return nil
}

// GetCachedOrFetchAttachment returns cached content or fetches from IMAP
func (s *AttachmentService) GetCachedOrFetchAttachment(client *imap.Client, uid uint32, attachment *storage.Attachment) ([]byte, error) {
	// Try to get from cache first
	if attachment.IsCached {
		var data, _, err = storage.GetCachedAttachmentContent(attachment.ID)
		if err == nil {
			return data, nil
		}
		// Cache miss or error, fetch from IMAP
	}

	// Fetch from IMAP
	var content, err = s.GetAttachmentContent(client, uid, attachment)
	if err != nil {
		return nil, err
	}

	// Cache for future use (async would be better but keeping it simple)
	storage.CacheAttachmentContent(attachment.ID, content, false)

	return content, nil
}

// GetAttachmentsByEmail returns attachments for an email from database
func (s *AttachmentService) GetAttachmentsByEmail(emailID int64) ([]storage.AttachmentSummary, error) {
	return storage.GetAttachmentSummariesByEmail(emailID)
}

// GetAttachmentByID returns a single attachment
func (s *AttachmentService) GetAttachmentByID(id int64) (*storage.Attachment, error) {
	return storage.GetAttachmentByID(id)
}

// FormatSize formats file size for display
func FormatSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	}
	if size < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(size)/1024)
	}
	if size < 1024*1024*1024 {
		return fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
	}
	return fmt.Sprintf("%.1f GB", float64(size)/(1024*1024*1024))
}

// GetFileIcon returns an emoji icon based on content type
func GetFileIcon(contentType string) string {
	switch {
	case contentType == "application/pdf":
		return "📄"
	case contentType == "application/zip" || contentType == "application/x-zip-compressed":
		return "📦"
	case contentType == "application/msword" || contentType == "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		return "📝"
	case contentType == "application/vnd.ms-excel" || contentType == "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":
		return "📊"
	case contentType == "application/vnd.ms-powerpoint" || contentType == "application/vnd.openxmlformats-officedocument.presentationml.presentation":
		return "📽"
	case len(contentType) >= 5 && contentType[:5] == "image":
		return "🖼"
	case len(contentType) >= 5 && contentType[:5] == "video":
		return "🎬"
	case len(contentType) >= 5 && contentType[:5] == "audio":
		return "🎵"
	case len(contentType) >= 4 && contentType[:4] == "text":
		return "📃"
	default:
		return "📎"
	}
}
