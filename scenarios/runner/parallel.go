package main

import "sync"

// gateSize is how many runs may proceed at once. A limit below 1 means all of them.
func gateSize(n, limit int) int {
	if n < 1 {
		return 1
	}
	if limit < 1 || limit > n {
		return n
	}
	return limit
}

// runLimited calls fn for each index. At most gateSize(n, limit) calls run at once.
func runLimited(n, limit int, fn func(i int) error) error {
	if n < 1 {
		return nil
	}
	limit = gateSize(n, limit)
	sem := make(chan struct{}, limit)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var first error
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			if err := fn(i); err != nil {
				mu.Lock()
				if first == nil {
					first = err
				}
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()
	return first
}
