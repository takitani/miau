# AI-12: AI Translation

## Overview

Automatically translate emails from foreign languages to user's preferred language.

## User Stories

1. As a user, I want foreign language emails automatically translated
2. As a user, I want to toggle between original and translated text
3. As a user, I want to compose emails in another language
4. As a user, I want to set preferred reading language

## Technical Requirements

### Service Layer

Create `internal/services/translation.go`:

```go
package services

type TranslationService interface {
    // TranslateEmail translates email content
    TranslateEmail(ctx context.Context, emailID int64, targetLang string) (*Translation, error)

    // TranslateText translates arbitrary text
    TranslateText(ctx context.Context, text string, sourceLang, targetLang string) (string, error)

    // DetectLanguage detects the language of text
    DetectLanguage(ctx context.Context, text string) (*LanguageDetection, error)

    // GetTranslation retrieves cached translation
    GetTranslation(ctx context.Context, emailID int64, targetLang string) (*Translation, error)

    // TranslateCompose translates draft before sending
    TranslateCompose(ctx context.Context, text string, targetLang string) (string, error)
}

type Translation struct {
    EmailID       int64
    SourceLang    string
    TargetLang    string
    OriginalText  string
    TranslatedText string
    Confidence    float64
    CreatedAt     time.Time
}

type LanguageDetection struct {
    Language   string
    Confidence float64
    IsReliable bool
}

// Supported languages
var SupportedLanguages = []string{
    "en", "es", "fr", "de", "it", "pt", "zh", "ja", "ko", "ru", "ar",
}
```

### Database Schema

```sql
CREATE TABLE email_translations (
    id INTEGER PRIMARY KEY,
    email_id INTEGER REFERENCES emails(id),
    source_lang TEXT NOT NULL,
    target_lang TEXT NOT NULL,
    original_subject TEXT,
    translated_subject TEXT,
    original_body TEXT,
    translated_body TEXT,
    confidence REAL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(email_id, target_lang)
);
```

### AI Prompt Template

```go
var translationPrompt = `Translate this email from {{.SourceLang}} to {{.TargetLang}}.

Subject: {{.Subject}}
Body:
{{.Body}}

Requirements:
- Preserve formatting (paragraphs, lists)
- Maintain professional tone
- Keep proper nouns unchanged
- Translate dates/times to target locale

Output JSON:
{
  "translated_subject": "...",
  "translated_body": "...",
  "notes": "Any translation notes or clarifications"
}`
```

## UI/UX

### TUI
- Auto-detect non-native language emails
- Press `T` to toggle translation
- Translation indicator in email list

```
┌─ Email (Translated: Spanish → English) ───────────────────────────┐
│ From: cliente@empresa.es                                          │
│ Subject: Meeting confirmation (Confirmación de reunión)           │
├───────────────────────────────────────────────────────────────────┤
│ Good morning,                                                     │
│                                                                   │
│ I confirm our meeting for tomorrow at 3pm.                        │
│ Please let me know if you need anything else.                     │
│                                                                   │
│ Best regards,                                                     │
│ Carlos                                                            │
├───────────────────────────────────────────────────────────────────┤
│ [T] Toggle Original  [R] Reply in Spanish                         │
└───────────────────────────────────────────────────────────────────┘
```

### Desktop
- Translation banner with toggle
- Language selector in compose
- Settings for auto-translate
- Side-by-side view option

## Testing

1. Test translation accuracy (spot check)
2. Test language detection
3. Test cache behavior
4. Test with multiple languages
5. Test compose translation
6. Test with HTML emails

## Acceptance Criteria

- [ ] Detects non-native language emails
- [ ] Auto-translates to preferred language
- [ ] Toggle shows original text
- [ ] Translations cached
- [ ] Can compose in other languages
- [ ] Preserves email formatting
- [ ] Works in TUI and Desktop

## Configuration

```yaml
# config.yaml
translation:
  enabled: true
  auto_translate: true
  preferred_language: "en"
  show_original: false
```

## Estimated Complexity

Medium - AI integration with caching
