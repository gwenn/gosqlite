// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sqlite_test

import (
	"strings"
	"testing"

	. "github.com/gwenn/gosqlite"
)

func panicOnError(b *testing.B, err error) {
	if err != nil {
		panic(err)
	}
}

func fill(b *testing.B, db *Conn, n int) {
	panicOnError(b, db.Exec("DROP TABLE IF EXISTS test"))
	panicOnError(b, db.Exec("CREATE TABLE test (id INTEGER PRIMARY KEY NOT NULL, float_num REAL, int_num INTEGER, a_string TEXT)"))
	s, err := db.Prepare("INSERT INTO test (float_num, int_num, a_string) VALUES (?, ?, ?)")
	panicOnError(b, err)

	panicOnError(b, db.Begin())
	for i := 0; i < n; i++ {
		panicOnError(b, s.Exec(float64(i)*3.14, i, "hello"))
	}
	panicOnError(b, s.Finalize())
	panicOnError(b, db.Commit())
}

func BenchmarkValuesScan(b *testing.B) {
	b.StopTimer()
	db, err := Open(":memory:")
	panicOnError(b, err)
	defer db.Close()
	fill(b, db, 1)

	cs, err := db.Prepare("SELECT float_num, int_num, a_string FROM test")
	panicOnError(b, err)
	defer cs.Finalize()

	b.StartTimer()
	for i := 0; i < b.N; i++ {

		values := make([]interface{}, 3)
		if Must(cs.Next()) {
			cs.ScanValues(values)
		}
		/*panicOnError(b, */ cs.Reset() /*)*/
	}
}

func BenchmarkScan(b *testing.B) {
	b.StopTimer()
	db, err := Open(":memory:")
	panicOnError(b, err)
	defer db.Close()
	fill(b, db, 1)

	cs, err := db.Prepare("SELECT float_num, int_num, a_string FROM test")
	panicOnError(b, err)
	defer cs.Finalize()

	b.StartTimer()
	for i := 0; i < b.N; i++ {

		var fnum float64
		var inum int64
		var sstr string

		if Must(cs.Next()) {
			/*panicOnError(b, */ cs.Scan(&fnum, &inum, &sstr) /*)*/
		}
		/*panicOnError(b, */ cs.Reset() /*)*/
	}
}

func BenchmarkNamedScan(b *testing.B) {
	b.StopTimer()
	db, err := Open(":memory:")
	panicOnError(b, err)
	defer db.Close()
	fill(b, db, 1)

	cs, err := db.Prepare("SELECT float_num, int_num, a_string FROM test")
	panicOnError(b, err)
	defer cs.Finalize()

	b.StartTimer()
	for i := 0; i < b.N; i++ {

		var fnum float64
		var inum int64
		var sstr string

		if Must(cs.Next()) {
			/*panicOnError(b, */ cs.NamedScan("float_num", &fnum, "int_num", &inum, "a_string", &sstr) /*)*/
		}
		/*panicOnError(b, */ cs.Reset() /*)*/
	}
}

func BenchmarkInsert(b *testing.B) {
	b.StopTimer()
	db, err := Open(":memory:")
	panicOnError(b, err)
	defer db.Close()
	panicOnError(b, db.Exec("DROP TABLE IF EXISTS test"))
	panicOnError(b, db.Exec("CREATE TABLE test (id INTEGER PRIMARY KEY NOT NULL, float_num REAL, int_num INTEGER, a_string TEXT)"))
	s, err := db.Prepare("INSERT INTO test (float_num, int_num, a_string) VALUES (?, ?, ?)")
	panicOnError(b, err)
	defer s.Finalize()

	b.StartTimer()
	panicOnError(b, db.Begin())
	for i := 0; i < b.N; i++ {
		/*panicOnError(b, */ s.Exec(float64(i)*3.14, i, "hello") /*)*/
	}
	panicOnError(b, db.Commit())
}

func BenchmarkNamedInsert(b *testing.B) {
	b.StopTimer()
	db, err := Open(":memory:")
	panicOnError(b, err)
	defer db.Close()
	panicOnError(b, db.Exec("DROP TABLE IF EXISTS test"))
	panicOnError(b, db.Exec("CREATE TABLE test (id INTEGER PRIMARY KEY NOT NULL,"+
		" float_num REAL, int_num INTEGER, a_string TEXT)"))
	s, err := db.Prepare("INSERT INTO test (float_num, int_num, a_string)" +
		" VALUES (:f, :i, :s)")
	panicOnError(b, err)
	defer s.Finalize()

	b.StartTimer()
	panicOnError(b, db.Begin())
	for i := 0; i < b.N; i++ {
		/*panicOnError(b, */ s.NamedBind(":f", float64(i)*3.14, ":i", i, ":s", "hello") /*)*/
		Must(s.Next())
	}
	panicOnError(b, db.Commit())
}

func BenchmarkExec(b *testing.B) {
	b.StopTimer()
	db, err := Open(":memory:")
	panicOnError(b, err)
	defer db.Close()

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		panicOnError(b, db.Exec(strings.Repeat("BEGIN;ROLLBACK;", 5)))
	}
}

func BenchmarkFastExec(b *testing.B) {
	b.StopTimer()
	db, err := Open(":memory:")
	panicOnError(b, err)
	defer db.Close()

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		panicOnError(b, db.FastExec(strings.Repeat("BEGIN;ROLLBACK;", 5)))
	}
}
