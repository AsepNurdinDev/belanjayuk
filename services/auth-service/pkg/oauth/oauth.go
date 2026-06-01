package oauth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// =============================================================
// Google OAuth Client
// =============================================================

const googleUserInfoURL = "https://www.googleapis.com/oauth2/v2/userinfo"

type GoogleClient struct {
	config      *oauth2.Config
	stateSecret string // secret untuk HMAC-SHA256 state verification
}

func NewGoogleClient(clientID, clientSecret, redirectURL string) *GoogleClient {
	return &GoogleClient{
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes: []string{
				"https://www.googleapis.com/auth/userinfo.email",
				"https://www.googleapis.com/auth/userinfo.profile",
			},
			Endpoint: google.Endpoint,
		},
	}
}

// WithStateSecret — set HMAC secret untuk state verification
func (g *GoogleClient) WithStateSecret(secret string) *GoogleClient {
	g.stateSecret = secret
	return g
}

// =============================================================
// GoogleUserInfo — data user dari Google API
// =============================================================

type GoogleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	FullName      string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	VerifiedEmail bool   `json:"verified_email"`
}

// =============================================================
// GenerateState — buat CSRF state parameter dengan HMAC
// Format: <timestamp>:<hmac>
// =============================================================

func (g *GoogleClient) GenerateState() string {
	ts := fmt.Sprintf("%d", time.Now().Unix())
	sig := g.signState(ts)
	return ts + ":" + sig
}

// VerifyState — verifikasi CSRF state parameter
func (g *GoogleClient) VerifyState(state string) error {
	parts := strings.SplitN(state, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid state format")
	}

	ts := parts[0]
	sig := parts[1]

	expectedSig := g.signState(ts)
	if !hmac.Equal([]byte(sig), []byte(expectedSig)) {
		return fmt.Errorf("state signature mismatch")
	}

	return nil
}

// =============================================================
// GetAuthURL — generate URL redirect ke Google login
// =============================================================

func (g *GoogleClient) GetAuthURL() string {
	state := g.GenerateState()
	return g.config.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

// =============================================================
// ExchangeCode — tukar authorization code dengan token
// =============================================================

func (g *GoogleClient) ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error) {
	token, err := g.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("exchange code: %w", err)
	}
	return token, nil
}

// =============================================================
// GetUserInfo — ambil data user dari Google API
// =============================================================

func (g *GoogleClient) GetUserInfo(ctx context.Context, token *oauth2.Token) (*GoogleUserInfo, error) {
	client := g.config.Client(ctx, token)

	resp, err := client.Get(googleUserInfoURL)
	if err != nil {
		return nil, fmt.Errorf("get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google api returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	var userInfo GoogleUserInfo
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return nil, fmt.Errorf("unmarshal user info: %w", err)
	}

	return &userInfo, nil
}

// =============================================================
// signState — HMAC-SHA256 signature untuk state parameter
// =============================================================

func (g *GoogleClient) signState(data string) string {
	secret := g.stateSecret
	if secret == "" {
		secret = "default-insecure-secret" // hanya fallback, set OAUTH_STATE_SECRET di prod
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))
}
