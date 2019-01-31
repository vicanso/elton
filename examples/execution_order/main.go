package main

import (
	"bytes"
	"fmt"

	"github.com/vicanso/cod"
)

// 日志输出如下
// middleware 1 start
// middleware 2 start
// middleware 2 end
// middleware 2 defer
// middleware 1 end
// middleware 1 defer

func main() {
	d := cod.New()

	d.Use(func(c *cod.Context) (err error) {
		defer fmt.Println("middleware 1 defer")
		fmt.Println("middleware 1 start")
		err = c.Next()
		fmt.Println("middleware 1 end")
		return
	})

	d.Use(func(c *cod.Context) (err error) {
		defer fmt.Println("middleware 2 defer")
		fmt.Println("middleware 2 start")
		err = c.Next()
		fmt.Println("middleware 2 end")
		return
	})

	d.GET("/", func(c *cod.Context) error {
		c.BodyBuffer = bytes.NewBufferString("hello world")
		return nil
	})

	d.ListenAndServe(":8001")
}
