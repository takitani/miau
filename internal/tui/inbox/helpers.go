package inbox

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

// extractText extrai conteúdo text/plain de um email MIME
func extractText(rawData []byte) string {
	var msg, err = mail.ReadMessage(bytes.NewReader(rawData))
	if err != nil {
		return ""
	}

	var contentType = msg.Header.Get("Content-Type")
	var mediaType, params, _ = mime.ParseMediaType(contentType)

	// Se for texto direto
	if strings.HasPrefix(mediaType, "text/plain") {
		var body, _ = io.ReadAll(msg.Body)
		return decodeBody(body, msg.Header.Get("Content-Transfer-Encoding"))
	}

	// Se for multipart, procura a parte text/plain
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
			return decodeBody(body, part.Header.Get("Content-Transfer-Encoding"))
		}

		// Multipart aninhado
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

// extractHTML extrai conteúdo HTML de um email MIME
func extractHTML(rawData []byte) string {
	var htmlContent, cidMap = extractHTMLWithCID(rawData)

	// Substitui referências cid: por data URIs
	if len(cidMap) > 0 {
		htmlContent = replaceCIDReferences(htmlContent, cidMap)
	}

	return htmlContent
}

// extractHTMLWithCID extrai HTML e mapa de imagens CID
func extractHTMLWithCID(rawData []byte) (string, map[string]string) {
	var cidMap = make(map[string]string)

	var msg, err = mail.ReadMessage(bytes.NewReader(rawData))
	if err != nil {
		return "", cidMap
	}

	var contentType = msg.Header.Get("Content-Type")
	var mediaType, params, _ = mime.ParseMediaType(contentType)

	// Se for HTML direto
	if strings.HasPrefix(mediaType, "text/html") {
		var body, _ = io.ReadAll(msg.Body)
		var charset = params["charset"]
		return decodeBodyWithCharset(body, msg.Header.Get("Content-Transfer-Encoding"), charset), cidMap
	}

	// Se for multipart, procura a parte HTML e imagens
	if strings.HasPrefix(mediaType, "multipart/") {
		var boundary = params["boundary"]
		if boundary != "" {
			return findHTMLAndImages(msg.Body, boundary, cidMap)
		}
	}

	return "", cidMap
}

