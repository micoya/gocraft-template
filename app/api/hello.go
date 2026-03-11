package api

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"gocraft-template/app/common"
	"gocraft-template/app/service"
)

// HelloHandler 处理打招呼相关的 HTTP 请求
type HelloHandler struct {
	svc *service.HelloService
	log *slog.Logger
}

func NewHelloHandler(svc *service.HelloService, log *slog.Logger) *HelloHandler {
	return &HelloHandler{svc: svc, log: log}
}

// Welcome GET /
func (h *HelloHandler) Welcome(c *gin.Context) {
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(`<!doctype html><html><head><meta charset="utf-8"><title>gocraft ✨</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{min-height:100vh;display:flex;align-items:center;justify-content:center;background:linear-gradient(135deg,#fce4ec,#e8eaf6);font-family:sans-serif}
.card{background:#fff;border-radius:24px;padding:48px 56px;text-align:center;box-shadow:0 8px 32px rgba(0,0,0,.08)}
.emoji{font-size:64px;margin-bottom:16px}
h1{font-size:2rem;color:#5c6bc0;margin-bottom:8px}
p{color:#90a4ae;font-size:1rem;margin-bottom:24px}
.badge{display:inline-block;background:#f3e5f5;color:#ab47bc;border-radius:999px;padding:4px 16px;font-size:.85rem}
</style></head><body>
<div class="card">
  <div class="emoji">🚀</div>
  <h1>Welcome to gocraft!</h1>
  <p>Your service is up and running ✨</p>
  <span class="badge">powered by gocraft</span>
</div>
</body></html>`))
}

// SayHello GET /api/v1/hello?name=xxx
func (h *HelloHandler) SayHello(c *gin.Context) {
	ctx := c.Request.Context()
	name := c.DefaultQuery("name", "World")

	h.log.InfoContext(ctx, "handle say hello request", slog.String("name", name))

	hello, err := h.svc.SayHello(ctx, name)
	if err != nil {
		h.log.ErrorContext(ctx, "say hello failed", slog.Any("error", err))
		common.InternalError(c, err.Error())
		return
	}

	common.OK(c, hello)
}
