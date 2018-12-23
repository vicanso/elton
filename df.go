package cod

import (
	"net/http"

	"github.com/vicanso/errors"
)

var (
	methods = []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodHead,
		http.MethodOptions,
		http.MethodTrace,
	}

	// ErrInvalidRedirect invalid redirect
	ErrInvalidRedirect = &errors.HTTPError{
		StatusCode: 400,
		Message:    "invalid redirect",
		Category:   ErrCategoryCod,
	}
	// ErrInvalidResponse invalid response(body an status is nil)
	ErrInvalidResponse = &errors.HTTPError{
		StatusCode: 500,
		Message:    "invalid response",
		Category:   ErrCategoryCod,
	}
)

const (
	// ErrCategoryCod cod category
	ErrCategoryCod = "cod"
	// HeaderXForwardedFor x-forwarded-for
	HeaderXForwardedFor = "X-Forwarded-For"
	// HeaderXRealIp x-real-ip
	HeaderXRealIp = "X-Real-Ip"
	// HeaderSetCookie Set-Cookie
	HeaderSetCookie = "Set-Cookie"
	// HeaderLocation Location
	HeaderLocation = "Location"
	// HeaderContentType Content-Type
	HeaderContentType = "Content-Type"
	// HeaderAuthorization Authorization
	HeaderAuthorization = "Authorization"
	// HeaderWWWAuthenticate WWW-Authenticate
	HeaderWWWAuthenticate = "WWW-Authenticate"
	// HeaderCacheControl Cache-Control
	HeaderCacheControl = "Cache-Control"

	// MinRedirectCode min redirect code
	MinRedirectCode = 300
	// MaxRedirectCode max redirect code
	MaxRedirectCode = 308

	// MIMETextPlain text plain
	MIMETextPlain = "text/plain;charset=UTF-8"
	// MIMEApplicationJSON application json
	MIMEApplicationJSON = "application/json;charset=UTF-8"
	// MIMEBinary binary data
	MIMEBinary = "application/octet-stream"
)
