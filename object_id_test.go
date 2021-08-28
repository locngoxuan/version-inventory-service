package main

import (
	"sync"
	"testing"
	"time"
)

func TestNewObjectId(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(100)                         //using 100 goroutine to generate 10000 ids
	results := make(chan string, 10000) //store result
	for i := 0; i < 100; i++ {
		go func() {
			for i := 0; i < 100; i++ {
				id := NewObjectID().Hex()
				results <- id
			}
			wg.Done()
		}()
	}
	wg.Wait()

	m := make(map[string]bool)
	for i := 0; i < 10000; i++ {
		select {
		case id := <-results:
			if _, ok := m[id]; ok {
				t.Errorf("Found duplicated id: %x", id)
				//return
			} else {
				m[id] = true
			}
		case <-time.After(2 * time.Second):
			t.Errorf("Expect 10000 ids in results, but got %d", i)
			return
		}
	}
}

func BenchmarkGenObjectIDP(t *testing.B) {
	t.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			NewObjectID().Hex()
		}
	})
}

func BenchmarkGenObjectIDS(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewObjectID().Hex()
	}
}
