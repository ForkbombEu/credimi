// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package githubapp

import (
	"bytes"
	"context"
	"crypto/rsa"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/golang-jwt/jwt/v5"
)

const DefaultMarker = "<!-- credimi-wallet-apk-pr-comment -->"

type PRComment struct {
	Repository        string
	PullRequestNumber int
	CommentID         int64
	Marker            string
	Body              string
}

type PRCommentResult struct {
	CommentID int64
}

type Client struct {
	httpClient *http.Client
	apiURL     string
	clientID   string
	privateKey *rsa.PrivateKey
}

func NewFromEnv() (*Client, error) {
	privateKey, err := parsePrivateKey(strings.TrimSpace(os.Getenv("GITHUB_APP_PRIVATE_KEY")))
	if err != nil {
		return nil, err
	}
	clientID := strings.TrimSpace(os.Getenv("GITHUB_APP_CLIENT_ID"))
	if clientID == "" {
		clientID = strings.TrimSpace(os.Getenv("GITHUB_APP_ID"))
	}
	if clientID == "" {
		return nil, fmt.Errorf("GITHUB_APP_CLIENT_ID or GITHUB_APP_ID is required")
	}
	apiURL := strings.TrimRight(strings.TrimSpace(os.Getenv("GITHUB_API_URL")), "/")
	if apiURL == "" {
		apiURL = "https://api.github.com"
	}
	return &Client{
		httpClient: &http.Client{Timeout: 15 * time.Second},
		apiURL:     apiURL,
		clientID:   clientID,
		privateKey: privateKey,
	}, nil
}

func (c *Client) CreateOrUpdatePRComment(
	ctx context.Context,
	input PRComment,
) (PRCommentResult, error) {
	owner, repo, err := splitRepository(input.Repository)
	if err != nil {
		return PRCommentResult{}, err
	}
	if input.PullRequestNumber <= 0 {
		return PRCommentResult{}, fmt.Errorf("pull request number is required")
	}
	marker := strings.TrimSpace(input.Marker)
	if marker == "" {
		return PRCommentResult{}, fmt.Errorf("comment marker is required")
	}
	body := ensureMarker(input.Body, marker)

	token, err := c.installationToken(ctx, owner, repo)
	if err != nil {
		return PRCommentResult{}, err
	}
	if input.CommentID > 0 {
		return c.patchComment(ctx, token, owner, repo, input.CommentID, body)
	}
	commentID, err := c.findCommentID(ctx, token, owner, repo, input.PullRequestNumber, marker)
	if err != nil {
		return PRCommentResult{}, err
	}
	if commentID > 0 {
		return c.patchComment(ctx, token, owner, repo, commentID, body)
	}
	return c.createComment(ctx, token, owner, repo, input.PullRequestNumber, body)
}

func (c *Client) PullRequestHeadSHA(
	ctx context.Context,
	repository string,
	prNumber int,
) (string, error) {
	owner, repo, err := splitRepository(repository)
	if err != nil {
		return "", err
	}
	if prNumber <= 0 {
		return "", fmt.Errorf("pull request number is required")
	}
	token, err := c.installationToken(ctx, owner, repo)
	if err != nil {
		return "", err
	}
	var out struct {
		Head struct {
			SHA string `json:"sha"`
		} `json:"head"`
	}
	if err := c.doJSON(
		ctx,
		http.MethodGet,
		c.githubAPIURL("repos", owner, repo, "pulls", strconv.Itoa(prNumber)),
		token,
		nil,
		&out,
	); err != nil {
		return "", err
	}
	if strings.TrimSpace(out.Head.SHA) == "" {
		return "", fmt.Errorf("github pull request response missing head sha")
	}
	return strings.TrimSpace(out.Head.SHA), nil
}

func (c *Client) installationToken(ctx context.Context, owner, repo string) (string, error) {
	jwtToken, err := c.jwt()
	if err != nil {
		return "", err
	}

	var installation struct {
		ID int64 `json:"id"`
	}
	if err := c.doJSON(
		ctx,
		http.MethodGet,
		c.githubAPIURL("repos", owner, repo, "installation"),
		jwtToken,
		nil,
		&installation,
	); err != nil {
		return "", err
	}
	if installation.ID <= 0 {
		return "", fmt.Errorf("github app installation not found for %s/%s", owner, repo)
	}

	var tokenResp struct {
		Token string `json:"token"`
	}
	if err := c.doJSON(
		ctx,
		http.MethodPost,
		c.githubAPIURL(
			"app",
			"installations",
			strconv.FormatInt(installation.ID, 10),
			"access_tokens",
		),
		jwtToken,
		map[string]any{},
		&tokenResp,
	); err != nil {
		return "", err
	}
	if strings.TrimSpace(tokenResp.Token) == "" {
		return "", fmt.Errorf("github installation token response missing token")
	}
	return tokenResp.Token, nil
}

