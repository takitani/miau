package services

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/opik/miau/internal/ports"
	"github.com/opik/miau/internal/storage"
)

// AttachmentServicePort implements ports.AttachmentService
type AttachmentServicePort struct {
	mu      sync.RWMutex
	storage ports.StoragePort
	imap    ports.IMAPPort
	account *ports.AccountInfo
	folder  *ports.Folder
}

// NewAttachmentServicePort creates a new AttachmentServicePort
func NewAttachmentServicePort(storage ports.StoragePort, imap ports.IMAPPort) *AttachmentServicePort {
	return &AttachmentServicePort{
		storage: storage,
		imap:    imap,
	}
}

// SetAccount sets the current account
func (s *AttachmentServicePort) SetAccount(account *ports.AccountInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.account = account
}

// SetFolder sets the current folder
func (s *AttachmentServicePort) SetFolder(folder *ports.Folder) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.folder = folder
}

// SetIMAPAdapter updates the IMAP adapter (used when switching accounts)
func (s *AttachmentServicePort) SetIMAPAdapter(imap ports.IMAPPort) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.imap = imap
}

// GetAttachments returns all attachments for an email
// First tries to get from database, then falls back to IMAP if not found
func (s *AttachmentServicePort) GetAttachments(ctx context.Context, emailID int64) ([]ports.Attachment, error) {
	// Try database first
	var attachments, err = s.storage.GetAttachmentsByEmail(ctx, emailID)
	if err == nil && len(attachments) > 0 {
		return attachments, nil
	}

	// Not in database - fetch from IMAP using BODYSTRUCTURE
	var email, emailErr = s.storage.GetEmail(ctx, emailID)
	if emailErr != nil {
		return nil, fmt.Errorf("failed to get email: %w", emailErr)
	}

	// Check if IMAP is connected
	if !s.imap.IsConnected() {
		return nil, fmt.Errorf("not connected to IMAP server")
	}

	// Select mailbox
	if email.FolderName != "" {
		if _, selErr := s.imap.SelectMailbox(ctx, email.FolderName); selErr != nil {
			return nil, fmt.Errorf("failed to select mailbox: %w", selErr)
		}
	}

	// Fetch attachment metadata via BODYSTRUCTURE
	var imapAtts, hasAtts, fetchErr = s.imap.FetchAttachmentMetadata(ctx, email.UID)
	if fetchErr != nil {
		return nil, fmt.Errorf("failed to fetch attachment metadata: %w", fetchErr)
	}

	if !hasAtts || len(imapAtts) == 0 {
		return []ports.Attachment{}, nil
	}

	// Convert to ports.Attachment
	var result []ports.Attachment
	for _, att := range imapAtts {
		var contentID = att.ContentID
		if len(contentID) > 2 && contentID[0] == '<' && contentID[len(contentID)-1] == '>' {
			contentID = contentID[1 : len(contentID)-1]
		}
		result = append(result, ports.Attachment{
			EmailID:     emailID,
			Filename:    att.Filename,
			ContentType: att.ContentType,
			ContentID:   contentID,
			Size:        att.Size,
			IsInline:    att.IsInline,
			PartNumber:  att.PartNumber,
			Encoding:    att.Encoding,
		})
	}

	return result, nil
}

// GetAttachment returns a single attachment by ID
func (s *AttachmentServicePort) GetAttachment(ctx context.Context, id int64) (*ports.Attachment, error) {
	return s.storage.GetAttachment(ctx, id)
}

// Download downloads an attachment and returns its content
func (s *AttachmentServicePort) Download(ctx context.Context, id int64) ([]byte, error) {
	// First, try to get from cache
	var data, err = s.storage.GetAttachmentContent(ctx, id)
	if err == nil && len(data) > 0 {
		return data, nil
	}

	// Not cached, need to fetch from IMAP
	var att, attErr = s.storage.GetAttachment(ctx, id)
	if attErr != nil {
		return nil, fmt.Errorf("failed to get attachment: %w", attErr)
	}

	// Get the email to get the UID
	var email, emailErr = s.storage.GetEmail(ctx, att.EmailID)
	if emailErr != nil {
		return nil, fmt.Errorf("failed to get email: %w", emailErr)
	}

	// Fetch using the direct storage package (for now)
	// This is a workaround - ideally we'd use the IMAP port directly
	var storageAtt, _ = storage.GetAttachmentByID(id)
	if storageAtt == nil {
		return nil, fmt.Errorf("attachment not found")
	}

	// Get the raw content from IMAP using the storage functions
	var partNumber = ""
	if storageAtt.PartNumber.Valid {
		partNumber = storageAtt.PartNumber.String
	}

	// We need to connect and fetch from IMAP
	// For now, return error if not connected
	if !s.imap.IsConnected() {
		return nil, fmt.Errorf("not connected to IMAP server")
	}

	// Select mailbox
	if email.FolderName != "" {
		if _, err := s.imap.SelectMailbox(ctx, email.FolderName); err != nil {
			return nil, fmt.Errorf("failed to select mailbox: %w", err)
		}
	}

	// Use the raw IMAP fetch - we need to access the underlying client
	// This is a limitation of the current architecture
	// For now, return an error suggesting the attachment should be cached during sync
	if partNumber == "" {
		return nil, fmt.Errorf("attachment part number not available")
	}

	// Return error - the full implementation requires access to the IMAP client
	// which would need refactoring of the architecture
	return nil, fmt.Errorf("attachment not cached - please sync emails first")
}

