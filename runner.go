package worker

import (
	"sync"
)

type workResult[T any] struct {
	result T
	err    error
}

// UniqueWorkRunner is useful when many routines need to perform the same job that is expected to have the same result
// for all of them. Instead of all the routines executing the work, only one routine is allowed to perform the work
// while the rest of the routines wait for the result.
type UniqueWorkRunner[Identifier comparable, Result any] struct {
	concurrencyChans map[Identifier]chan struct{}
	resChans         map[Identifier][]chan workResult[Result]
	mux              *sync.Mutex
}

// NewUniqueWorkRunner returns a new unique work runner.
func NewUniqueWorkRunner[Identifier comparable, Result any]() *UniqueWorkRunner[Identifier, Result] {
	return &UniqueWorkRunner[Identifier, Result]{
		concurrencyChans: make(map[Identifier]chan struct{}),
		resChans:         make(map[Identifier][]chan workResult[Result]),
		mux:              &sync.Mutex{},
	}
}

// Run runs the work with the given ID and returns the result.
// If multiple calls to Run with the same id happen concurrently, only the first call will actually run the work;
// the other calls will wait for the result of the work already being performed.
func (q *UniqueWorkRunner[Identifier, Result]) Run(id Identifier, work func() (Result, error)) (Result, error) {
	resCh := make(chan workResult[Result])
	concurrencyCh := make(chan struct{}, 1)

	q.mux.Lock()
	// Problem: If in the `select` later we want to read from q.concurrencyChans[id], it would be a concurrent map read;
	// it cannot be protected with a mutex because we are in a switch statement.
	//
	// Solution: We set the value in the map to the local variable if it hasn't been set yet, or else use the value that
	// is stored in the map and store it in a local variable, so that the `select` won't be doing a map read while some
	// other routine might be writing to that same map, but will read our local value instead.
	if q.concurrencyChans[id] == nil {
		q.concurrencyChans[id] = concurrencyCh
	} else {
		concurrencyCh = q.concurrencyChans[id]
	}
	q.resChans[id] = append(q.resChans[id], resCh)
	q.mux.Unlock()

	select {
	case concurrencyCh <- struct{}{}:
		res, err := work()

		// Listen to own result channel to prevent a block when we broadcast the result, which will also by design will be
		// broadcast to the current result channel.
		go func() {
			_ = <-resCh
		}()

		// We use the same mutex as during setup, to prevent the code below from immediately cleaning up the things that
		// are being set up at the start of the function.
		q.mux.Lock()

		q.broadcastResult(id, workResult[Result]{
			result: res,
			err:    err,
		})
		q.cleanUp(id)

		q.mux.Unlock()

		return res, err
	default:
		res := <-resCh
		return res.result, res.err
	}
}

func (q *UniqueWorkRunner[Identifier, Result]) broadcastResult(id Identifier, res workResult[Result]) {
	for _, ch := range q.resChans[id] {
		ch <- res
	}
}

func (q *UniqueWorkRunner[Identifier, Result]) cleanUp(id Identifier) {
	q.resChans[id] = make([]chan workResult[Result], 0, 100) // 100 to prevent many slice grows from happening.
	q.concurrencyChans[id] = nil
}
