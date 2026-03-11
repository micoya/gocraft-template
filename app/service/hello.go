package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"gocraft-template/app/model"
)

var helloServiceTracer = otel.Tracer("hello-service")

// HelloService 提供打招呼相关的业务逻辑
type HelloService struct {
	log *slog.Logger
}

func NewHelloService(log *slog.Logger) *HelloService {
	return &HelloService{log: log}
}

// SayHello 根据名字生成打招呼消息
func (s *HelloService) SayHello(ctx context.Context, name string) (*model.Hello, error) {
	ctx, span := helloServiceTracer.Start(ctx, "SayHello")
	defer span.End()
	span.SetAttributes(
		attribute.KeyValue{Key: "name", Value: attribute.StringValue(name)},
	)

	if name == "" {
		name = "World"
	}

	s.log.InfoContext(ctx, "say hello", slog.String("name", name))

	return &model.Hello{
		ID:        1,
		Message:   fmt.Sprintf("Hello, %s!", name),
		CreatedAt: time.Now(),
	}, nil
}
