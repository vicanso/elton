package middleware

import (
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/vicanso/cod"
)

const (
	defaultRealm         = "basic auth tips"
	errBasicAuthCategory = "cod-basic-auth"
)

type (
	// Validate validate function
	Validate func(string, string, *cod.Context) (bool, error)
	// BasicAuthConfig basic auth config
	BasicAuthConfig struct {
		Realm    string
		Validate Validate
	}
)

var (
	errUnauthorized = getBasicAuthError("unAuthorized", http.StatusUnauthorized)
)

func getBasicAuthError(message string, status int) *cod.HTTPError {
	return &cod.HTTPError{
		Status:   status,
		Message:  message,
		Category: errBasicAuthCategory,
	}
}

// NewBasicAuth new basic auth
func NewBasicAuth(config BasicAuthConfig) cod.Handle {
	if config.Validate == nil {
		panic("require validate function")
	}
	basic := "basic"
	basicLen := len(basic)
	realm := defaultRealm
	if config.Realm != "" {
		realm = config.Realm
	}
	wwwAuthenticate := basic + " realm=" + realm
	return func(c *cod.Context) (err error) {
		auth := c.Request.Header.Get(cod.HeaderAuthorization)
		if len(auth) < basicLen+1 ||
			strings.ToLower(auth[:basicLen]) != basic {
			c.SetHeader(cod.HeaderWWWAuthenticate, wwwAuthenticate)
			err = errUnauthorized
			return
		}

		v, e := base64.StdEncoding.DecodeString(auth[basicLen+1:])
		if e != nil {
			err = getBasicAuthError(e.Error(), http.StatusBadRequest)
			return err
		}

		arr := strings.Split(string(v), ":")
		valid, e := config.Validate(arr[0], arr[1], c)
		if e != nil {
			err = getBasicAuthError(e.Error(), http.StatusBadRequest)
			return err
		}
		if !valid {
			c.SetHeader(cod.HeaderWWWAuthenticate, wwwAuthenticate)
			err = errUnauthorized
			return
		}
		return c.Next()
	}
}
