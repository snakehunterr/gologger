package gologger

import (
	"context"
	"errors"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func Test_LoggerServiceNoNilPanic(t *testing.T) {
	t.Run("no panic if LoggerService is nil", func(t *testing.T) {
		logger := NewLoggerService("test")

		do := func(logger *LoggerService) {
			ctx := context.Background()
			c := fiber.Ctx{}
			logger.
				Error().
				Err(errors.New("some")).
				FiberCtx(&c).
				Ctx(ctx).
				Stack().
				StatusCode(200).
				Msg("some")
		}

		do(logger)
		do(nil)
	})
}
