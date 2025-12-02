package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// Scopes necessários para IMAP
var GmailScopes = []string{
	"https://mail.google.com/", // Full access for IMAP
}

type OAuth2Config struct {
	ClientID     string `json:"client_id" yaml:"client_id"`
	ClientSecret string `json:"client_secret" yaml:"client_secret"`
}

type TokenInfo struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	Expiry       time.Time `json:"expiry"`
}

func GetOAuth2Config(clientID, clientSecret string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     google.Endpoint,
		Scopes:       GmailScopes,
		RedirectURL:  "http://localhost:8089/callback",
	}
}

// AuthenticateWithBrowser inicia o fluxo OAuth2 abrindo o navegador
func AuthenticateWithBrowser(cfg *oauth2.Config) (*oauth2.Token, error) {
	// Canal para receber o código de autorização
	var codeChan = make(chan string)
	var errChan = make(chan error)

	// Servidor HTTP temporário para callback
	var server = &http.Server{Addr: ":8089"}

	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		var code = r.URL.Query().Get("code")
		if code == "" {
			errChan <- fmt.Errorf("código de autorização não recebido")
			fmt.Fprintf(w, "<html><body><h1>Erro!</h1><p>Código não recebido. Feche esta janela.</p></body></html>")
			return
		}

		fmt.Fprintf(w, `<html><body style="font-family: sans-serif; display: flex; justify-content: center; align-items: center; height: 100vh; margin: 0; background: #1a1a2e;">
			<div style="text-align: center; color: #eee;">
				<h1 style="color: #FF6B6B;">miau 🐱</h1>
				<p style="color: #73D216; font-size: 1.5em;">✓ Autenticação concluída!</p>
				<p>Pode fechar esta janela e voltar ao terminal.</p>
			</div>
		</body></html>`)
		codeChan <- code
	})

	// Inicia servidor em goroutine
	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	// Gera URL de autorização
	var authURL = cfg.AuthCodeURL("state-token", oauth2.AccessTypeOffline, oauth2.ApprovalForce)

	fmt.Println("\n🔐 Abrindo navegador para autenticação...")
	fmt.Println("Se não abrir automaticamente, acesse:")
	fmt.Println(authURL)

	// Abre navegador
	if err := openBrowser(authURL); err != nil {
		fmt.Printf("Não foi possível abrir o navegador: %v\n", err)
	}

	// Aguarda código ou erro
	var code string
	select {
	case code = <-codeChan:
		// OK
	case err := <-errChan:
		server.Shutdown(context.Background())
		return nil, err
	case <-time.After(5 * time.Minute):
		server.Shutdown(context.Background())
		return nil, fmt.Errorf("timeout aguardando autorização")
	}

	// Encerra servidor
	server.Shutdown(context.Background())

	// Troca código por token
	var ctx = context.Background()
	var token, err = cfg.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("erro ao trocar código por token: %w", err)
	}

	return token, nil
}

func openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	case "darwin":
		cmd = "open"
		args = []string{url}
	default: // linux
		cmd = "xdg-open"
		args = []string{url}
	}

	return exec.Command(cmd, args...).Start()
}

// GetTokenPath retorna o caminho do arquivo de token para uma conta
func GetTokenPath(configDir, accountName string) string {
	return filepath.Join(configDir, "tokens", accountName+".json")
}

// SaveToken salva o token em arquivo
func SaveToken(path string, token *oauth2.Token) error {
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

// LoadToken carrega o token de arquivo
func LoadToken(path string) (*oauth2.Token, error) {
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

// GetValidToken retorna um token válido, renovando se necessário
func GetValidToken(cfg *oauth2.Config, tokenPath string) (*oauth2.Token, error) {
	var token, err = LoadToken(tokenPath)
	if err != nil {
		return nil, err
	}

	// Se token expirou, tenta renovar
	if token.Expiry.Before(time.Now()) {
		var ctx = context.Background()
		var tokenSource = cfg.TokenSource(ctx, token)
		var newToken, err = tokenSource.Token()
		if err != nil {
			return nil, fmt.Errorf("erro ao renovar token: %w", err)
		}

		// Salva token renovado
		if err := SaveToken(tokenPath, newToken); err != nil {
			return nil, err
		}

		return newToken, nil
	}

	return token, nil
}

// GenerateXOAuth2String gera a string de autenticação XOAUTH2 para IMAP
func GenerateXOAuth2String(email, accessToken string) string {
	var authStr = fmt.Sprintf("user=%s\x01auth=Bearer %s\x01\x01", email, accessToken)
	return authStr
}
