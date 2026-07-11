# Middlewares

常用中间件已内置在本包（`github.com/vicanso/elton/v2/middleware`），包括：

- **Recommended**：`Recover → Error → RequestID → BodyParser → Fresh → ETag → Responder`
- **CORS / Timeout / RequestID**
- recover、error、body parser、responder、compress（gzip/br/zstd）、cache、proxy、static（FS/embed）、logger、stats 等

```go
e.Use(middleware.Recommended()...)
e.Use(middleware.NewDefaultTimeout(5 * time.Second))
e.Use(middleware.NewDefaultCORS())
```

详细说明见 [middlewares](../docs/middlewares.md)；示例见 [examples](../examples/)；1.x 升级见 [migration-v2](../docs/migration-v2.md)。
