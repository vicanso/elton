# Elton 2.0 迁移指南

Elton 2.0 是一次破坏性升级，模块路径变更为 `github.com/vicanso/elton/v2`，最低要求 **Go 1.24**。

```bash
go get github.com/vicanso/elton/v2
```

```go
import (
    "github.com/vicanso/elton/v2"
    "github.com/vicanso/elton/v2/middleware"
)
```

中间件的洋葱模型、`Handler func(*Context) error` 签名、`c.Next()` 调用链、`Body`/`BodyBuffer` 响应模型均保持不变，业务代码大部分无需调整。

## 核心包破坏性变更

### Context 类型 getter 改为泛型函数

`GetInt`/`GetInt64`/`GetString`/`GetBool`/`GetFloat32`/`GetFloat64`/`GetTime`/`GetDuration`/`GetStringSlice` 已删除，统一替换为泛型函数 `GetContextValue[T]`：

```go
// v1
value := c.GetString("account")
count := c.GetInt("count")

// v2
value := elton.GetContextValue[string](c, "account")
count := elton.GetContextValue[int](c, "count")
```

key 不存在或类型不匹配时返回 T 的零值，行为与 v1 一致。

### 生命周期 API

| v1 | v2 | 说明 |
|---|---|---|
| `Shutdown()` | `Shutdown(ctx context.Context)` | 直接透传给 `http.Server.Shutdown` |
| `GracefulClose(delay)` | `GracefulClose(ctx, delay)` | 等待期间可通过 ctx 取消，不再使用 `time.Sleep` 阻塞 |
| `ListenAndServe` / `ListenAndServeTLS` / `Serve` / `Close` | 同名 | Server 为 nil 时不再 panic，返回 `elton.ErrServerNotInitialized` |
| `GetStatus() int32` | `Status() Status` | 状态常量类型化为 `elton.Status`，并去掉 Get 前缀 |

### 移除的 API

- `Context.Push`（HTTP/2 Server Push 已被主流浏览器废弃），连带删除 `ErrNilResponse`、`ErrNotSupportPush`。
- 常量 `ReuseContextEnabled`、`ReuseContextDisabled`（内部实现细节，`DisableReuse()` 不受影响）。

### Group

- `GET`/`POST` 等方法现在返回 `*Group`，支持链式调用。
- 新增嵌套分组 `g.NewGroup(path, handlers...)`，子分组随父分组一起通过 `e.AddGroup` 注册。

```go
api := elton.NewGroup("/api", authMiddleware)
v1 := api.NewGroup("/v1")
v1.GET("/users", listUsers).POST("/users", createUser)
e.AddGroup(api)
```

### 命名规范化

按 Go 官方命名惯例（无参属性 getter 不加 Get 前缀、缩写词全大写、语义对齐标准库）调整以下符号：

| v1 | v2 | 说明 |
|---|---|---|
| `Elton.GetStatus()` | `Elton.Status()` | 属性 getter 去 Get 前缀 |
| `Elton.GetRouters()` | `Elton.Routers()` | 同上 |
| `Context.GetTrace()` | `Context.Trace()` | 与 `NewTrace()` 配对 |
| `elton.GetTrace(ctx)` | `elton.TraceFromContext(ctx)` | context 取值的生态惯例；**注意**：context 中无 trace 时返回**未挂载**的新 `*Trace`，不会写回 request context，业务打点请用 `c.NewTrace()` 或开启 `EnableTrace` |
| `SignedKeysGenerator.GetKeys()` | `Keys()` | 接口方法，自定义实现需同步修改 |
| `CacheCompressor.GetEncoding()` | `Encoding()` | 接口方法 |
| `CacheCompressor.GetCompression()` | `Compression()` | 接口方法 |
| `Context.ReadFile(key)` | `Context.ReadFormFile(key)` | 实为读取 multipart 上传文件，避免与 `os.ReadFile` 语义混淆 |
| `Context.GetSignedCookie(name)` | `Context.SignedCookieWithIndex(name)` | 名称体现"额外返回 key index"的差异 |
| `MultipartForm.Destroy()` | `MultipartForm.Close()` | 实现 `io.Closer` 惯例 |
| `middleware.EmbedFs` | `middleware.EmbedFS` | 缩写词全大写 |
| `middleware.NewLruStore` / `NewPeekLruStore` | `NewLRUStore` / `NewPeekLRUStore` | 缩写词全大写 |
| `middleware.NewDefaultStaticServe` | `middleware.NewFSStaticServe` | 语义为"使用默认 os 文件系统"，与其他 `NewDefaultXXX`（零配置）区分 |
| `TraceInfos.FilterDurationGT` | `TraceInfos.FilterDurationGreaterThan` | 避免非惯例缩写 |

