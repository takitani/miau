// Package email provides email parsing utilities.
// This package extracts text, HTML, and attachments from raw email data.
package email

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"regexp"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/htmlindex"
)

// Attachment represents an email attachment
type Attachment struct {
	Filename    string
	ContentType string
	ContentID   string // For inline images (cid:xxx)
	Size        int64
	Data        []byte
	IsInline    bool
}

// ParsedEmail contains all parsed content from a raw email
type ParsedEmail struct {
	TextBody    string
	HTMLBody    string
	Attachments []Attachment
	CIDMap      map[string]string // Maps Content-ID to data URI
}

// Parse extracts all content from raw email data
func Parse(rawData []byte) (*ParsedEmail, error) {
	var parsed = &ParsedEmail{
		CIDMap: make(map[string]string),
	}

	parsed.TextBody = ExtractText(rawData)
	parsed.HTMLBody, parsed.CIDMap = ExtractHTMLWithCID(rawData)

	// Replace CID references with data URIs
	if len(parsed.CIDMap) > 0 {
		parsed.HTMLBody = ReplaceCIDReferences(parsed.HTMLBody, parsed.CIDMap)
	}

	parsed.Attachments = ExtractAttachments(rawData)

	return parsed, nil
}

// ExtractText extracts plain text content from raw email data
func ExtractText(rawData []byte) string {
	var msg, err = mail.ReadMessage(bytes.NewReader(rawData))
	if err != nil {
		return ""
	}

	var contentType = msg.Header.Get("Content-Type")
	var mediaType, params, _ = mime.ParseMediaType(contentType)

	// Direct text
	if strings.HasPrefix(mediaType, "text/plain") {
		var body, _ = io.ReadAll(msg.Body)
		return DecodeBody(body, msg.Header.Get("Content-Transfer-Encoding"))
	}

	// Multipart - find text/plain part
	if strings.HasPrefix(mediaType, "multipart/") {
		var boundary = params["boundary"]
		if boundary != "" {
			return findTextPart(msg.Body, boundary)
		}
	}

	return ""
}

func findTextPart(r io.Reader, boundary string) string {
	var mr = multipart.NewReader(r, boundary)
	for {
		var part, err = mr.NextPart()
		if err != nil {
			break
		}

		var contentType = part.Header.Get("Content-Type")
		var mediaType, params, _ = mime.ParseMediaType(contentType)

		if strings.HasPrefix(mediaType, "text/plain") {
			var body, _ = io.ReadAll(part)
			return DecodeBody(body, part.Header.Get("Content-Transfer-Encoding"))
		}

		// Nested multipart
		if strings.HasPrefix(mediaType, "multipart/") {
			var boundary = params["boundary"]
			if boundary != "" {
				if text := findTextPart(part, boundary); text != "" {
					return text
				}
			}
		}
	}
	return ""
}

// ExtractHTML extracts HTML content from raw email data with CID references replaced
func ExtractHTML(rawData []byte) string {
	var htmlContent, cidMap = ExtractHTMLWithCID(rawData)

	// Replace cid: references with data URIs
	if len(cidMap) > 0 {
		htmlContent = ReplaceCIDReferences(htmlContent, cidMap)
	}

	return htmlContent
}

// ExtractHTMLWithCID extracts HTML and returns CID map separately
func ExtractHTMLWithCID(rawData []byte) (string, map[string]string) {
	var cidMap = make(map[string]string)

	var msg, err = mail.ReadMessage(bytes.NewReader(rawData))
	if err != nil {
		return "", cidMap
	}

	var contentType = msg.Header.Get("Content-Type")
	var mediaType, params, _ = mime.ParseMediaType(contentType)

	// Direct HTML
	if strings.HasPrefix(mediaType, "text/html") {
		var body, _ = io.ReadAll(msg.Body)
		var charset = params["charset"]
		return DecodeBodyWithCharset(body, msg.Header.Get("Content-Transfer-Encoding"), charset), cidMap
	}

	// Multipart - find HTML and images
	if strings.HasPrefix(mediaType, "multipart/") {
		var boundary = params["boundary"]
		if boundary != "" {
			return findHTMLAndImages(msg.Body, boundary, cidMap)
		}
	}

	return "", cidMap
}

