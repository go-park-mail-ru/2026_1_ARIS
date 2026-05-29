package usecase

//go:generate mockgen -destination=mocks/round_tripper_mock.go -package=mocks net/http RoundTripper

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	userpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/user"
)

const (
	defaultVKIDTokenURL    = "https://id.vk.com/oauth2/auth"
	defaultVKIDUserInfoURL = "https://id.vk.com/oauth2/user_info"
	defaultVKUsersGetURL   = "https://api.vk.com/method/users.get"
)

type VKIDConfig struct {
	ClientID     string
	ClientSecret string
	TokenURL     string
	UserInfoURL  string
	UsersGetURL  string
	HTTPClient   *http.Client
}

type VKIDHTTPClient struct {
	client       *http.Client
	clientID     string
	clientSecret string
	tokenURL     string
	userInfoURL  string
	usersGetURL  string
}

func NewVKIDHTTPClient(config VKIDConfig) *VKIDHTTPClient {
	client := config.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	tokenURL := strings.TrimSpace(config.TokenURL)
	if tokenURL == "" {
		tokenURL = defaultVKIDTokenURL
	}
	userInfoURL := strings.TrimSpace(config.UserInfoURL)
	if userInfoURL == "" {
		userInfoURL = defaultVKIDUserInfoURL
	}
	usersGetURL := strings.TrimSpace(config.UsersGetURL)
	if usersGetURL == "" {
		usersGetURL = defaultVKUsersGetURL
	}
	return &VKIDHTTPClient{
		client:       client,
		clientID:     strings.TrimSpace(config.ClientID),
		clientSecret: strings.TrimSpace(config.ClientSecret),
		tokenURL:     tokenURL,
		userInfoURL:  userInfoURL,
		usersGetURL:  usersGetURL,
	}
}

func (c *VKIDHTTPClient) ExchangeCode(ctx context.Context, in VKIDCallbackInput) (*VKIDUser, error) {
	if c.clientID == "" {
		return nil, ErrOAuthUnavailable
	}

	token, err := c.exchangeToken(ctx, in)
	if err != nil {
		return nil, err
	}

	user, err := c.userInfo(ctx, token.AccessToken)
	if err != nil {
		return nil, err
	}
	if user.ID == "" {
		user.ID = stringFromRawJSON(token.UserID)
	}
	if enriched, err := c.vkAPIUser(ctx, token.AccessToken, user.ID); err == nil && enriched != nil {
		mergeVKIDUser(user, enriched)
	}
	return user, nil
}

type vkidTokenResponse struct {
	AccessToken string          `json:"access_token"`
	UserID      json.RawMessage `json:"user_id"`
	Error       json.RawMessage `json:"error"`
	ErrorText   string          `json:"error_description"`
}

