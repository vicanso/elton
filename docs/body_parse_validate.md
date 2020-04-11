---
description: body反序列化与校验
---

elton中`body-parser`中间件只将数据读取为字节，并没有做反序列化以及参数的校验。使用`json`来反序列化时，只能简单的对参数类型做校验，下面介绍如何使用[govalidator](https://github.com/asaskevich/govalidator)增强参数校验。

下面的例子是用户登录功能，参数为账号与密码，两个参数的限制如下：

- 账号：只允许为数字与字母，而且长度不能超过20位
- 密码：只允许为数字与字母，而且长度不能小于6位，不能超过20位


```go
package main

import (
	"encoding/json"

	"github.com/asaskevich/govalidator"
	"github.com/vicanso/elton"
	"github.com/vicanso/elton/middleware"
)

var (
	customTypeTagMap = govalidator.CustomTypeTagMap
)

func init() {
	// 添加自定义参数校验，如果返回false则表示参数不符合
	customTypeTagMap.Set("xAccount", func(i interface{}, _ interface{}) bool {
		v, ok := i.(string)
		if !ok || v == "" {
			return false
		}
		// 如果不是字母与数字
		if !govalidator.IsAlphanumeric(v) {
			return false
		}
		// 账号长度不能大于20
		if len(v) > 20 {
			return false
		}
		return true
	})
	customTypeTagMap.Set("xPassword", func(i interface{}, _ interface{}) bool {
		v, ok := i.(string)
		if !ok || v == "" {
			return false
		}
		// 如果不是字母与数字
		if !govalidator.IsAlphanumeric(v) {
			return false
		}
		// 密码长度不能大于20小于6
		if len(v) > 20 || len(v) < 6 {
			return false
		}
		return true
	})
}

type (
	loginParams struct {
		Account  string `json:"account,omitempty" valid:"xAccount~账号只允许数字与字母且不能超过20位"`
		Password string `json:"password,omitempty" valid:"xPassword~密码只允许数字与字母且不能少于6位超过20位"`
	}
)

func doValidate(s interface{}, data interface{}) (err error) {
	// 如果有数据则做反序列化
	if data != nil {
		switch data := data.(type) {
		case []byte:
			err = json.Unmarshal(data, s)
			if err != nil {
				return
			}
		default:
			// 如果数据不是字节，则先序列化（有可能是map）
			buf, err := json.Marshal(data)
			if err != nil {
				return err
			}
			err = json.Unmarshal(buf, s)
			if err != nil {
				return err
			}
		}
	}
	_, err = govalidator.ValidateStruct(s)
	return
}

func main() {
	e := elton.New()

	e.Use(middleware.NewError(middleware.ErrorConfig{
		ResponseType: "json",
	}))
	e.Use(middleware.NewDefaultBodyParser())
	e.Use(middleware.NewDefaultResponder())
	e.POST("/users/login", func(c *elton.Context) (err error) {
		params := &loginParams{}
		err = doValidate(params, c.RequestBody)
		if err != nil {
			return
		}
		c.Body = params
		return
	})
	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}

```


```
curl -XPOST -H 'Content-Type:application/json' -d '{"account":"treexie", "password": "123"}' 'http://127.0.0.1:3000/users/login'
```

从上面的代码中可以看到，`govalidator`可以自定义校验标签，要校验函数中可以针对值校验（一般都是长度，大小，字符类型等的校验），而且大部分的校验都可复用`govalidator`提供的常规校验函数，实现简单便捷。建议在实际项目中，针对每个不同的参数都自定义校验，尽可能保证参数的合法性。