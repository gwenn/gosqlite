// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sqlite_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/bmizerany/assert"
	. "github.com/gwenn/gosqlite"
)

func init() {
	if os.Getenv("SQLITE_LOG") == "" {
		err := ConfigLog(func(d interface{}, err error, msg string) {
			fmt.Printf("SQLITE: %s, %s\n", err, msg)
		}, nil)
		if err != nil {
			panic(fmt.Sprintf("couldn't config log: '%s'", err))
		}
		err = ConfigLog(nil, nil)
		if err != nil {
			panic(fmt.Sprintf("couldn't unset logger: '%s'", err))
		}
	}
}

func trace(d interface{}, sql string) {
	if t, ok := d.(*testing.T); ok {
		t.Logf("TRACE: %s\n", sql)
	} else {
		fmt.Printf("%s: %s\n", d, sql)
	}
}

func authorizer(d interface{}, action Action, arg1, arg2, dbName, triggerName string) Auth {
	if t, ok := d.(*testing.T); ok {
		t.Logf("AUTH: %s, %s, %s, %s, %s\n", action, arg1, arg2, dbName, triggerName)
	} else {
		fmt.Printf("%s: %s, %s, %s, %s, %s\n", d, action, arg1, arg2, dbName, triggerName)
	}
	return AuthOk
}

func profile(d interface{}, sql string, duration time.Duration) {
	if t, ok := d.(*testing.T); ok {
		t.Logf("PROFILE: %s = %s\n", sql, duration)
	} else {
		fmt.Printf("%s: %s = %s\n", d, sql, duration)
	}
}

func progressHandler(d interface{}) bool {
	if t, ok := d.(*testing.T); ok {
		t.Log("+")
	} else {
		fmt.Print("+")
	}
	return false
}

func TestNoTrace(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)
	db.Trace(nil, nil)
	db.SetAuthorizer(nil, nil)
	db.Profile(nil, nil)
	db.ProgressHandler(nil, 0, nil)
	db.BusyHandler(nil, nil)
}

func TestTrace(t *testing.T) {
	skipIfCgoCheckActive(t)

	db := open(t)
	defer checkClose(db, t)
	db.Trace(trace, t)
	err := db.SetAuthorizer(authorizer, t)
	checkNoError(t, err, "couldn't set an authorizer: %s")
	db.Profile(profile, t)
	db.ProgressHandler(progressHandler, 1, t)
	b, err := db.Exists("SELECT 1 WHERE 1 = ?", 1)
	checkNoError(t, err, "error while executing stmt: %s")
	assert.T(t, b, "exists")
}

func TestProfile(t *testing.T) {
	skipIfCgoCheckActive(t)

	db := open(t)
	defer checkClose(db, t)
	db.Profile(profile, t)
	defer db.Profile(nil, nil)
	createTable(db, t)
}

func TestLog(t *testing.T) {
	Log(0, "One message")
}

func TestMemory(t *testing.T) {
	used := MemoryUsed()
	assert.T(t, used >= 0, "memory used")
	highwater := MemoryHighwater(false)
	assert.T(t, highwater >= 0, "memory highwater")
	limit := SoftHeapLimit()
	assert.T(t, limit >= 0, "soft heap limit positive")
}

func TestExplainQueryPlan(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)
	createTable(db, t)
	s, err := db.Prepare("SELECT * FROM test WHERE a_string like ?")
	checkNoError(t, err, "error while preparing stmt: %s")
	defer checkFinalize(s, t)
	w, err := os.Open(os.DevNull)
	checkNoError(t, err, "couldn't open /dev/null: %s")
	defer w.Close()
	err = s.ExplainQueryPlan(w)
	checkNoError(t, err, "error while explaining query plan: %s")

	e, err := db.Prepare("")
	checkNoError(t, err, "error while preparing stmt: %s")
	defer checkFinalize(e, t)
	err = e.ExplainQueryPlan(w)
	assert.T(t, err != nil)
}
