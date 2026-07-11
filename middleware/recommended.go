// MIT License

// Copyright (c) 2026 Tree Xie

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package middleware

import "github.com/vicanso/elton/v2"

// Recommended returns a common global middleware stack suitable for JSON APIs:
//
//	Recover → Error → RequestID → BodyParser → Fresh → ETag → Responder
//
// Usage:
//
//	e.Use(middleware.Recommended()...)
//
// Order notes:
//   - Recover outermost so panics become errors (and Error can format them if re-entered via EmitError path separately)
//   - Error wraps the rest so returned errors become JSON/text responses
//   - RequestID early for logging / tracing correlation
//   - BodyParser before handlers that need RequestBody
//   - Fresh/ETag before Responder so 304 can skip body serialization costs where applicable
//   - Responder innermost among globals so c.Body is converted after business handlers
//
// Not included (add as needed): CORS, Timeout, Compress, Logger, Stats.
func Recommended() []elton.Handler {
	return []elton.Handler{
		NewRecover(),
		NewDefaultError(),
		NewDefaultRequestID(),
		NewDefaultBodyParser(),
		NewDefaultFresh(),
		NewDefaultETag(),
		NewDefaultResponder(),
	}
}
