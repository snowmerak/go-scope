package scope

import (
	"context"
	"errors"
	"fmt"
	"io"
)

type Void struct{}

func Catch[I, O any](fn func(context.Context, I) (O, error)) func(context.Context, I) (O, error) {
	return func(ctx context.Context, input I) (output O, err error) {
		defer func() {
			if r := recover(); r != nil {
				if e, ok := r.(error); ok {
					err = fmt.Errorf("panic caught: %w", e)
				} else {
					err = fmt.Errorf("panic caught: %+v", r)
				}
			}
		}()

		return fn(ctx, input)
	}
}

func With[I, O any](fn func(ctx context.Context, capture func(io.Closer), input I) (O, error)) func(context.Context, I) (O, error) {
	return func(ctx context.Context, input I) (output O, err error) {
		errs := make([]error, 0, 4)

		defer func() {
			if r := recover(); r != nil {
				if e, ok := r.(error); ok {
					e := fmt.Errorf("panic caught: %w", e)
					errs = append(errs, e)
				} else {
					e := fmt.Errorf("panic caught: %+v", r)
					errs = append(errs, e)
				}

				output = *new(O) // zero value
				err = errors.Join(errs...)
			}
		}()

		capture := func(closer io.Closer) {
			if err := closer.Close(); err != nil {
				errs = append(errs, err)
			}
		}

		output, err = fn(ctx, capture, input)
		if err != nil {
			errs = append(errs, err)
		}

		return output, errors.Join(errs...)
	}
}
