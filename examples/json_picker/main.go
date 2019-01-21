package main

import (
	"github.com/vicanso/cod"
	"github.com/vicanso/cod/middleware"
)

// curl 'http://127.0.0.1:8001/books/1?fields=name,price'
// {"name":"测试书","price":12.3}

func main() {
	d := cod.New()

	d.Use(middleware.NewRecover())

	d.Use(middleware.NewFresh(middleware.FreshConfig{}))
	d.Use(middleware.NewETag(middleware.ETagConfig{}))

	d.Use(middleware.NewJSONPicker(middleware.JSONPickerConfig{
		// 指定querystring中哪个字段
		Field: "fields",
	}))

	d.Use(middleware.NewResponder(middleware.ResponderConfig{}))

	d.GET("/users/me", func(c *cod.Context) (err error) {
		c.Body = &struct {
			Account string `json:"account"`
		}{
			"tree.xie",
		}
		return
	})

	d.GET("/books/:id", func(c *cod.Context) (err error) {
		c.Body = &struct {
			Name     string  `json:"name"`
			Price    float32 `json:"price"`
			Category string  `json:"category"`
		}{
			"测试书",
			12.3,
			"IT",
		}
		return
	})

	d.ListenAndServe(":8001")
}
