package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

/*
Readers-writers problem; rules:

1. Reader reads the data.
2. Writer modifies the data.
3. When writer is modifying the data no other thread can access it.
4. At the same time, one or more readers can read the data.
5. There must be one or more readers and one or more writers.
6. No thread should be starved.
*/

// value and its pointer - data that will be read and modified
var value int = 0
var p = &value

// mutex for control variables and its condition, mutex for accessing data,
var mu sync.Mutex
var db sync.Mutex
var cond = sync.NewCond(&mu)

// waitgroups for readers and writers
var rWg sync.WaitGroup
var wWg sync.WaitGroup

// readerCount - active readers, pendingWriters - number of writers awaiting access
var readerCount = 0
var pendingWriters = 0

// simulating task which takes time, read and write
func sleep() {
	time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)
}

var read = sleep
var write = sleep

// descriptive output
func verboseReader(id int) {
	defer rWg.Done()

	mu.Lock()
	fmt.Printf("reader%d: wants to read\n", id)
	for pendingWriters > 0 {
		fmt.Printf("reader%d: pending writer(s), wait\n", id)
		cond.Wait()
	}

	fmt.Printf("reader%d: no pending writer\n", id)
	readerCount++
	if readerCount == 1 {
		fmt.Printf("reader%d: first reader -> lock db\n", id)
		db.Lock()
	}
	mu.Unlock()

	fmt.Printf("reader%d: read: %d\n", id, *p)
	read()

	mu.Lock()
	fmt.Printf("reader%d: done reading\n", id)
	readerCount--
	if readerCount == 0 {
		fmt.Printf("reader%d: last reader -> unlock db\n", id)
		db.Unlock()
	}
	mu.Unlock()
}

// descriptive output
func verboseWriter(id int) {
	defer wWg.Done()

	mu.Lock()
	pendingWriters++
	fmt.Printf("writer%d: wants to write\n", id)
	mu.Unlock()

	fmt.Printf("writer%d: wait for access to db\n", id)
	db.Lock()
	fmt.Printf("writer%d: got access to db\n", id)
	*p += 5
	fmt.Printf("writer%d: wrote: %d\n", id, *p)
	write()
	fmt.Printf("writer%d: done writing, unlocks access to db\n", id)
	db.Unlock()

	mu.Lock()
	pendingWriters--
	if pendingWriters == 0 {
		fmt.Printf("writer%d: last writer, notify readers\n", id)
		cond.Broadcast()
	}
	mu.Unlock()
}

// just pattern
func reader(id int) {
	defer rWg.Done()

	mu.Lock()
	// block reader until there is no pending writer
	for pendingWriters > 0 {
		cond.Wait()
	}
	// new reader, non-recursive lock
	readerCount++
	if readerCount == 1 {
		db.Lock()
	}
	mu.Unlock()

	// read
	fmt.Printf("Reader%d: %d\n", id, *p)
	read()

	mu.Lock()
	// reader leaves, non-recursive unlock
	readerCount--
	if readerCount == 0 {
		db.Unlock()
	}
	mu.Unlock()
}

// just pattern
func writer(id int) {
	defer wWg.Done()

	// new writer pending, disallow new readers
	mu.Lock()
	pendingWriters++
	mu.Unlock()

	// wait until db lock is available, lock, modify data, unlock
	db.Lock()
	*p += 5
	fmt.Printf("Writer%d: %d\n", id, *p)
	write()
	db.Unlock()

	mu.Lock()
	// writer done, notify readers if there are no more pending writers
	pendingWriters--
	if pendingWriters == 0 {
		cond.Broadcast()
	}
	mu.Unlock()
}

func main() {
	/*
		Options:

		1. Change reader or writer count
		2. Change delay between spawning reader or writer goroutines by adding or substracting sleep() function
		3. Change output option: minimal = {reader(), writer()}, full = {verboseReader(), verboseWriter()}
	*/

	// reader count, writer count
	rCount, wCount := 5, 3
	var spawn sync.WaitGroup

	// inline anonymous goroutines for concurrent reader and writer spawning
	spawn.Add(1)
	go func() {
		defer spawn.Done()

		for i := 0; i < wCount; i++ {
			wWg.Add(1)
			go verboseWriter(i)
			// go writer(i)
			sleep()
		}
		wWg.Wait()
		fmt.Println("writers are done")
	}()

	spawn.Add(1)
	go func() {
		defer spawn.Done()

		for i := 0; i < rCount; i++ {
			rWg.Add(1)
			go verboseReader(i)
			// go reader(i)
			sleep()
		}
		rWg.Wait()
		fmt.Println("readers are done")
	}()

	spawn.Wait()
}
