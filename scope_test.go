package scope

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

type mockCloser struct {
	err error
}

func (m *mockCloser) Close() error {
	return m.err
}

func TestCatch(t *testing.T) {
	t.Run("Normal", func(t *testing.T) {
		fn := func(ctx context.Context, i int) (int, error) {
			return i * 2, nil
		}
		caught := Catch(fn)
		out, err := caught(context.Background(), 5)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if out != 10 {
			t.Errorf("expected 10, got %d", out)
		}
	})

	t.Run("PanicError", func(t *testing.T) {
		cause := errors.New("original error")
		fn := func(ctx context.Context, i int) (int, error) {
			panic(cause)
		}
		caught := Catch(fn)
		_, err := caught(context.Background(), 5)
		if err == nil {
			t.Fatal("expected error from panic, got nil")
		}
		if !errors.Is(err, cause) {
			t.Errorf("expected error to wrap cause, but errors.Is failed")
		}
		if !strings.Contains(err.Error(), "panic caught") {
			t.Errorf("expected error message to contain 'panic caught', got %q", err.Error())
		}
	})

	t.Run("PanicString", func(t *testing.T) {
		fn := func(ctx context.Context, i int) (int, error) {
			panic("something went wrong")
		}
		caught := Catch(fn)
		_, err := caught(context.Background(), 5)
		if err == nil {
			t.Fatal("expected error from panic, got nil")
		}
		if !strings.Contains(err.Error(), "something went wrong") {
			t.Errorf("expected error message to contain panic string, got %q", err.Error())
		}
	})
}

func TestWith(t *testing.T) {
	t.Run("Normal", func(t *testing.T) {
		fn := func(ctx context.Context, capture func(io.Closer), i int) (int, error) {
			closer := &mockCloser{err: nil}
			capture(closer)
			return i + 1, nil
		}
		w := With(fn)
		out, err := w(context.Background(), 10)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if out != 11 {
			t.Errorf("expected 11, got %d", out)
		}
	})

	t.Run("FnError", func(t *testing.T) {
		fnErr := errors.New("fn error")
		fn := func(ctx context.Context, capture func(io.Closer), i int) (int, error) {
			return 0, fnErr
		}
		w := With(fn)
		_, err := w(context.Background(), 0)
		if !errors.Is(err, fnErr) {
			t.Errorf("expected error to include fnErr, got %v", err)
		}
	})

	t.Run("CloserError", func(t *testing.T) {
		closeErr := errors.New("close error")
		fn := func(ctx context.Context, capture func(io.Closer), i int) (int, error) {
			capture(&mockCloser{err: closeErr})
			return 100, nil
		}
		w := With(fn)
		out, err := w(context.Background(), 0)
		if !errors.Is(err, closeErr) {
			t.Errorf("expected error to include closeErr, got %v", err)
		}
		if out != 100 {
			t.Errorf("expected output 100 even if closer fails, got %d", out)
		}
	})

	t.Run("MultipleErrors", func(t *testing.T) {
		fnErr := errors.New("fn error")
		closeErr := errors.New("close error")
		fn := func(ctx context.Context, capture func(io.Closer), i int) (int, error) {
			capture(&mockCloser{err: closeErr})
			return 0, fnErr
		}
		w := With(fn)
		_, err := w(context.Background(), 0)
		if !errors.Is(err, fnErr) || !errors.Is(err, closeErr) {
			t.Errorf("expected error to join both, got %v", err)
		}
	})

	t.Run("PanicHandling", func(t *testing.T) {
		fn := func(ctx context.Context, capture func(io.Closer), i int) (int, error) {
			panic("oops")
		}
		w := With(fn)
		out, err := w(context.Background(), 0)
		if err == nil {
			t.Fatal("expected error from panic, got nil")
		}
		if !strings.Contains(err.Error(), "panic caught: oops") {
			t.Errorf("unexpected error message: %v", err)
		}
		if out != 0 {
			t.Errorf("expected zero value output on panic, got %d", out)
		}
	})
}

func TestWrap(t *testing.T) {
	type Session struct {
		IsRolledBack bool
		Logs         []string
	}

	t.Run("NormalExecute", func(t *testing.T) {
		session := &Session{}
		fn := func(ctx context.Context, check func(error) bool, input string, s *Session) (int, error) {
			s.Logs = append(s.Logs, "started")
			return len(input), nil
		}
		catcher := func(s *Session, err error) {
			s.IsRolledBack = true
		}

		wrapped := Wrap(fn, catcher)
		out, err := wrapped(context.Background(), "hello", session)

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if out != 5 {
			t.Errorf("expected 5, got %d", out)
		}
		if session.IsRolledBack {
			t.Error("catcher should not be called on success")
		}
	})

	t.Run("ErrorHandlingWithCheck", func(t *testing.T) {
		session := &Session{}
		expectedErr := errors.New("something failed")

		fn := func(ctx context.Context, check func(error) bool, input string, s *Session) (int, error) {
			if check(expectedErr) {
				return 0, expectedErr
			}
			return 1, nil
		}
		catcher := func(s *Session, err error) {
			s.IsRolledBack = true
			s.Logs = append(s.Logs, "catcher called: "+err.Error())
		}

		wrapped := Wrap(fn, catcher)
		_, err := wrapped(context.Background(), "test", session)

		if !errors.Is(err, expectedErr) {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
		if !session.IsRolledBack {
			t.Error("expected catcher to be called")
		}
		if len(session.Logs) == 0 || session.Logs[0] != "catcher called: something failed" {
			t.Errorf("unexpected logs: %v", session.Logs)
		}
	})

	t.Run("PanicHandling", func(t *testing.T) {
		session := &Session{}
		fn := func(ctx context.Context, check func(error) bool, input string, s *Session) (int, error) {
			panic("unexpected panic")
		}
		catcher := func(s *Session, err error) {
			s.IsRolledBack = true
		}

		wrapped := Wrap(fn, catcher)
		out, err := wrapped(context.Background(), "test", session)

		if err == nil {
			t.Error("expected error from panic, got nil")
		}
		if out != 0 {
			t.Errorf("expected zero value on panic, got %d", out)
		}
		// Note: Based on current scope.go implementation, catcher is not called in defer for Catch/Wrap
		// unless we explicitly added it.
	})
}
