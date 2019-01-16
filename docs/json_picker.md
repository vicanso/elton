# json picker

json picker用于从响应的json数据中挑选指定的字段，便于根据实际使用需要筛选字段，减少无用数据。此中间件调用需要在NewResponder之前，因为需要NewResponder将interface{}转换为字节。

默认只针对成功的请求，并且响应数据为json的筛选，下面是默认的校验函数，也可根据实际场景自定义。

```go
defaultJSONPickerValidate = func(c *cod.Context) bool {
  // 如果响应数据为空，则不符合
  if c.BodyBuffer == nil || c.BodyBuffer.Len() == 0 {
    return false
  }
  statusCode := c.StatusCode
  // http状态码如果非 >= 200 < 300，则不符合
  if statusCode < http.StatusOK ||
    statusCode >= http.StatusMultipleChoices {
    return false
  }
  // 如果非json，则不符合
  if !strings.Contains(c.GetHeader(cod.HeaderContentType), "json") {
    return false
  }
  return true
}
```

```go
// 指定使用querystring中的fields来筛选响应数据
d.Use(middleware.NewJSONPicker(middleware.JSONPickerConfig{
  Field: "fields",
}))

// 针对出错error生成相应的HTTP响应数据（http状态码以及响应数据）
// 或者成功处理的Body生成相应的HTTP响应数据
d.Use(middleware.NewResponder(middleware.ResponderConfig{}))
```

如上面例子所示，Field指定query中的字段，如需要筛选`name`与`type`两个字段，query部分为`?fields=name,type`。

