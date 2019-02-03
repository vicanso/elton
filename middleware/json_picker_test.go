package middleware

import (
	"bytes"
	"net/http/httptest"
	"testing"

	"github.com/vicanso/cod"
)

func TestDefaultJSONPickerValidate(t *testing.T) {
	resp := httptest.NewRecorder()
	c := cod.NewContext(resp, nil)
	if defaultJSONPickerValidate(c) {
		t.Fatalf("nil body buffer should return false")
	}
	c.BodyBuffer = bytes.NewBufferString("")
	if defaultJSONPickerValidate(c) {
		t.Fatalf("empty body buffer should return false")
	}
	c.BodyBuffer = bytes.NewBufferString(`{
		"name": "tree.xie"
	}`)
	if defaultJSONPickerValidate(c) {
		t.Fatalf("status code <200 should return false")
	}
	c.StatusCode = 400

	if defaultJSONPickerValidate(c) {
		t.Fatalf("status code >= 300 should return false")
	}

	c.StatusCode = 200

	if defaultJSONPickerValidate(c) {
		t.Fatalf("not json should return false")
	}

	c.SetHeader(cod.HeaderContentType, cod.MIMEApplicationJSON)
	if !defaultJSONPickerValidate(c) {
		t.Fatalf("should be valid")
	}
}

func TestJSONPicker(t *testing.T) {

	t.Run("no field", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/users/me", nil)
		c := cod.NewContext(nil, req)
		c.BodyBuffer = bytes.NewBufferString(`{
			"name": "tree.xie",
			"type": "vip"
		}`)
		c.Next = func() error {
			return nil
		}
		fn := NewJSONPicker(JSONPickerConfig{
			Field: "fields",
		})
		err := fn(c)
		if err != nil {
			t.Fatalf("json pick fail, %v", err)
		}
	})

	t.Run("pick", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/users/me?fields=i,f,s,b,arr,m,null,xx", nil)
		resp := httptest.NewRecorder()
		c := cod.NewContext(resp, req)
		m := map[string]interface{}{
			"_x": "abcd",
			"i":  1,
			"f":  1.12,
			"s":  "\"abc",
			"b":  false,
			"arr": []interface{}{
				1,
				"2",
				true,
			},
			"m": map[string]interface{}{
				"a": 1,
				"b": "2",
				"c": false,
			},
			"null": nil,
		}
		buf, _ := json.Marshal(m)
		c.BodyBuffer = bytes.NewBuffer(buf)
		c.StatusCode = 200
		c.Next = func() error {
			return nil
		}
		c.SetHeader(cod.HeaderContentType, cod.MIMEApplicationJSON)
		fn := NewJSONPicker(JSONPickerConfig{
			Field: "fields",
		})
		t.Run("pick fields", func(t *testing.T) {
			err := fn(c)
			if err != nil {
				t.Fatalf("json picker fail, %v", err)
			}
			if c.BodyBuffer.String() != `{"arr":[1,"2",true],"b":false,"f":1.12,"i":1,"m":{"a":1,"b":"2","c":false},"s":"\"abc"}` {
				t.Fatalf("json pick fail")
			}
		})

		t.Run("omit fields", func(t *testing.T) {
			req := httptest.NewRequest("GET", "/users/me?fields=-x", nil)
			c.Request = req
			err := fn(c)
			if err != nil {
				t.Fatalf("omit picker fail, %v", err)
			}
			if c.BodyBuffer.String() != `{"arr":[1,"2",true],"b":false,"f":1.12,"i":1,"m":{"a":1,"b":"2","c":false},"s":"\"abc"}` {
				t.Fatalf("json pick fail")
			}
		})
	})
}
