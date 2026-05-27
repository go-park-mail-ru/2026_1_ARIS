package usecase

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	userpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/user"
	transportmock "github.com/go-park-mail-ru/2026_1_ARIS/services/auth/internal/usecase/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestVKIDHTTPClientExchangeCode(t *testing.T) {
	ctrl := gomock.NewController(t)
	transport := transportmock.NewMockRoundTripper(ctrl)
	gomock.InOrder(
		transport.EXPECT().RoundTrip(gomock.Any()).DoAndReturn(func(r *http.Request) (*http.Response, error) {
			require.Equal(t, http.MethodPost, r.Method)
			require.Equal(t, "https://vk.example/token", r.URL.String())
			require.NoError(t, r.ParseForm())
			require.Equal(t, "client", r.Form.Get("client_id"))
			require.Equal(t, "code", r.Form.Get("code"))
			require.Equal(t, "verifier", r.Form.Get("code_verifier"))
			return jsonResponse(http.StatusOK, `{"access_token":"token","user_id":123}`), nil
		}),
		transport.EXPECT().RoundTrip(gomock.Any()).DoAndReturn(func(r *http.Request) (*http.Response, error) {
			require.Equal(t, "https://vk.example/user_info", r.URL.String())
			require.Equal(t, "Bearer token", r.Header.Get("Authorization"))
			return jsonResponse(http.StatusOK, `{"user":{"user_id":"123","first_name":"Old","last_name":"Name","first_name_nom":"Neo","last_name_nom":"Anderson","email":"neo@example.com","sex":"2"}}`), nil
		}),
		transport.EXPECT().RoundTrip(gomock.Any()).DoAndReturn(func(r *http.Request) (*http.Response, error) {
			require.Equal(t, "https://vk.example/users_get", r.URL.String())
			return jsonResponse(http.StatusOK, `{"response":[{"id":123,"first_name":"Thomas","last_name":"Anderson","sex":2}]}`), nil
		}),
	)

	client := NewVKIDHTTPClient(VKIDConfig{
		ClientID:    "client",
		TokenURL:    "https://vk.example/token",
		UserInfoURL: "https://vk.example/user_info",
		UsersGetURL: "https://vk.example/users_get",
		HTTPClient:  &http.Client{Transport: transport},
	})

	user, err := client.ExchangeCode(context.Background(), VKIDCallbackInput{
		Code:         "code",
		CodeVerifier: "verifier",
		RedirectURI:  "https://app/callback",
	})

	require.NoError(t, err)
	require.Equal(t, "123", user.ID)
	require.Equal(t, "Thomas", user.FirstName)
	require.Equal(t, "Anderson", user.LastName)
	require.Equal(t, "neo@example.com", *user.Email)
	require.Equal(t, userpb.Gender_GENDER_MALE, user.Gender)
}

func TestVKIDHTTPClientErrorsAndHelpers(t *testing.T) {
	t.Run("missing client id", func(t *testing.T) {
		user, err := NewVKIDHTTPClient(VKIDConfig{}).ExchangeCode(context.Background(), VKIDCallbackInput{})
		require.Nil(t, user)
		require.ErrorIs(t, err, ErrOAuthUnavailable)
	})

	t.Run("token provider error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		transport := transportmock.NewMockRoundTripper(ctrl)
		transport.EXPECT().
			RoundTrip(gomock.Any()).
			Return(jsonResponse(http.StatusBadGateway, "bad"), nil)
		client := NewVKIDHTTPClient(VKIDConfig{
			ClientID:   "client",
			TokenURL:   "https://vk.example/token",
			HTTPClient: &http.Client{Transport: transport},
		})

		user, err := client.ExchangeCode(context.Background(), VKIDCallbackInput{Code: "c", CodeVerifier: "v", RedirectURI: "u"})

		require.Nil(t, user)
		require.ErrorIs(t, err, ErrOAuthProvider)
	})

	t.Run("raw json scalar", func(t *testing.T) {
		require.Equal(t, "42", stringFromRawJSON(json.RawMessage(`"42"`)))
		require.Equal(t, "42", stringFromRawJSON(json.RawMessage(`42`)))
		require.Equal(t, "abc", stringFromRawJSON(json.RawMessage(`"abc"`)))
		require.Equal(t, "", stringFromRawJSON(nil))
	})

	t.Run("merge and gender helpers", func(t *testing.T) {
		email := "base@example.com"
		base := &VKIDUser{ID: "1", FirstName: "Base", LastName: "User", Email: &email}
		mergeVKIDUser(base, &VKIDUser{ID: " 2 ", FirstName: " Neo ", LastName: " Anderson ", Gender: userpb.Gender_GENDER_FEMALE})
		require.Equal(t, "2", base.ID)
		require.Equal(t, "Neo", base.FirstName)
		require.Equal(t, "Anderson", base.LastName)
		require.Equal(t, userpb.Gender_GENDER_FEMALE, base.Gender)
		require.Equal(t, "x", firstNonEmpty(" ", "x"))
		require.Equal(t, userpb.Gender_GENDER_MALE, vkSexToProtoGender(json.RawMessage(`2`)))
		require.Equal(t, userpb.Gender_GENDER_FEMALE, vkSexToProtoGender(json.RawMessage(`"1"`)))
		require.Equal(t, userpb.Gender_GENDER_UNSPECIFIED, vkSexToProtoGender(json.RawMessage(`0`)))
	})
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}
