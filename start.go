package xtractr

import (
	"fmt"
	"os"
	"reflect"
)

// Config is the input data to configure the Xtract queue. Fill this out and
// pass it into NewQueue() to create a queue for archive extractions.
type Config struct {
	// Use -1 for unbuffered channel. Not recommend.
	BuffSize int         // Size of the extraction channel buffer. Default=1000.
	Parallel int         // Number of concurrent extractions.
	FileMode os.FileMode // Filemode used when writing files, tar ignores this.
	DirMode  os.FileMode // Filemode used when writing folders, tar ignores this.
	Suffix   string      // The suffix used for temporary folders.
	Logger               // Logs are sent to this Logger.
}

// Logger allows this library to write logs.
// Use this to capture them in your own flow.
type Logger interface {
	Printf(string, ...interface{})
	Debugf(string, ...interface{})
}

// Xtractr is what you get from NewQueue(). This is the main app struct.
// Use this struct to call Xtractr.Extract() to queue an extraction.
type Xtractr struct {
	config *Config
	queue  chan *Xtract
	done   chan struct{}
}

// Custom errors returned by this module.
var (
	ErrQueueStopped       = fmt.Errorf("extractor queue stopped")
	ErrNoCompressedFiles  = fmt.Errorf("no compressed files found")
	ErrUnknownArchiveType = fmt.Errorf("unknown archive file type")
	ErrInvalidPath        = fmt.Errorf("archived file contains invalid path")
	ErrInvalidHead        = fmt.Errorf("archived file contains invalid header file")
	ErrQueueRunning       = fmt.Errorf("queue is running, cannot start")
	ErrNoConfig           = fmt.Errorf("call NewQueue() to initialize a queue")
	ErrNoLogger           = fmt.Errorf("xtractr.Config.Logger must be non-nil")
)

// NewQueue returns a new Xtractr Queue you can send Xtract jobs into.
// This is where to start if you're creating an extractor queue.
func NewQueue(config *Config) *Xtractr {
	x := parseConfig(config)

	err := x.Start()
	if err != nil {
		panic(err)
	}

	return x
}

// Start restarts the queue. This can be called only after you call Stop().
func (x *Xtractr) Start() error {
	if x.queue != nil {
		return ErrQueueRunning
	}

	if x.config == nil {
		return ErrNoConfig
	}

	if x.config.Logger == nil {
		return ErrNoLogger
	}

	x.queue = make(chan *Xtract, x.config.BuffSize)

	for i := 0; i < x.config.Parallel; i++ {
		go x.processQueue()
	}

	return nil
}

// DefaultBufferSize is the size of the extraction buffer.
// ie. How many jobs can be queued before things get slow.
const DefaultBufferSize = 1000

// parseConfig verifies sane config data and returns the Xtractr struct.
func parseConfig(config *Config) *Xtractr {
	if config.Parallel < 1 {
		config.Parallel = 1
	}

	if config.BuffSize == 0 {
		config.BuffSize = DefaultBufferSize
	} else if config.BuffSize < 0 {
		config.BuffSize = 0
	}

	if config.Suffix == "" {
		config.Suffix = "_" + reflect.TypeOf(Config{}).PkgPath() // xtractr
	}

	return &Xtractr{
		config: config,
		done:   make(chan struct{}),
	}
}

// Stop shuts down the extractor routines. Call this to shut things down.
func (x *Xtractr) Stop() {
	if x.queue == nil {
		return
	}

	close(x.queue)

	// Wait until all running extractions are done.
	for i := 0; i < x.config.Parallel; i++ {
		<-x.done
	}

	x.queue = nil
}
