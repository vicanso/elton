package main

import (
	"github.com/vicanso/cod"
	"github.com/vicanso/cod/middleware"
)

// curl 'http://127.0.0.1:8001/assets/gen_cookie.js'
// const crypto = require('crypto');

// function sign(val, secret) {
// 	return val + '.' + crypto
//     .createHmac('sha256', secret)
//     .update(val)
//     .digest('base64')
//     .replace(/\=+$/, '');
// }

// const v = sign('72c48620-f8f8-11e8-9abf-5b92bb50b8bd_6774_157741_180', 'agile2013');
// console.dir(v);

func main() {
	d := cod.New()

	d.Use(middleware.NewRecover())

	d.Use(middleware.NewDefaultFresh())
	d.Use(middleware.NewDefaultETag())

	d.Use(middleware.NewDefaultResponder())

	d.GET("/users/me", func(c *cod.Context) (err error) {
		c.Body = &struct {
			Account string `json:"account"`
		}{
			"tree.xie",
		}
		return
	})

	static := middleware.NewDefaultStaticServe(middleware.StaticServeConfig{
		Path:            "/Users/xieshuzhou/tmp",
		Mount:           "/assets",
		DenyQueryString: true,
		DenyDot:         true,
		MaxAge:          60 * 60,
	})
	d.GET("/assets/*file", static, func(c *cod.Context) (err error) {
		return
	})

	d.ListenAndServe(":8001")
}
