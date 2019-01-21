package main

import (
	"github.com/vicanso/cod"
	"github.com/vicanso/cod/middleware"
)

// curl -XPOST --header "Content-Type: application/json" -d '{"account": "tree.xie"}' 'http://127.0.0.1:8001/users/login'
// {"account": "tree.xie"}

func main() {
	d := cod.New()

	d.Use(middleware.NewRecover())

	d.Use(middleware.NewFresh(middleware.FreshConfig{}))
	d.Use(middleware.NewETag(middleware.ETagConfig{}))

	d.Use(middleware.NewBodyParser(middleware.BodyParserConfig{}))

	d.Use(middleware.NewResponder(middleware.ResponderConfig{}))

	d.GET("/users/me", func(c *cod.Context) (err error) {
		c.Body = &struct {
			Account string `json:"account"`
		}{
			"tree.xie",
		}
		return
	})

	d.POST("/users/login", func(c *cod.Context) (err error) {
		c.SetHeader(cod.HeaderContentType, cod.MIMEApplicationJSON)
		// 演示直接将提交的数据返回
		c.Body = c.RequestBody
		return
	})

	d.ListenAndServe(":8001")
}
