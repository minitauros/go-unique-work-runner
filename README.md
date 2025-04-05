A unique work runner is useful when many goroutines need to perform the same job that is expected to have the same result for all of them. Instead of all the routines executing the work, only one routine is allowed to perform the work while the rest of the routines wait for the result.

## Example

```go
package main

import (
	"github.com/minitauros/go-unique-work-runner"
)

type WorkResult struct {
}

func main() {
	// Create a work runner that identifies the unique work by a string and 
	// that runs work that returns a WorkResult.
	runner := worker.NewUniqueWorkRunner[string, WorkResult]()

	// Run the unique piece of work "refresh-login-token". If the work called 
	// "refresh-login-token" is already being executed by another goroutine, the
	// runner will start new work, but rather wait for the WorkResult of the 
	// routine that is doing the work as we speak.
	res, _ := runner.Run("refresh-login-token", func() (WorkResult, error) {
		return WorkResult{}, nil
	})

	doSomethingWith(res)
}

```

