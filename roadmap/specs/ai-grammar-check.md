# AI-13: AI Grammar Check

## Overview

Check and correct grammar, spelling, and style in composed emails before sending.

## User Stories

1. As a user, I want grammar errors highlighted in my draft
2. As a user, I want suggestions for improving clarity
3. As a user, I want to check tone (formal/casual)
4. As a user, I want one-click corrections

## Technical Requirements

### Service Layer

Create `internal/services/grammarcheck.go`:

```go
package services

type GrammarCheckService interface {
    // CheckGrammar analyzes text for errors
    CheckGrammar(ctx context.Context, text string, opts GrammarOptions) (*GrammarResult, error)

    // ApplyCorrection applies a suggested correction
    ApplyCorrection(ctx context.Context, text string, correction Correction) (string, error)

    // ApplyAllCorrections applies all corrections
    ApplyAllCorrections(ctx context.Context, text string, result *GrammarResult) (string, error)

    // CheckTone analyzes writing tone
    CheckTone(ctx context.Context, text string, desiredTone string) (*ToneAnalysis, error)
}

type GrammarOptions struct {
    Language    string
    CheckSpelling bool
    CheckGrammar  bool
    CheckStyle    bool
    CheckTone     bool
    DesiredTone   string
}

type GrammarResult struct {
    Text        string
    Issues      []GrammarIssue
    Score       float64  // 0-100 overall quality
    ToneMatch   bool
    Suggestions []StyleSuggestion
}

type GrammarIssue struct {
    Type        IssueType
    Message     string
    Offset      int
    Length      int
    Original    string
    Suggestions []string
    Severity    string  // "error", "warning", "info"
}

type IssueType string

const (
    IssueSpelling IssueType = "spelling"
    IssueGrammar  IssueType = "grammar"
    IssuePunct    IssueType = "punctuation"
    IssueStyle    IssueType = "style"
    IssueTone     IssueType = "tone"
)

type StyleSuggestion struct {
    Original    string
    Suggestion  string
    Reason      string
}

type ToneAnalysis struct {
    CurrentTone   string
    DesiredTone   string
    Match         bool
    Suggestions   []string
}
```

### AI Prompt Template

```go
var grammarPrompt = `Check this email draft for grammar, spelling, and style.

Text:
{{.Text}}

Language: {{.Language}}
Desired tone: {{.DesiredTone}}

Find:
1. Spelling errors
2. Grammar mistakes
3. Punctuation issues
4. Style improvements
5. Tone mismatches

Output JSON:
{
  "score": 85,
  "issues": [
    {
      "type": "spelling",
      "message": "Misspelled word",
      "offset": 45,
      "length": 8,
      "original": "recieved",
      "suggestions": ["received"],
      "severity": "error"
    }
  ],
  "tone_match": true,
  "style_suggestions": [
    {
      "original": "I want to",
      "suggestion": "I would like to",
      "reason": "More formal tone"
    }
  ]
}`
```

## UI/UX

### TUI
- Check triggered before send
- Underline errors in compose
- Tab through suggestions

```
┌─ Compose (3 issues found) ────────────────────────────────────────┐
│ To: boss@company.com                                              │
│ Subject: Project Update                                           │
├───────────────────────────────────────────────────────────────────┤
│ Hi,                                                               │
│                                                                   │
│ I recieved your email yesterday. The project is going good.       │
│     ~~~~~~~~                                          ~~~~        │
│     [received]                                        [well]      │
│                                                                   │
│ Let me know if you need more infomation.                          │
│                               ~~~~~~~~~~                          │
│                               [information]                       │
├───────────────────────────────────────────────────────────────────┤
│ [Tab] Next issue  [Enter] Apply fix  [a] Apply all  [i] Ignore    │
└───────────────────────────────────────────────────────────────────┘
```

### Desktop
- Real-time underlines as you type
- Right-click for suggestions
- Grammar score indicator
- Tone selector dropdown

## Testing

1. Test common spelling errors
2. Test grammar detection
3. Test punctuation rules
4. Test tone analysis
5. Test multilingual support
6. Test with long emails

## Acceptance Criteria

- [ ] Detects spelling errors
- [ ] Detects grammar issues
- [ ] Suggests corrections
- [ ] One-click fix works
- [ ] Can fix all at once
- [ ] Shows quality score
- [ ] Checks tone appropriateness
- [ ] Works in TUI and Desktop

## Configuration

```yaml
# config.yaml
compose:
  grammar_check:
    enabled: true
    check_on_send: true
    language: "en"
    check_spelling: true
    check_grammar: true
    check_tone: true
```

## Estimated Complexity

Medium - AI text analysis
