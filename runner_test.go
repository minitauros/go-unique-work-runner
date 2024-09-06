package worker

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func Test_UniqueWorkRunner_Run(t *testing.T) {
	Convey("UniqueWorkRunner_Run()", t, func() {
		runner := &UniqueWorkRunner[int, int]{
			concurrencyChans: make(map[int]chan struct{}),
			resChans:         make(map[int][]chan workResult[int]),
			mux:              &sync.Mutex{},
		}
		wg := sync.WaitGroup{}

		Convey("If multiple routines try to execute the same work", func() {
			Convey("If there is one unique unit of work to be done, the work is only executed once", func() {
				var numWorkExecuted atomic.Int64

				for i := 0; i < 100; i++ {
					wg.Add(1)
					go func() {
						_, _ = runner.Run(1, func() (int, error) {
							// Make sure the work takes long enough for all the non-work-performing routines to queue up and wait for
							// the result.
							time.Sleep(50 * time.Millisecond)

							numWorkExecuted.Add(1)

							return 1, nil
						})
						wg.Done()
					}()
				}

				wg.Wait()

				So(numWorkExecuted.Load(), ShouldEqual, 1)
			})

			Convey("If there is more than one unique unit of work to be done, each unit work is only executed once", func() {
				numWorkExecuted := make([]*atomic.Int64, 100)

				// Run multiple unique pieces of work.
				for workID := 0; workID < 100; workID++ {
					numWorkExecuted[workID] = &atomic.Int64{}

					// Run each unique piece of work multiple times.
					for i := 0; i < 100; i++ {
						wg.Add(1)
						go func() {
							_, _ = runner.Run(workID, func() (int, error) {
								// Make sure the work takes long enough for all the non-work-performing routines to queue up and wait for
								// the result.
								time.Sleep(50 * time.Millisecond)

								numWorkExecuted[workID].Add(1)

								return 1, nil
							})
							wg.Done()
						}()
					}
				}

				wg.Wait()

				for _, val := range numWorkExecuted {
					So(val.Load(), ShouldEqual, 1)
				}
			})
		})

		Convey("If one routine tries to execute the work, the work is executed once for each call", func() {
			var numWorkExecuted int

			for i := 0; i < 100; i++ {
				_, _ = runner.Run(1, func() (int, error) {
					numWorkExecuted++

					return 1, nil
				})
			}

			So(numWorkExecuted, ShouldEqual, 100)
		})
	})
}
