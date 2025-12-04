package main

import (
	"fmt"
	"log"

	"github.com/opik/miau/internal/config"
	"github.com/opik/miau/internal/imap"
	"github.com/opik/miau/internal/storage"
)

func main() {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	if len(cfg.Accounts) == 0 {
		log.Fatal("No accounts configured")
	}

	account := &cfg.Accounts[0]
	fmt.Printf("Testing with account: %s\n", account.Email)

	// Init storage
	if err := storage.Init(cfg.Storage.Database); err != nil {
		log.Fatal("Failed to init storage:", err)
	}

	// Connect to IMAP
	client, err := imap.Connect(account)
	if err != nil {
		log.Fatal("Failed to connect to IMAP:", err)
	}
	defer client.Close()

	fmt.Println("Connected to IMAP")

	// Select INBOX
	_, err = client.SelectMailbox("INBOX")
	if err != nil {
		log.Fatal("Failed to select INBOX:", err)
	}

	// Get some emails from DB to test
	emails, err := storage.GetEmails(1, 1, 20, 0)
	if err != nil {
		log.Fatal("Failed to get emails:", err)
	}

	fmt.Printf("Testing %d emails for attachments...\n\n", len(emails))

	var foundAttachments = 0
	for _, email := range emails {
		attachments, hasAttachments, err := client.FetchAttachmentMetadata(email.UID)
		if err != nil {
			fmt.Printf("Email %d (UID %d): ERROR - %v\n", email.ID, email.UID, err)
			continue
		}

		if hasAttachments {
			fmt.Printf("Email %d (UID %d): %s\n", email.ID, email.UID, email.Subject)
			fmt.Printf("  Found %d attachments:\n", len(attachments))
			for _, att := range attachments {
				fmt.Printf("    - %s (%s, %d bytes, part %s)\n", att.Filename, att.ContentType, att.Size, att.PartNumber)
			}
			fmt.Println()
			foundAttachments++
		}
	}

	fmt.Printf("\nFound %d emails with attachments out of %d tested\n", foundAttachments, len(emails))
}