// findHTMLAndImages searches for HTML and extracts embedded images
func findHTMLAndImages(r io.Reader, boundary string, cidMap map[string]string) (string, map[string]string) {
	var htmlContent string
	var mr = multipart.NewReader(r, boundary)

	// First pass: collect all parts
	type mimePart struct {
		contentType string
		contentID   string
		encoding    string
		body        []byte
	}
	var parts []mimePart

	for {
		var part, err = mr.NextPart()
		if err != nil {
			break
		}

		var body, _ = io.ReadAll(part)
		parts = append(parts, mimePart{
			contentType: part.Header.Get("Content-Type"),
			contentID:   part.Header.Get("Content-Id"),
			encoding:    part.Header.Get("Content-Transfer-Encoding"),
			body:        body,
		})
	}

	// Process parts
	for _, part := range parts {
		var mediaType, params, _ = mime.ParseMediaType(part.contentType)

		// HTML
		if strings.HasPrefix(mediaType, "text/html") && htmlContent == "" {
			var charset = params["charset"]
			htmlContent = DecodeBodyWithCharset(part.body, part.encoding, charset)
		}

		// Images with Content-ID
		var contentID = part.contentID
		if contentID != "" && strings.HasPrefix(mediaType, "image/") {
			// Remove < > from Content-ID
			contentID = strings.Trim(contentID, "<>")

			// Decode image body
			var imageData = DecodeImageBody(part.body, part.encoding)

			// Create data URI
			var dataURI = fmt.Sprintf("data:%s;base64,%s", mediaType, base64.StdEncoding.EncodeToString(imageData))
			cidMap[contentID] = dataURI
		}

		// Nested multipart
		if strings.HasPrefix(mediaType, "multipart/") {
			var nestedBoundary = params["boundary"]
			if nestedBoundary != "" {
				var nestedHTML, nestedCID = findHTMLAndImages(bytes.NewReader(part.body), nestedBoundary, cidMap)
				if nestedHTML != "" && htmlContent == "" {
					htmlContent = nestedHTML
				}
				for k, v := range nestedCID {
					cidMap[k] = v
				}
			}
		}
	}

	return htmlContent, cidMap
}

// DecodeImageBody decodes image body content
func DecodeImageBody(body []byte, encoding string) []byte {
	switch strings.ToLower(encoding) {
	case "base64":
		var decoded, err = base64.StdEncoding.DecodeString(string(body))
		if err != nil {
			// Try removing whitespace
			var cleaned = strings.ReplaceAll(string(body), "\n", "")
			cleaned = strings.ReplaceAll(cleaned, "\r", "")
			cleaned = strings.ReplaceAll(cleaned, " ", "")
			decoded, _ = base64.StdEncoding.DecodeString(cleaned)
		}
		return decoded
	case "quoted-printable":
		var decoded, _ = io.ReadAll(quotedprintable.NewReader(bytes.NewReader(body)))
		return decoded
	default:
		return body
	}
}

// ExtractAttachments extracts all image attachments from an email
func ExtractAttachments(rawData []byte) []Attachment {
	var attachments []Attachment

	var msg, err = mail.ReadMessage(bytes.NewReader(rawData))
	if err != nil {
		return attachments
	}

	var contentType = msg.Header.Get("Content-Type")
	var mediaType, params, _ = mime.ParseMediaType(contentType)

	// Multipart - find attachments and images
	if strings.HasPrefix(mediaType, "multipart/") {
		var boundary = params["boundary"]
		if boundary != "" {
			attachments = findImageAttachments(msg.Body, boundary)
		}
	}

	return attachments
}

// findImageAttachments searches for images (inline and attachments) in email
func findImageAttachments(r io.Reader, boundary string) []Attachment {
	var attachments []Attachment
	var mr = multipart.NewReader(r, boundary)

	for {
		var part, err = mr.NextPart()
		if err != nil {
			break
		}

		var body, _ = io.ReadAll(part)
		var contentType = part.Header.Get("Content-Type")
		var mediaType, params, _ = mime.ParseMediaType(contentType)
		var disposition = part.Header.Get("Content-Disposition")
		var contentID = strings.Trim(part.Header.Get("Content-Id"), "<>")
		var encoding = part.Header.Get("Content-Transfer-Encoding")

		// Check if it's an image (inline or attachment)
		if strings.HasPrefix(mediaType, "image/") {
			var decoded = DecodeImageBody(body, encoding)

			// Try to get filename
			var filename = params["name"]
			if filename == "" {
				var _, dispParams, _ = mime.ParseMediaType(disposition)
				filename = dispParams["filename"]
			}
			if filename == "" && contentID != "" {
				filename = contentID
			}
			if filename == "" {
				// Generate name based on type
				var ext = "img"
				switch mediaType {
				case "image/jpeg":
					ext = "jpg"
				case "image/png":
					ext = "png"
				case "image/gif":
					ext = "gif"
				case "image/webp":
					ext = "webp"
				}
				filename = fmt.Sprintf("image.%s", ext)
			}

			var isInline = contentID != "" || strings.HasPrefix(disposition, "inline")

			attachments = append(attachments, Attachment{
				Filename:    filename,
				ContentType: mediaType,
				ContentID:   contentID,
				Size:        int64(len(decoded)),
				Data:        decoded,
				IsInline:    isInline,
			})
		}

		// Nested multipart (common in emails with alternative + related)
		if strings.HasPrefix(mediaType, "multipart/") {
			var nestedBoundary = params["boundary"]
			if nestedBoundary != "" {
				var nested = findImageAttachments(bytes.NewReader(body), nestedBoundary)
				attachments = append(attachments, nested...)
			}
		}
	}

	return attachments
}

