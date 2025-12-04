//go:build ignore

package main

import (
    "bytes"
    "fmt"
    "io"
    "mime"
    "mime/multipart"
    "net/mail"
    "os"
    "path/filepath"
    "strings"

    "github.com/opik/miau/internal/config"
    "github.com/opik/miau/internal/imap"
)

func main() {
    // Load config
    config.Init()
    accounts := config.GetAccounts()
    if len(accounts) == 0 {
        fmt.Println("No accounts configured")
        return
    }
    
    acc := accounts[0]
    fmt.Printf("Account: %s\n", acc.Email)
    
    // Load OAuth tokens if needed
    tokensDir := filepath.Join(os.Getenv("HOME"), ".config/miau/tokens")
    client, err := imap.NewClient(acc.IMAP.Host, acc.IMAP.Port, acc.Email, acc.Password, acc.AuthType, acc.IMAP.UseTLS, tokensDir)
    if err != nil {
        fmt.Println("Error creating client:", err)
        return
    }
    defer client.Close()
    
    if err := client.Connect(); err != nil {
        fmt.Println("Error connecting:", err)
        return
    }
    
    if _, err := client.SelectMailbox("INBOX"); err != nil {
        fmt.Println("Error selecting INBOX:", err)
        return
    }
    
    // Fetch email UID 315001
    uid := uint32(315001)
    fmt.Printf("Fetching email UID %d...\n", uid)
    
    rawData, err := client.FetchEmailRaw(uid)
    if err != nil {
        fmt.Println("Error fetching:", err)
        return
    }
    
    fmt.Printf("Got %d bytes of raw data\n", len(rawData))
    
    // Parse email
    msg, err := mail.ReadMessage(bytes.NewReader(rawData))
    if err != nil {
        fmt.Println("Error parsing:", err)
        return
    }
    
    contentType := msg.Header.Get("Content-Type")
    fmt.Printf("Content-Type: %s\n", contentType)
    
    mediaType, params, _ := mime.ParseMediaType(contentType)
    fmt.Printf("Media type: %s\n", mediaType)
    
    if strings.HasPrefix(mediaType, "multipart/") {
        boundary := params["boundary"]
        fmt.Printf("Boundary: %s\n", boundary)
        
        if boundary != "" {
            mr := multipart.NewReader(msg.Body, boundary)
            partNum := 0
            for {
                part, err := mr.NextPart()
                if err != nil {
                    break
                }
                partNum++
                
                partCT := part.Header.Get("Content-Type")
                partDisp := part.Header.Get("Content-Disposition")
                body, _ := io.ReadAll(part)
                
                fmt.Printf("\nPart %d:\n", partNum)
                fmt.Printf("  Content-Type: %s\n", partCT)
                fmt.Printf("  Content-Disposition: %s\n", partDisp)
                fmt.Printf("  Size: %d bytes\n", len(body))
                
                // Check for nested multipart
                partMediaType, partParams, _ := mime.ParseMediaType(partCT)
                if strings.HasPrefix(partMediaType, "multipart/") {
                    fmt.Printf("  (nested multipart: %s)\n", partMediaType)
                    nestedBoundary := partParams["boundary"]
                    if nestedBoundary != "" {
                        nestedMR := multipart.NewReader(bytes.NewReader(body), nestedBoundary)
                        nestedPartNum := 0
                        for {
                            nestedPart, err := nestedMR.NextPart()
                            if err != nil {
                                break
                            }
                            nestedPartNum++
                            nestedBody, _ := io.ReadAll(nestedPart)
                            nestedCT := nestedPart.Header.Get("Content-Type")
                            nestedDisp := nestedPart.Header.Get("Content-Disposition")
                            fmt.Printf("    Nested part %d.%d:\n", partNum, nestedPartNum)
                            fmt.Printf("      Content-Type: %s\n", nestedCT)
                            fmt.Printf("      Content-Disposition: %s\n", nestedDisp)
                            fmt.Printf("      Size: %d bytes\n", len(nestedBody))
                        }
                    }
                }
            }
        }
    }
}