// findHTMLAndImages procura HTML e extrai imagens embutidas
func findHTMLAndImages(r io.Reader, boundary string, cidMap map[string]string) (string, map[string]string) {
	var htmlContent string
	var mr = multipart.NewReader(r, boundary)

	// Primeira passagem: coleta todas as partes
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

	// Processa as partes
	for _, part := range parts {
		var mediaType, params, _ = mime.ParseMediaType(part.contentType)

		// HTML
		if strings.HasPrefix(mediaType, "text/html") && htmlContent == "" {
			var charset = params["charset"]
			htmlContent = decodeBodyWithCharset(part.body, part.encoding, charset)
		}

		// Imagens com Content-ID
		var contentID = part.contentID
		if contentID != "" && strings.HasPrefix(mediaType, "image/") {
			// Remove < > do Content-ID
			contentID = strings.Trim(contentID, "<>")

			// Decodifica o body da imagem
			var imageData = decodeImageBody(part.body, part.encoding)

			// Cria data URI
			var dataURI = fmt.Sprintf("data:%s;base64,%s", mediaType, base64.StdEncoding.EncodeToString(imageData))
			cidMap[contentID] = dataURI
		}

		// Multipart aninhado
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

// decodeImageBody decodifica o corpo de uma imagem
func decodeImageBody(body []byte, encoding string) []byte {
	switch strings.ToLower(encoding) {
	case "base64":
		var decoded, err = base64.StdEncoding.DecodeString(string(body))
		if err != nil {
			// Tenta remover espaços/newlines
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

// extractAttachments extrai todos os anexos de imagem de um email
func extractAttachments(rawData []byte) []Attachment {
	var attachments []Attachment

	var msg, err = mail.ReadMessage(bytes.NewReader(rawData))
	if err != nil {
		return attachments
	}

	var contentType = msg.Header.Get("Content-Type")
	var mediaType, params, _ = mime.ParseMediaType(contentType)

	// Se for multipart, procura anexos e imagens
	if strings.HasPrefix(mediaType, "multipart/") {
		var boundary = params["boundary"]
		if boundary != "" {
			attachments = findImageAttachments(msg.Body, boundary)
		}
	}

	return attachments
}

// findImageAttachments procura imagens (inline e anexos) no email
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

		// Verifica se é uma imagem (inline ou anexo)
		if strings.HasPrefix(mediaType, "image/") {
			var decoded = decodeImageBody(body, encoding)

			// Tenta obter o filename
			var filename = params["name"]
			if filename == "" {
				var _, dispParams, _ = mime.ParseMediaType(disposition)
				filename = dispParams["filename"]
			}
			if filename == "" && contentID != "" {
				filename = contentID
			}
			if filename == "" {
				// Gera nome baseado no tipo
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

		// Multipart aninhado (comum em emails com alternative + related)
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

// extractAllAttachments extrai todos os anexos de um email (não apenas imagens)
func extractAllAttachments(rawData []byte) []Attachment {
	var attachments []Attachment

	var msg, err = mail.ReadMessage(bytes.NewReader(rawData))
	if err != nil {
		return attachments
	}

	var contentType = msg.Header.Get("Content-Type")
	var mediaType, params, _ = mime.ParseMediaType(contentType)

	// Se for multipart, procura anexos
	if strings.HasPrefix(mediaType, "multipart/") {
		var boundary = params["boundary"]
		if boundary != "" {
			attachments = findAllAttachmentParts(msg.Body, boundary)
		}
	}

	return attachments
}

// findAllAttachmentParts procura todos os anexos no email (não apenas imagens)
func findAllAttachmentParts(r io.Reader, boundary string) []Attachment {
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

		// Determina se é um anexo baseado no Content-Disposition ou tipo
		var isAttachment = strings.HasPrefix(disposition, "attachment")
		var isInline = contentID != "" || strings.HasPrefix(disposition, "inline")

		// Pula partes text/plain e text/html que são o corpo do email (não anexos)
		var isBodyPart = (strings.HasPrefix(mediaType, "text/plain") || strings.HasPrefix(mediaType, "text/html")) && !isAttachment

		if !isBodyPart && (isAttachment || isInline || strings.HasPrefix(mediaType, "image/") ||
			strings.HasPrefix(mediaType, "application/") || strings.HasPrefix(mediaType, "audio/") ||
			strings.HasPrefix(mediaType, "video/")) {

			// Tenta obter o filename
			var filename = params["name"]
			if filename == "" {
				var _, dispParams, _ = mime.ParseMediaType(disposition)
				filename = dispParams["filename"]
			}
			if filename == "" && contentID != "" {
				filename = contentID
			}
			if filename == "" {
				// Gera nome baseado no tipo
				filename = generateFilename(mediaType)
			}

			// Decodifica o conteúdo
			var decoded = decodeAttachmentBody(body, encoding)

			attachments = append(attachments, Attachment{
				Filename:    filename,
				ContentType: mediaType,
				ContentID:   contentID,
				Size:        int64(len(decoded)),
				Data:        decoded,
				IsInline:    isInline && !isAttachment,
			})
		}

		// Multipart aninhado (comum em emails com alternative + related)
		if strings.HasPrefix(mediaType, "multipart/") {
			var nestedBoundary = params["boundary"]
			if nestedBoundary != "" {
				var nested = findAllAttachmentParts(bytes.NewReader(body), nestedBoundary)
				attachments = append(attachments, nested...)
			}
		}
	}

	return attachments
}

// generateFilename gera um nome de arquivo baseado no tipo MIME
func generateFilename(mediaType string) string {
	var ext = "bin"
	switch mediaType {
	case "image/jpeg":
		ext = "jpg"
	case "image/png":
		ext = "png"
	case "image/gif":
		ext = "gif"
	case "image/webp":
		ext = "webp"
	case "application/pdf":
		ext = "pdf"
	case "application/zip", "application/x-zip-compressed":
		ext = "zip"
	case "application/msword":
		ext = "doc"
	case "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		ext = "docx"
	case "application/vnd.ms-excel":
		ext = "xls"
	case "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":
		ext = "xlsx"
	case "text/plain":
		ext = "txt"
	case "text/csv":
		ext = "csv"
	case "audio/mpeg":
		ext = "mp3"
	case "video/mp4":
		ext = "mp4"
	}
	return fmt.Sprintf("attachment.%s", ext)
}

// decodeAttachmentBody decodifica o corpo de um anexo
func decodeAttachmentBody(body []byte, encoding string) []byte {
	switch strings.ToLower(encoding) {
	case "base64":
		var decoded, err = base64.StdEncoding.DecodeString(strings.TrimSpace(string(body)))
		if err != nil {
			// Tenta com tolerância a quebras de linha
			var cleaned = strings.ReplaceAll(string(body), "\n", "")
			cleaned = strings.ReplaceAll(cleaned, "\r", "")
			decoded, err = base64.StdEncoding.DecodeString(strings.TrimSpace(cleaned))
			if err != nil {
				return body
			}
		}
		return decoded
	case "quoted-printable":
		var decoded, err = io.ReadAll(quotedprintable.NewReader(bytes.NewReader(body)))
		if err != nil {
			return body
		}
		return decoded
	default:
		return body
	}
}

// getAttachmentIcon retorna um ícone baseado no tipo do anexo
func getAttachmentIcon(contentType string) string {
	switch {
	case contentType == "application/pdf":
		return "📄"
	case contentType == "application/zip" || contentType == "application/x-zip-compressed":
		return "📦"
	case strings.HasPrefix(contentType, "application/vnd.ms-word") || strings.HasPrefix(contentType, "application/vnd.openxmlformats-officedocument.word"):
		return "📝"
	case strings.HasPrefix(contentType, "application/vnd.ms-excel") || strings.HasPrefix(contentType, "application/vnd.openxmlformats-officedocument.spreadsheet"):
		return "📊"
	case strings.HasPrefix(contentType, "application/vnd.ms-powerpoint") || strings.HasPrefix(contentType, "application/vnd.openxmlformats-officedocument.presentation"):
		return "📽"
	case strings.HasPrefix(contentType, "image/"):
		return "🖼"
	case strings.HasPrefix(contentType, "video/"):
		return "🎬"
	case strings.HasPrefix(contentType, "audio/"):
		return "🎵"
	case strings.HasPrefix(contentType, "text/"):
		return "📃"
	default:
		return "📎"
	}
}

// renderAttachmentList renders a list of attachments for display in the email viewer
func renderAttachmentList(attachments []Attachment) string {
	if len(attachments) == 0 {
		return ""
	}

	var lines []string
	lines = append(lines, "─────────────────────────────────────────")
	lines = append(lines, fmt.Sprintf("📎 Anexos (%d)", len(attachments)))
	lines = append(lines, "")

	for _, att := range attachments {
		var icon = getAttachmentIcon(att.ContentType)
		var size = formatAttachmentSize(att.Size)
		var inline = ""
		if att.IsInline {
			inline = " (inline)"
		}
		lines = append(lines, fmt.Sprintf("  %s %s  %s%s", icon, att.Filename, size, inline))
	}

	lines = append(lines, "")
	lines = append(lines, "Pressione 'x' para baixar anexos")

	return strings.Join(lines, "\n")
}

// formatAttachmentSize formata o tamanho do anexo
func formatAttachmentSize(size int64) string {
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

// replaceCIDReferences substitui cid:xxx por data URIs
func replaceCIDReferences(html string, cidMap map[string]string) string {
	// Padrão: src="cid:xxx" ou src='cid:xxx'
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

func findHTMLPart(r io.Reader, boundary string) string {
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
			return decodeBody(body, part.Header.Get("Content-Transfer-Encoding"))
		}

		// Multipart aninhado
		if strings.HasPrefix(mediaType, "multipart/") {
			var boundary = params["boundary"]
			if boundary != "" {
				if html := findHTMLPart(part, boundary); html != "" {
					return html
				}
			}
		}
	}
	return ""
}

func decodeBody(body []byte, encoding string) string {
	return decodeBodyWithCharset(body, encoding, "")
}

func decodeBodyWithCharset(body []byte, encoding string, charset string) string {
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
			// Tenta limpar
			var cleaned = strings.ReplaceAll(string(body), "\n", "")
			cleaned = strings.ReplaceAll(cleaned, "\r", "")
			d, _ = base64.StdEncoding.DecodeString(cleaned)
		}
		decoded = d
	default:
		decoded = body
	}

	// Converte charset se necessário
	if charset != "" && !strings.EqualFold(charset, "utf-8") && !strings.EqualFold(charset, "us-ascii") {
		var converted = convertCharset(decoded, charset)
		if converted != "" {
			return converted
		}
	}

	return string(decoded)
}

// convertCharset converte de um charset para UTF-8
func convertCharset(data []byte, charset string) string {
	// Tenta usar htmlindex primeiro
	var enc, err = htmlindex.Get(charset)
	if err == nil {
		var decoder = enc.NewDecoder()
		var result, err2 = decoder.Bytes(data)
		if err2 == nil {
			return string(result)
		}
	}

	// Fallback para charsets comuns
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

// htmlToText converte HTML para texto legível
func htmlToText(htmlContent string) string {
	var doc, err = html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return htmlContent
	}

	var buf bytes.Buffer
	var extractTextFromNode func(*html.Node)
	extractTextFromNode = func(n *html.Node) {
		// Ignora scripts, styles e comments
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

		// Adiciona quebra de linha após elementos de bloco
		if n.Type == html.ElementNode {
			switch n.Data {
			case "p", "div", "tr", "li", "h1", "h2", "h3", "h4", "h5", "h6", "blockquote":
				buf.WriteString("\n")
			}
		}
	}

	extractTextFromNode(doc)

	// Limpa múltiplas linhas em branco
	var result = buf.String()
	for strings.Contains(result, "\n\n\n") {
		result = strings.ReplaceAll(result, "\n\n\n", "\n\n")
	}
	return strings.TrimSpace(result)
}