注意：`GetHeader`/`SetHeader`/`GetRequestHeader`、`Context.Get/Set`、`GetClientIP(req)` 等**带 key 参数的查找型方法**符合标准库惯例（如 `http.Header.Get`），保持不变。另外提醒：elton 中裸名 `GetHeader`/`SetHeader` 操作的是**响应**头（与 gin 的 `c.GetHeader` 读请求头相反），读请求头请使用 `GetRequestHeader`。

### 其它

- 所有导出签名中的 `interface{}` 改为 `any`（源码兼容）。
- `NewMultipartForm` 返回导出类型 `*MultipartForm`。
- 默认错误处理使用 `errors.As` 判断 `*hes.Error`，经 `fmt.Errorf("%w")` 包装的错误现在也能被正确识别状态码。
- `SetFunctionName`/`GetFunctionName` 增加并发保护，可在服务运行期间安全注册路由。

## middleware 包破坏性变更

### 路由并发限制（原 RCL）改名

| v1 | v2 |
|---|---|
| `NewRCL` | `NewRouterConcurrentLimiter` |
| `RCLConfig` | `RouterConcurrentLimiterConfig` |
| `RCLLimiter` | `RouterConcurrencyLimiter` |
| `RCLLocalLimiter` | `LocalRouterConcurrencyLimiter` |
| `NewLocalLimiter` | `NewLocalRouterConcurrencyLimiter` |
| `ErrRCLCategory` | `ErrRouterConcurrentLimiterCategory` |
| `ErrRCLRequireLimiter` | `ErrRequireLimiter` |

### 导出返回类型

- `NewLRUStore` / `NewPeekLRUStore` 返回 `*LRUStore`（原为未导出 `*lruStore`，构造函数同步修正缩写大小写）。
- `NewEmbedStaticFS` 返回 `*EmbedStaticFS`，`NewTarFS` 返回 `*TarFS`。
- `MaxBytesReader` 返回 `io.ReadCloser`（与 `http.MaxBytesReader` 一致）。
- **`StaticFile` 接口的 `NewReader` 返回 `io.ReadCloser`**（原为 `io.Reader`）。自定义实现需同步调整：返回 `*os.File`、`fs.File` 等本身即满足；内存型 reader 用 `io.NopCloser` 包装即可。reader 的关闭由框架统一负责（见下方行为变化）。

### 压缩器统一

- 新增构造函数 `NewGzipCompressor()`、`NewBrCompressor()`、`NewZstdCompressor()`。
- `CacheGzipCompressor` / `CacheBrCompressor` / `CacheZstdCompressor` 改为内嵌对应的运行时压缩器，`Level`、`MinLength` 字段通过内嵌字段提升访问方式不变；但**结构体字面量**构造方式需调整：

```go
// v1
compressor := &middleware.CacheBrCompressor{Level: 6}

// v2
compressor := middleware.NewCacheBrCompressor()
compressor.Level = 6
```

### 行为变化

- `NewGlobalConcurrentLimiter` 现在会应用 `Skipper` 配置（v1 中该字段被忽略）。注意其 `Max` 语义为"在途请求数达到 Max 即拒绝"，实际允许的最大并发为 `Max - 1`（与 v1 一致，文档已注明）。
- `error`、`basic_auth`、`stats`、`static_serve` 中间件改用 `errors.As` 识别 `*hes.Error`，包装过的错误也能正确解析状态码。
- `NewHTTPHeaders` 的忽略头匹配从子串匹配改为小写精确匹配，名称恰好是 `content-encoding` 等子串的自定义头（如 `Encoding`）不再被误删。
- **框架兜底关闭 reader body**：当 `c.Body` 为实现了 `io.Closer` 的 reader，且最终未通过 Pipe 流式输出（处理链出错、或被 `BodyBuffer` 覆盖）时，框架会在响应结束前自动关闭它，修复了 v1 中此路径的文件句柄泄漏窗口（如 static serve、`SendFile` 打开的文件）。

