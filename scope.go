package scope

import (
	"context"
	"errors"
	"fmt"
	"io"
)

func Catch[I, O any](fn func(context.Context, I) (O, error)) func(context.Context, I) (O, error) {
	return func(ctx context.Context, input I) (output O, err error) {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("panic caught: %+v", r)
			}
		}()

		return fn(ctx, input)
	}
}

func With[I, O any](fn func(context.Context, func(io.Closer), I) (O, error)) func(context.Context, I) (O, error) {
	return func(ctx context.Context, input I) (O, error) {
		errs := make([]error, 0, 4)
		capture := func(closer io.Closer) {
			if err := closer.Close(); err != nil {
				errs = append(errs, err)
			}
		}

		output, err := fn(ctx, capture, input)
		if err != nil {
			errs = append(errs, err)
		}
		return output, errors.Join(errs...)
	}
}
