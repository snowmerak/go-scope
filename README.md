# go-scope

`go-scope` is a utility library for Go that helps you manage resources and handle exceptions (panics) in a cleaner and safer way.

## Features

- **`Catch`**: Automatically catches panics during function execution and converts them into standard Go `error`s.
- **`With`**: Manages the safe closing of `io.Closer` resources, joining execution errors, resource closing errors, and even panics into a single returned error.
- **Safe Panic Handling**: Prevents program crashes by capturing panics and returning them as error information.
- **Thread-safe**: All utilities maintain independent states for each call, making them safe for use in concurrent environments.

## Installation

```bash
go get github.com/snowmerak/go-scope
```

## Usage

### 1. Catch: Convert Panic to Error

Wrap functions that might panic with `Catch` to handle them using standard Go error handling patterns.

```go
package main

import (
    "context"
    "fmt"
    "github.com/snowmerak/go-scope"
)

func main() {
    fn := func(ctx context.Context, s string) (string, error) {
        if s == "" {
            panic("input is empty")
        }
        return "Hello " + s, nil
    }

    protectedFn := scope.Catch(fn)
    
    // Panic occurs but is returned as an error
    result, err := protectedFn(context.Background(), "")
    if err != nil {
        fmt.Println(err) // "panic caught: input is empty"
    }
}
```

### 2. With: Resource Management and Error Joining

`With` provides a `capture` function to register multiple resources, ensuring they are closed when the function completes. It joins execution errors, closer errors, and panics into one.

```go
package main

import (
    "context"
    "io"
    "os"
    "github.com/snowmerak/go-scope"
)

func main() {
    processFile := scope.With(func(ctx context.Context, capture func(io.Closer), path string) (int64, error) {
        f, err := os.Open(path)
        if err != nil {
            return 0, err
        }
        // Use defer capture(f) instead of manual defer f.Close()
        defer capture(f)

        info, err := f.Stat()
        if err != nil {
            return 0, err
        }
        
        return info.Size(), nil
    })

    size, err := processFile(context.Background(), "example.txt")
}
```

### 3. Wrap: Session-based Error Handling

`Wrap` is ideal for operations that require a session (like a database transaction) where you want to centralize error side-effects (like rolling back or logging).

```go
package main

import (
    "context"
    "database/sql"
    "fmt"
    "github.com/snowmerak/go-scope"
)

func main() {
    // Define a function that uses a session (e.g., *sql.Tx)
    process := scope.Wrap(func(ctx context.Context, check func(error) bool, id int, tx *sql.Tx) (string, error) {
        var name string
        err := tx.QueryRowContext(ctx, "SELECT name FROM users WHERE id = ?", id).Scan(&name)
        
        // Use check(err) to trigger the catcher and return early
        if check(err) {
            return "", err
        }
        
        return name, nil
    }, func(tx *sql.Tx, err error) {
        // Catcher: centralized error cleanup
        fmt.Printf("Error occurred: %v. Rolling back...\n", err)
        tx.Rollback()
    })

    // Prepare session
    db, _ := sql.Open("mysql", "...")
    tx, _ := db.Begin()

    // Pass the session to the wrapped function
    name, err := process(context.Background(), 1, tx)
    if err == nil {
        tx.Commit()
    }
}
```

## License

Refer to the [LICENSE](LICENSE) file for details.