### v2 修复的 v1 缺陷

- **proxy**：并发请求共享同一个错误变量，可能导致 A 请求的代理错误被 B 请求返回（数据竞争）；错误改为通过请求级 context 传递。同时静态 Target 场景下 `ReverseProxy` 只构建一次并复用。
- **renderer**：配置了 `ViewPath` 时，仅传 `Text` 的模板会被错误地当文件渲染。
- **static_serve**：路径前缀校验按分隔符边界比较，修复同前缀兄弟目录（如 `public` 与 `publicsecret`）可被越权访问的问题。
- **tracker**：字段截断改为按 rune 边界，不再产生非法 UTF-8。
- **body_parser**：读取出错时 reader 池对象漏归还（现已移除不必要的池）。
- **cache**：`CacheResponse.Bytes` 对 `IgnoreHeaders` 的 append 存在底层数组并发写风险，改为拼接新切片。
- **cache**：命中缓存时若上游响应本身带 `Age` 头，会输出重复的 `Age`；`Age` 已加入默认忽略头。
- **cache**：`NewCacheResponse` 解码损坏/截断数据（如自定义 store 返回异常内容）时可能越界 panic，已补齐边界校验，异常数据按 fetch 处理。
- **logger**：预置格式 `LoggerTiny` 使用了不存在的 `{url}` 标签导致 URL 永远输出为空，已修正为 `{uri}`。
- **http_header**：`SetShortHeaders` 的全局字典改为 `atomic.Pointer`，消除运行期调用时的数据竞争（仍建议仅在启动期调用，短头索引会持久化进缓存数据）。
- **body_parser**：读取 body 时若存在 `Content-Length`，预分配容量上限为 `min(Content-Length, Limit)`，避免恶意过大 CL 导致内存放大；非法或非正 CL 改走 buffer 路径，不再按声明长度分配。
- **body_parser**（form urlencoded）：转 JSON 改为 `json.Marshal`，自动转义引号/反斜杠等，避免手工拼接产生非法 JSON 或字段注入。
- **proxy**：rewrite 规则改为按首个 `:` 分割（`SplitN`），replacement 可含 `:`；缺少 `:` 或 pattern 为空的规则在构造时返回错误（不再静默跳过）。

## 依赖升级带来的行为变化

v2 依赖 `hes v1.0.0` 与 `keygrip v1.4.0`，相关行为变化：

- **非 hes 错误被包装后 `Exception=true`**：`hes.Wrap` 对普通 error 默认设置 `StatusCode=500` 与 `Exception=true`，错误文本相应带有 `exception=true` 后缀（如 recover、error、static_serve 等中间件输出）。
- **proxy 上游失败状态码 400 → 502**：旧版 `hes.NewWithError` 隐式默认 400，语义不当；现与 `httputil.ReverseProxy` 默认行为一致，返回 `502 Bad Gateway`。
- **error 中间件对未设置 `StatusCode` 的 `*hes.Error` 兜底为 500**（原会得到 0 并按 200 输出）。
- **`*hes.Error` 视为不可变**：自定义中间件若使用 `hes.Wrap(err)` 后就地修改字段，可能污染共享错误对象（无 opts 时返回链中原指针），应改用 `hes.Wrap(err, hes.WithStatus(...), hes.WithCategory(...))` 等 options 形式。
- **`hes.Error.Err` 字段更名为 `Cause`**，且不再参与 JSON 序列化。
- **`keygrip.New` 对空 keys 会 panic**：elton 内部在构建前已做空值守卫（`SignedKeys` 返回空 keys 时签名 cookie 功能不生效），直接使用 keygrip 的业务代码需自行保证 keys 非空。

## 仓库与工具链

- `go.mod`：`go 1.24`，`hes v1.0.0`、`keygrip v1.4.0`。
- CI 测试矩阵：Go stable（最新稳定版）/ 1.25 / 1.24，lint 仅在 stable 上执行。