func (c *Client) findCommentID(
	ctx context.Context,
	token, owner, repo string,
	prNumber int,
	marker string,
) (int64, error) {
	requestURL := c.githubAPIURL("repos", owner, repo, "issues", strconv.Itoa(prNumber), "comments")
	requestURL = withQueryParam(requestURL, "per_page", "100")
	var comments []struct {
		ID   int64  `json:"id"`
		Body string `json:"body"`
	}
	if err := c.doJSON(ctx, http.MethodGet, requestURL, token, nil, &comments); err != nil {
		return 0, err
	}
	for _, comment := range comments {
		if strings.Contains(comment.Body, marker) {
			return comment.ID, nil
		}
	}
	return 0, nil
}

func (c *Client) createComment(
	ctx context.Context,
	token, owner, repo string,
	prNumber int,
	body string,
) (PRCommentResult, error) {
	var out struct {
		ID int64 `json:"id"`
	}
	err := c.doJSON(
		ctx,
		http.MethodPost,
		c.githubAPIURL("repos", owner, repo, "issues", strconv.Itoa(prNumber), "comments"),
		token,
		map[string]any{"body": body},
		&out,
	)
	return PRCommentResult{CommentID: out.ID}, err
}

func (c *Client) patchComment(
	ctx context.Context,
	token, owner, repo string,
	commentID int64,
	body string,
) (PRCommentResult, error) {
	var out struct {
		ID int64 `json:"id"`
	}
	err := c.doJSON(
		ctx,
		http.MethodPatch,
		c.githubAPIURL(
			"repos",
			owner,
			repo,
			"issues",
			"comments",
			strconv.FormatInt(commentID, 10),
		),
		token,
		map[string]any{"body": body},
		&out,
	)
	if out.ID == 0 {
		out.ID = commentID
	}
	return PRCommentResult{CommentID: out.ID}, err
}

func (c *Client) githubAPIURL(parts ...string) string {
	return utils.JoinURL(c.apiURL, parts...)
}

func withQueryParam(rawURL string, key string, value string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed == nil {
		return rawURL
	}
	query := parsed.Query()
	query.Set(key, value)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func (c *Client) doJSON(
	ctx context.Context,
	method, requestURL, token string,
	payload any,
	out any,
) error {
	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, requestURL, body)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("Authorization", "Bearer "+token)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf(
			"github %s %s: %s: %s",
			method,
			requestURL,
			resp.Status,
			strings.TrimSpace(string(data)),
		)
	}
	if out == nil {
		io.Copy(io.Discard, resp.Body)
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode github response: %w", err)
	}
	return nil
}

func (c *Client) jwt() (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Issuer:    c.clientID,
		IssuedAt:  jwt.NewNumericDate(now.Add(-time.Minute)),
		ExpiresAt: jwt.NewNumericDate(now.Add(9 * time.Minute)),
	}
	return jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(c.privateKey)
}

func parsePrivateKey(raw string) (*rsa.PrivateKey, error) {
	if raw == "" {
		return nil, fmt.Errorf("GITHUB_APP_PRIVATE_KEY is required")
	}
	raw = strings.ReplaceAll(raw, `\n`, "\n")
	block, _ := pem.Decode([]byte(raw))
	if block == nil {
		return nil, fmt.Errorf("GITHUB_APP_PRIVATE_KEY must be PEM encoded")
	}
	key, err := jwt.ParseRSAPrivateKeyFromPEM(pem.EncodeToMemory(block))
	if err != nil {
		return nil, fmt.Errorf("parse GITHUB_APP_PRIVATE_KEY: %w", err)
	}
	return key, nil
}

func splitRepository(repository string) (string, string, error) {
	repository = strings.TrimSpace(repository)
	if parsed, err := url.Parse(repository); err == nil && parsed.Host != "" {
		repository = strings.TrimPrefix(parsed.Path, "/")
	}
	parts := strings.Split(repository, "/")
	if len(parts) < 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", fmt.Errorf("repository must be owner/name")
	}
	return url.PathEscape(parts[0]), url.PathEscape(parts[1]), nil
}

func Marker() string {
	return DefaultMarker
}

func ensureMarker(body string, marker string) string {
	body = strings.TrimSpace(body)
	if strings.Contains(body, marker) {
		return body
	}
	return body + "\n\n" + marker
}

func IntFromAny(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case json.Number:
		i, _ := strconv.Atoi(v.String())
		return i
	case string:
		i, _ := strconv.Atoi(strings.TrimSpace(v))
		return i
	default:
		return 0
	}
}
