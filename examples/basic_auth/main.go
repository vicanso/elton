package main

import (
	"github.com/vicanso/cod"
	"github.com/vicanso/cod/middleware"
)

// 在浏览器打开则会弹出认证框
// curl 'http://127.0.0.1:8001/admin/index.html'
// {"statusCode":401,"category":"cod-basic-auth","message":"unAuthorized"}

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

	basicAuth := middleware.NewBasicAuth(middleware.BasicAuthConfig{
		Validate: func(account, password string, c *cod.Context) (bool, error) {
			// 校验账号密码是否正确，此示例直接忽略校验
			return true, nil
		},
	})

	// 增加一个group admin，指定url前缀以及公共的中间件
	adminGroup := cod.NewGroup("/admin", basicAuth)
	adminGroup.GET("/index.html", basicAuth, func(c *cod.Context) (err error) {
		c.SetFileContentType(".html")
		c.Body = `<html>
			<body>
				...TODO
			</body>
		</html>`
		return
	})
	d.AddGroup(adminGroup)

	d.ListenAndServe(":8001")
}
