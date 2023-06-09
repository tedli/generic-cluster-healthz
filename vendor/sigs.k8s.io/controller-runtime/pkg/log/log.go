/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package log contains utilities for fetching a new logger
// when one is not already available.
//
// # The Log Handle
//
// This package contains a root logr.Logger Log.  It may be used to
// get a handle to whatever the root logging implementation is.  By
// default, no implementation exists, and the handle returns "promises"
// to loggers.  When the implementation is set using SetLogger, these
// "promises" will be converted over to real loggers.
//
// # Logr
//
// All logging in controller-runtime is structured, using a set of interfaces
// defined by a package called logr
// (https://pkg.go.dev/github.com/go-logr/logr).  The sub-package zap provides
// helpers for setting up logr backed by Zap (go.uber.org/zap).
package log

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"
	"sync/atomic"
	"time"

	"github.com/go-logr/logr"
)

// SetLogger sets a concrete logging implementation for all deferred Loggers.
func SetLogger(l logr.Logger) {
	logFullfilled.Store(true)
	rootLog.Fulfill(l.GetSink())
}

func eventuallyFulfillRoot() {
	if logFullfilled.Load() {
		return
	}
	if time.Since(rootLogCreated).Seconds() >= 30 {
		if logFullfilled.CompareAndSwap(false, true) {
			fmt.Fprintf(os.Stderr, "[controller-runtime] log.SetLogger(...) was never called, logs will not be displayed:\n%s", debug.Stack())
			SetLogger(logr.New(NullLogSink{}))
		}
	}
}

var (
	logFullfilled atomic.Bool
)

// Log is the base logger used by kubebuilder.  It delegates
// to another logr.Logger. You *must* call SetLogger to
// get any actual logging. If SetLogger is not called within
// the first 30 seconds of a binaries lifetime, it will get
// set to a NullLogSink.
var (
	rootLog, rootLogCreated = func() (*delegatingLogSink, time.Time) {
		return newDelegatingLogSink(NullLogSink{}), time.Now()
	}()
	Log = logr.New(rootLog)
)

// FromContext returns a logger with predefined values from a context.Context.
func FromContext(ctx context.Context, keysAndValues ...interface{}) logr.Logger {
	log := Log
	if ctx != nil {
		if logger, err := logr.FromContext(ctx); err == nil {
			log = logger
		}
	}
	return log.WithValues(keysAndValues...)
}

// IntoContext takes a context and sets the logger as one of its values.
// Use FromContext function to retrieve the logger.
func IntoContext(ctx context.Context, log logr.Logger) context.Context {
	return logr.NewContext(ctx, log)
}