func (c *VKIDHTTPClient) exchangeToken(ctx context.Context, in VKIDCallbackInput) (*vkidTokenResponse, error) {
	values := url.Values{}
	values.Set("grant_type", "authorization_code")
	values.Set("client_id", c.clientID)
	values.Set("code", strings.TrimSpace(in.Code))
	values.Set("code_verifier", strings.TrimSpace(in.CodeVerifier))
	values.Set("redirect_uri", strings.TrimSpace(in.RedirectURI))
	if strings.TrimSpace(in.DeviceID) != "" {
		values.Set("device_id", strings.TrimSpace(in.DeviceID))
	}
	if strings.TrimSpace(in.State) != "" {
		values.Set("state", strings.TrimSpace(in.State))
	}
	if c.clientSecret != "" {
		values.Set("client_secret", c.clientSecret)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.tokenURL, strings.NewReader(values.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	var token vkidTokenResponse
	if err := c.doJSON(req, &token); err != nil {
		return nil, err
	}
	if len(token.Error) > 0 || strings.TrimSpace(token.AccessToken) == "" {
		return nil, ErrOAuthProvider
	}
	return &token, nil
}

type vkidUserInfoResponse struct {
	User struct {
		ID           json.RawMessage `json:"user_id"`
		FirstName    string          `json:"first_name"`
		LastName     string          `json:"last_name"`
		FirstNameNom string          `json:"first_name_nom"`
		LastNameNom  string          `json:"last_name_nom"`
		Email        string          `json:"email"`
		Sex          json.RawMessage `json:"sex"`
	} `json:"user"`
	Error json.RawMessage `json:"error"`
}

func (c *VKIDHTTPClient) userInfo(ctx context.Context, accessToken string) (*VKIDUser, error) {
	values := url.Values{}
	values.Set("client_id", c.clientID)
	values.Set("access_token", accessToken)
	values.Set("lang", "ru")
	values.Set("name_case", "nom")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.userInfoURL, strings.NewReader(values.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	var resp vkidUserInfoResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, err
	}
	if len(resp.Error) > 0 {
		return nil, ErrOAuthProvider
	}

	var email *string
	if strings.TrimSpace(resp.User.Email) != "" {
		value := strings.TrimSpace(resp.User.Email)
		email = &value
	}

	return &VKIDUser{
		ID:        stringFromRawJSON(resp.User.ID),
		FirstName: firstNonEmpty(resp.User.FirstNameNom, resp.User.FirstName),
		LastName:  firstNonEmpty(resp.User.LastNameNom, resp.User.LastName),
		Email:     email,
		Gender:    vkSexToProtoGender(resp.User.Sex),
	}, nil
}

type vkAPIUsersGetResponse struct {
	Response []struct {
		ID        json.RawMessage `json:"id"`
		FirstName string          `json:"first_name"`
		LastName  string          `json:"last_name"`
		Sex       json.RawMessage `json:"sex"`
	} `json:"response"`
	Error json.RawMessage `json:"error"`
}

func (c *VKIDHTTPClient) vkAPIUser(ctx context.Context, accessToken, userID string) (*VKIDUser, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, ErrOAuthProvider
	}

	values := url.Values{}
	values.Set("access_token", accessToken)
	values.Set("user_ids", userID)
	values.Set("fields", "sex")
	values.Set("name_case", "nom")
	values.Set("lang", "ru")
	values.Set("v", "5.199")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.usersGetURL, strings.NewReader(values.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	var resp vkAPIUsersGetResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, err
	}
	if len(resp.Error) > 0 || len(resp.Response) == 0 {
		return nil, ErrOAuthProvider
	}

	user := resp.Response[0]
	return &VKIDUser{
		ID:        firstNonEmpty(stringFromRawJSON(user.ID), userID),
		FirstName: strings.TrimSpace(user.FirstName),
		LastName:  strings.TrimSpace(user.LastName),
		Gender:    vkSexToProtoGender(user.Sex),
	}, nil
}

func mergeVKIDUser(base, extra *VKIDUser) {
	if strings.TrimSpace(extra.ID) != "" {
		base.ID = strings.TrimSpace(extra.ID)
	}
	if strings.TrimSpace(extra.FirstName) != "" {
		base.FirstName = strings.TrimSpace(extra.FirstName)
	}
	if strings.TrimSpace(extra.LastName) != "" {
		base.LastName = strings.TrimSpace(extra.LastName)
	}
	if extra.Gender != userpb.Gender_GENDER_UNSPECIFIED {
		base.Gender = extra.Gender
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func vkSexToProtoGender(raw json.RawMessage) userpb.Gender {
	switch stringFromRawJSON(raw) {
	case "2":
		return userpb.Gender_GENDER_MALE
	case "1":
		return userpb.Gender_GENDER_FEMALE
	default:
		return userpb.Gender_GENDER_UNSPECIFIED
	}
}

func (c *VKIDHTTPClient) doJSON(req *http.Request, out any) error {
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return ErrOAuthProvider
	}
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(out); err != nil {
		return fmt.Errorf("%w: %v", ErrOAuthProvider, err)
	}
	return nil
}

func stringFromRawJSON(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		return normalizeJSONScalar(asString)
	}

	var asNumber json.Number
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&asNumber); err == nil {
		return asNumber.String()
	}

	return strings.Trim(string(raw), `"`)
}

func normalizeJSONScalar(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if _, err := strconv.ParseInt(value, 10, 64); err == nil {
		return value
	}
	return strings.Trim(value, `"`)
}

var _ VKIDClient = (*VKIDHTTPClient)(nil)
