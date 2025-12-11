# PL-04: CLI Commands (miau ls, send)

## Overview

Add command-line subcommands for common email operations without starting the TUI.

## User Stories

1. As a user, I want to check emails from a script
2. As a user, I want to send emails from the command line
3. As a user, I want to search emails without opening TUI
4. As a user, I want to use miau in automated workflows

## Technical Requirements

### Command Structure

```bash
miau                      # Start TUI (default)
miau ls                   # List recent emails
miau ls --unread          # List unread emails
miau ls --from john       # Filter by sender
miau read <id>            # Show email content
miau send                 # Interactive send
miau send --to <email> --subject <subj> --body <body>
miau send --file email.eml
miau search <query>       # Search emails
miau sync                 # Sync emails
miau archive <id>         # Archive email
miau delete <id>          # Delete email
miau folders              # List folders
miau contacts             # List contacts
```

### Implementation with Cobra

```go
package cmd

import (
    "github.com/spf13/cobra"
    "github.com/takitani/miau/internal/app"
)

var rootCmd = &cobra.Command{
    Use:   "miau",
    Short: "Mail Intelligence Assistant Utility",
    Run:   runTUI,  // Default behavior
}

var lsCmd = &cobra.Command{
    Use:   "ls",
    Short: "List emails",
    Run:   runList,
}

var sendCmd = &cobra.Command{
    Use:   "send",
    Short: "Send an email",
    Run:   runSend,
}

var readCmd = &cobra.Command{
    Use:   "read <email-id>",
    Short: "Read an email",
    Args:  cobra.ExactArgs(1),
    Run:   runRead,
}

var searchCmd = &cobra.Command{
    Use:   "search <query>",
    Short: "Search emails",
    Args:  cobra.ExactArgs(1),
    Run:   runSearch,
}

func init() {
    // ls flags
    lsCmd.Flags().BoolP("unread", "u", false, "Show only unread")
    lsCmd.Flags().StringP("from", "f", "", "Filter by sender")
    lsCmd.Flags().StringP("folder", "F", "INBOX", "Folder to list")
    lsCmd.Flags().IntP("limit", "n", 20, "Number of emails")
    lsCmd.Flags().StringP("format", "o", "table", "Output format (table, json, csv)")

    // send flags
    sendCmd.Flags().StringP("to", "t", "", "Recipient email")
    sendCmd.Flags().StringP("cc", "c", "", "CC recipients")
    sendCmd.Flags().StringP("subject", "s", "", "Email subject")
    sendCmd.Flags().StringP("body", "b", "", "Email body")
    sendCmd.Flags().StringP("file", "F", "", "Send from .eml file")
    sendCmd.Flags().BoolP("html", "H", false, "Body is HTML")

    // search flags
    searchCmd.Flags().IntP("limit", "n", 20, "Number of results")
    searchCmd.Flags().StringP("format", "o", "table", "Output format")

    rootCmd.AddCommand(lsCmd, sendCmd, readCmd, searchCmd)
}
```

### Command Implementations

```go
func runList(cmd *cobra.Command, args []string) {
    app, err := initApp()
    if err != nil {
        log.Fatal(err)
    }

    unread, _ := cmd.Flags().GetBool("unread")
    from, _ := cmd.Flags().GetString("from")
    folder, _ := cmd.Flags().GetString("folder")
    limit, _ := cmd.Flags().GetInt("limit")
    format, _ := cmd.Flags().GetString("format")

    ctx := context.Background()
    emails, err := app.Email().GetEmails(ctx, accountID, folder, limit, 0)
    if err != nil {
        log.Fatal(err)
    }

    // Apply filters
    if unread {
        emails = filterUnread(emails)
    }
    if from != "" {
        emails = filterByFrom(emails, from)
    }

    // Output
    switch format {
    case "json":
        outputJSON(emails)
    case "csv":
        outputCSV(emails)
    default:
        outputTable(emails)
    }
}

func runSend(cmd *cobra.Command, args []string) {
    app, err := initApp()
    if err != nil {
        log.Fatal(err)
    }

    to, _ := cmd.Flags().GetString("to")
    subject, _ := cmd.Flags().GetString("subject")
    body, _ := cmd.Flags().GetString("body")
    file, _ := cmd.Flags().GetString("file")

    ctx := context.Background()

    var email *Email
    if file != "" {
        // Parse .eml file
        email, err = parseEMLFile(file)
    } else if to == "" {
        // Interactive mode
        email = interactiveSend()
    } else {
        email = &Email{
            To:      to,
            Subject: subject,
            Body:    body,
        }
    }

    err = app.Send().SendEmail(ctx, email)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("Email sent successfully")
}

func runSearch(cmd *cobra.Command, args []string) {
    app, err := initApp()
    if err != nil {
        log.Fatal(err)
    }

    query := args[0]
    limit, _ := cmd.Flags().GetInt("limit")
    format, _ := cmd.Flags().GetString("format")

    ctx := context.Background()
    results, err := app.Search().Search(ctx, accountID, query, limit)
    if err != nil {
        log.Fatal(err)
    }

    switch format {
    case "json":
        outputJSON(results)
    default:
        outputTable(results)
    }
}
```

## Output Formats

### Table (default)

```
$ miau ls --limit 5
ID     FROM                SUBJECT                      DATE
1234   john@example.com    Project Update               Dec 15, 10:30
1233   newsletter@tech.co  Weekly Digest                Dec 14, 08:00
1232   amazon@amazon.com   Your order shipped           Dec 13, 14:15
1231   boss@company.com    Meeting tomorrow             Dec 13, 09:00
1230   client@external.co  Contract review              Dec 12, 16:45
```

### JSON

```bash
$ miau ls --format json | jq '.[] | {from, subject}'
{"from": "john@example.com", "subject": "Project Update"}
{"from": "newsletter@tech.co", "subject": "Weekly Digest"}
```

### CSV

```bash
$ miau ls --format csv > emails.csv
```

## Shell Integration

```bash
# Pipe email body
echo "Hello, this is a test" | miau send -t john@example.com -s "Test"

# Read from file
miau send -t john@example.com -s "Report" < report.txt

# Use in scripts
UNREAD_COUNT=$(miau ls --unread --format json | jq length)
if [ "$UNREAD_COUNT" -gt 0 ]; then
    notify-send "You have $UNREAD_COUNT unread emails"
fi

# Cron job sync
0 */15 * * * miau sync >> /var/log/miau-sync.log
```

## Testing

1. Test each subcommand
2. Test output formats
3. Test piping and stdin
4. Test error handling
5. Test with scripts

## Acceptance Criteria

- [ ] `miau ls` lists emails
- [ ] `miau read <id>` shows email
- [ ] `miau send` sends email
- [ ] `miau search` searches emails
- [ ] JSON output works
- [ ] Piping works
- [ ] Works in scripts
- [ ] Good error messages

## Estimated Complexity

Medium - Cobra commands wrapping existing services