// SaveToFile downloads an attachment and saves it to a file
func (s *AttachmentServicePort) SaveToFile(ctx context.Context, id int64, path string) error {
	var data, err = s.Download(ctx, id)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// DownloadByPart downloads an attachment by email ID and MIME part number
func (s *AttachmentServicePort) DownloadByPart(ctx context.Context, emailID int64, partNumber string) ([]byte, error) {
	// Get the email to get the UID and folder
	var email, emailErr = s.storage.GetEmail(ctx, emailID)
	if emailErr != nil {
		return nil, fmt.Errorf("failed to get email: %w", emailErr)
	}

	// We need to connect and fetch from IMAP
	if !s.imap.IsConnected() {
		return nil, fmt.Errorf("not connected to IMAP server")
	}

	// Select mailbox
	if email.FolderName != "" {
		if _, mailboxErr := s.imap.SelectMailbox(ctx, email.FolderName); mailboxErr != nil {
			return nil, fmt.Errorf("failed to select mailbox: %w", mailboxErr)
		}
	}

	// Get attachment metadata to find the encoding
	var encoding = "base64" // default encoding for attachments
	var attachments, _, metaErr = s.imap.FetchAttachmentMetadata(ctx, email.UID)
	if metaErr == nil {
		for _, att := range attachments {
			if att.PartNumber == partNumber {
				encoding = att.Encoding
				break
			}
		}
	}

	// Fetch the attachment part (raw data)
	var rawData, fetchErr = s.imap.FetchAttachmentPart(ctx, email.UID, partNumber)
	if fetchErr != nil {
		return nil, fmt.Errorf("failed to fetch attachment: %w", fetchErr)
	}

	// Decode based on encoding
	var decoded, decodeErr = decodeAttachmentContent(rawData, encoding)
	if decodeErr != nil {
		return nil, fmt.Errorf("failed to decode attachment: %w", decodeErr)
	}

	return decoded, nil
}

// SaveToFileByPart downloads by part number and saves to a file
func (s *AttachmentServicePort) SaveToFileByPart(ctx context.Context, emailID int64, partNumber, path string) error {
	var data, err = s.DownloadByPart(ctx, emailID, partNumber)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// decodeAttachmentContent decodes attachment content based on encoding
func decodeAttachmentContent(data []byte, encoding string) ([]byte, error) {
	switch strings.ToLower(encoding) {
	case "base64":
		var decoded = make([]byte, base64.StdEncoding.DecodedLen(len(data)))
		var n, err = base64.StdEncoding.Decode(decoded, data)
		if err != nil {
			return nil, fmt.Errorf("failed to decode base64: %w", err)
		}
		return decoded[:n], nil

	case "quoted-printable":
		var result []byte
		for i := 0; i < len(data); i++ {
			if data[i] == '=' && i+2 < len(data) {
				if data[i+1] == '\r' || data[i+1] == '\n' {
					// Soft line break
					if data[i+1] == '\r' && i+2 < len(data) && data[i+2] == '\n' {
						i += 2
					} else {
						i += 1
					}
					continue
				}
				// Hex encoded byte
				var hex = string(data[i+1 : i+3])
				var b byte
				fmt.Sscanf(hex, "%02X", &b)
				result = append(result, b)
				i += 2
			} else {
				result = append(result, data[i])
			}
		}
		return result, nil

	case "7bit", "8bit", "binary", "":
		return data, nil

	default:
		// Unknown encoding, return as-is
		return data, nil
	}
}
