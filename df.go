package cod

import "net/http"

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
)

const (
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
