package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/oauth2"
)

// Basecamp OAuth2 endpoints
var BasecampEndpoint = oauth2.Endpoint{
	AuthURL:  "https://launchpad.37signals.com/authorization/new",
	TokenURL: "https://launchpad.37signals.com/authorization/token",
}

// BasecampAccount represents a Basecamp account from the authorization response
type BasecampAccount struct {
	Product string `json:"product"` // "bc3" for Basecamp 3/4
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Href    string `json:"href"`    // API base URL for this account
	AppHref string `json:"app_href"` // Web URL
}

// BasecampIdentity represents the user identity from authorization response
type BasecampIdentity struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email_address"`
}

// BasecampAuthResponse is the response from /authorization.json
type BasecampAuthResponse struct {
	ExpiresAt time.Time          `json:"expires_at"`
	Identity  BasecampIdentity   `json:"identity"`
	Accounts  []BasecampAccount  `json:"accounts"`
}

// GetBasecampOAuth2Config creates OAuth2 config for Basecamp
func GetBasecampOAuth2Config(clientID, clientSecret string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     BasecampEndpoint,
		RedirectURL:  "http://localhost:8089/callback",
		// Basecamp doesn't use scopes - access is granted to all products the user authorizes
	}
}

// AuthenticateBasecampWithBrowser initiates the OAuth2 flow for Basecamp
func AuthenticateBasecampWithBrowser(cfg *oauth2.Config) (*oauth2.Token, error) {
	var codeChan = make(chan string)
	var errChan = make(chan error)

	var server = &http.Server{Addr: ":8089"}

	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		var code = r.URL.Query().Get("code")
		if code == "" {
			errChan <- fmt.Errorf("authorization code not received")
			fmt.Fprintf(w, "<html><body><h1>Error!</h1><p>Code not received. Close this window.</p></body></html>")
			return
		}

		fmt.Fprintf(w, `<html><body style="font-family: sans-serif; display: flex; justify-content: center; align-items: center; height: 100vh; margin: 0; background: #1a1a2e;">
			<div style="text-align: center; color: #eee;">
				<h1 style="color: #4CAF50;">miau + Basecamp</h1>
				<p style="color: #73D216; font-size: 1.5em;">✓ Basecamp authentication complete!</p>
				<p>You can close this window and return to miau.</p>
			</div>
		</body></html>`)
		codeChan <- code
	})

	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	// Basecamp requires type=web_server parameter
	var authURL = cfg.AuthCodeURL("state-token",
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("type", "web_server"),
	)

	fmt.Println("\n🔐 Opening browser for Basecamp authentication...")
	fmt.Println("If it doesn't open automatically, visit:")
	fmt.Println(authURL)

	if err := openBrowser(authURL); err != nil {
		fmt.Printf("Could not open browser: %v\n", err)
	}

	var code string
	select {
	case code = <-codeChan:
		// OK
	case err := <-errChan:
		server.Shutdown(context.Background())
		return nil, err
	case <-time.After(5 * time.Minute):
		server.Shutdown(context.Background())
		return nil, fmt.Errorf("timeout waiting for authorization")
	}

	server.Shutdown(context.Background())

	// Exchange code for token - Basecamp requires type=web_server
	var ctx = context.Background()
	var token, err = cfg.Exchange(ctx, code,
		oauth2.SetAuthURLParam("type", "web_server"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code for token: %w", err)
	}

	return token, nil
}

// GetBasecampTokenPath returns the path for storing Basecamp token
func GetBasecampTokenPath(configDir string) string {
	return filepath.Join(configDir, "tokens", "basecamp.json")
}

// SaveBasecampToken saves the Basecamp token to file
func SaveBasecampToken(path string, token *oauth2.Token) error {
	var dir = filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	var data, err = json.MarshalIndent(token, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

// LoadBasecampToken loads the Basecamp token from file
func LoadBasecampToken(path string) (*oauth2.Token, error) {
	var data, err = os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var token = &oauth2.Token{}
	if err := json.Unmarshal(data, token); err != nil {
		return nil, err
	}

	return token, nil
}

// GetValidBasecampToken returns a valid token, refreshing if necessary
func GetValidBasecampToken(cfg *oauth2.Config, tokenPath string) (*oauth2.Token, error) {
	var token, err = LoadBasecampToken(tokenPath)
	if err != nil {
		return nil, err
	}

	// If token expired, try to refresh
	if token.Expiry.Before(time.Now()) {
		var ctx = context.Background()
		var tokenSource = cfg.TokenSource(ctx, token)
		var newToken, err = tokenSource.Token()
		if err != nil {
			return nil, fmt.Errorf("failed to refresh token: %w", err)
		}

		if err := SaveBasecampToken(tokenPath, newToken); err != nil {
			return nil, err
		}

		return newToken, nil
	}

	return token, nil
}

// GetBasecampAccounts fetches the user's Basecamp accounts using the token
func GetBasecampAccounts(token *oauth2.Token) (*BasecampAuthResponse, error) {
	var client = &http.Client{}

	var req, err = http.NewRequest("GET", "https://launchpad.37signals.com/authorization.json", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req.Header.Set("User-Agent", "miau (https://github.com/opik/miau)")

	var resp, respErr = client.Do(req)
	if respErr != nil {
		return nil, respErr
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get accounts: status %d", resp.StatusCode)
	}

	var authResp BasecampAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return nil, err
	}

	return &authResp, nil
}

// GetBasecampAccountByID finds a specific account by ID from the auth response
func GetBasecampAccountByID(accounts []BasecampAccount, accountID int64) *BasecampAccount {
	for _, acc := range accounts {
		if acc.ID == accountID && acc.Product == "bc3" {
			return &acc
		}
	}
	return nil
}
