package scope

import (
	"context"
	"errors"
	"io"
)

func Catch[I, O any](fn func(context.Context, I) (O, error)) func(context.Context, I) (O, error) {
	return func(ctx context.Context, input I) (output O, err error) {
		defer func() {
			if r := recover(); r != nil {
				var ok bool
				output, ok = r.(O)
				if !ok {
					err = r.(error)
				}
			}
		}()

		return fn(ctx, input)
	}
}

func With[I, O any](fn func(context.Context, func(io.Closer), I) (O, error)) func(context.Context, I) (O, error) {
	errs := make([]error, 0, 4)
	capture := func(closer io.Closer) {
		if err := closer.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	return func(ctx context.Context, input I) (O, error) {
		output, err := fn(ctx, capture, input)
		switch err {
		case nil:
			switch len(errs) {
			case 0:
				return output, nil
			case 1:
				return output, errs[0]
			default:
				return output, errors.Join(errs...)
			}
		default:
			switch len(errs) {
			case 0:
				return output, err
			case 1:
				return output, errors.Join(err, errs[0])
			default:
				s := make([]error, len(errs)+1)
				s[0] = err
				copy(s[1:], errs)
				return output, errors.Join(s...)
			}
		}
	}
}
