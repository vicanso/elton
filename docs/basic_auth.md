# basic auth

HTTP Basic auth认证中间件，此认证方式非常简单，但是安全性比较低，适用对安全性要求较低，如内部管理系统等使用。

```go
d := cod.New()

d.Use(middleware.NewBasicAuth(middleware.BasicAuthConfig{
  Realm: "请输入管理用户与密码",
  Validate: func(account, password string, c *cod.Context) (bool, error) {
    // 校验用户与密码是否正确
    if account == "tree.xie" && password == "password" {
      return true, nil
    }
    return false, nil
  },
}))
```

`basic auth`中间件最主要的就是Validate函数，此函数参数中有账户与密码，判断是否正确，如果正确则返回true，否则返回false则可。例子中仅是简单的演示，实现使用中可以从数据库中读取相应的账号密码判断。