// ReplaceCIDReferences replaces cid:xxx references with data URIs
func ReplaceCIDReferences(html string, cidMap map[string]string) string {
	// Pattern: src="cid:xxx" or src='cid:xxx'
	var cidRegex = regexp.MustCompile(`(src=["'])cid:([^"']+)(["'])`)

	return cidRegex.ReplaceAllStringFunc(html, func(match string) string {
		var submatches = cidRegex.FindStringSubmatch(match)
		if len(submatches) >= 4 {
			var cid = submatches[2]
			if dataURI, ok := cidMap[cid]; ok {
				return submatches[1] + dataURI + submatches[3]
			}
		}
		return match
	})
}

// FindHTMLPart finds HTML part in multipart message
func FindHTMLPart(r io.Reader, boundary string) string {
	var mr = multipart.NewReader(r, boundary)
	for {
		var part, err = mr.NextPart()
		if err != nil {
			break
		}

		var contentType = part.Header.Get("Content-Type")
		var mediaType, params, _ = mime.ParseMediaType(contentType)

		if strings.HasPrefix(mediaType, "text/html") {
			var body, _ = io.ReadAll(part)
			return DecodeBody(body, part.Header.Get("Content-Transfer-Encoding"))
		}

		// Nested multipart
		if strings.HasPrefix(mediaType, "multipart/") {
			var boundary = params["boundary"]
			if boundary != "" {
				if html := FindHTMLPart(part, boundary); html != "" {
					return html
				}
			}
		}
	}
	return ""
}

// DecodeBody decodes body with transfer encoding
func DecodeBody(body []byte, encoding string) string {
	return DecodeBodyWithCharset(body, encoding, "")
}

// DecodeBodyWithCharset decodes body with transfer encoding and charset
func DecodeBodyWithCharset(body []byte, encoding string, charset string) string {
	var decoded []byte

	switch strings.ToLower(encoding) {
	case "quoted-printable":
		var d, err = io.ReadAll(quotedprintable.NewReader(bytes.NewReader(body)))
		if err != nil {
			decoded = body
		} else {
			decoded = d
		}
	case "base64":
		var d, err = base64.StdEncoding.DecodeString(string(body))
		if err != nil {
			// Try cleaning
			var cleaned = strings.ReplaceAll(string(body), "\n", "")
			cleaned = strings.ReplaceAll(cleaned, "\r", "")
			d, _ = base64.StdEncoding.DecodeString(cleaned)
		}
		decoded = d
	default:
		decoded = body
	}

	// Convert charset if needed
	if charset != "" && !strings.EqualFold(charset, "utf-8") && !strings.EqualFold(charset, "us-ascii") {
		var converted = ConvertCharset(decoded, charset)
		if converted != "" {
			return converted
		}
	}

	return string(decoded)
}

// ConvertCharset converts from a charset to UTF-8
func ConvertCharset(data []byte, charset string) string {
	// Try using htmlindex first
	var enc, err = htmlindex.Get(charset)
	if err == nil {
		var decoder = enc.NewDecoder()
		var result, err2 = decoder.Bytes(data)
		if err2 == nil {
			return string(result)
		}
	}

	// Fallback for common charsets
	charset = strings.ToLower(charset)
	switch {
	case strings.Contains(charset, "iso-8859-1"), strings.Contains(charset, "latin1"):
		var decoder = charmap.ISO8859_1.NewDecoder()
		var result, _ = decoder.Bytes(data)
		return string(result)
	case strings.Contains(charset, "iso-8859-15"), strings.Contains(charset, "latin9"):
		var decoder = charmap.ISO8859_15.NewDecoder()
		var result, _ = decoder.Bytes(data)
		return string(result)
	case strings.Contains(charset, "windows-1252"):
		var decoder = charmap.Windows1252.NewDecoder()
		var result, _ = decoder.Bytes(data)
		return string(result)
	}

	return ""
}

// HTMLToText converts HTML to readable plain text
func HTMLToText(htmlContent string) string {
	var doc, err = html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return htmlContent
	}

	var buf bytes.Buffer
	var extractTextFromNode func(*html.Node)
	extractTextFromNode = func(n *html.Node) {
		// Ignore scripts, styles and comments
		if n.Type == html.ElementNode {
			switch n.Data {
			case "script", "style", "head", "noscript":
				return
			case "br":
				buf.WriteString("\n")
				return
			case "p", "div", "tr", "li", "h1", "h2", "h3", "h4", "h5", "h6":
				buf.WriteString("\n")
			case "td", "th":
				buf.WriteString("\t")
			}
		}

		if n.Type == html.TextNode {
			var text = strings.TrimSpace(n.Data)
			if text != "" {
				buf.WriteString(text)
				buf.WriteString(" ")
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extractTextFromNode(c)
		}

		// Add line break after block elements
		if n.Type == html.ElementNode {
			switch n.Data {
			case "p", "div", "tr", "li", "h1", "h2", "h3", "h4", "h5", "h6", "blockquote":
				buf.WriteString("\n")
			}
		}
	}

	extractTextFromNode(doc)

	// Clean multiple blank lines
	var result = buf.String()
	for strings.Contains(result, "\n\n\n") {
		result = strings.ReplaceAll(result, "\n\n\n", "\n\n")
	}
	return strings.TrimSpace(result)
}
