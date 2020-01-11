// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sqlite_test

import (
	"fmt"
	"os"
	"path"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/bmizerany/assert"
	. "github.com/gwenn/gosqlite"
)

func checkNoError(t *testing.T, err error, format string) {
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		t.Fatalf("\n%s:%d: %s", path.Base(file), line, fmt.Sprintf(format, err))
	}
}

func skipIfCgoCheckActive(t *testing.T) {
	if /*runtime.Version() >= "go1.6" &&*/ !strings.Contains(os.Getenv("GODEBUG"), "cgocheck=0") {
		t.Skip("cgocheck")
	}
}

func Must(b bool, err error) bool {
	if err != nil {
		panic(err)
	}
	return b
}

func open(t *testing.T) *Conn {
	db, err := Open(":memory:", OpenReadWrite, OpenCreate, OpenFullMutex /*OpenNoMutex*/)
	checkNoError(t, err, "couldn't open database file: %s")
	if db == nil {
		t.Fatal("opened database is nil")
	}
	//db.SetLockingMode("", "exclusive")
	//db.SetSynchronous("", 0)
	//db.Profile(profile, t)
	//db.Trace(trace, t)
	if false /*testing.Verbose()*/ { // Go 1.1
		db.SetAuthorizer(authorizer, t)
	}
	return db
}

func checkClose(db *Conn, t *testing.T) {
	checkNoError(t, db.Close(), "Error closing database: %s")
}

func createTable(db *Conn, t *testing.T) {
	err := db.Exec("DROP TABLE IF EXISTS test;" +
		"CREATE TABLE test (id INTEGER PRIMARY KEY NOT NULL," +
		" float_num REAL, int_num INTEGER, a_string TEXT); -- bim")
	checkNoError(t, err, "error creating table: %s")
}

func TestVersion(t *testing.T) {
	v := Version()
	if !strings.HasPrefix(v, "3") {
		t.Fatalf("unexpected library version: %s", v)
	}
}

func TestOpen(t *testing.T) {
	db := open(t)
	checkNoError(t, db.Close(), "Error closing database: %s")
}

func TestOpenFailure(t *testing.T) {
	db, err := Open("doesnotexist.sqlite", OpenReadOnly)
	assert.T(t, db == nil && err != nil, "open failure expected")
	//println(err.Error())
}

func TestCreateTable(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)
	createTable(db, t)
}

func TestManualTransaction(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)
	checkNoError(t, db.Begin(), "Error while beginning transaction: %s")
	if err := db.Begin(); err == nil {
		t.Fatalf("Error expected (transaction cannot be nested)")
	}
	checkNoError(t, db.Commit(), "Error while committing transaction: %s")
	checkNoError(t, db.BeginTransaction(Immediate), "Error while beginning immediate transaction: %s")
	checkNoError(t, db.Commit(), "Error while committing transaction: %s")
	checkNoError(t, db.BeginTransaction(Exclusive), "Error while beginning immediate transaction: %s")
	checkNoError(t, db.Commit(), "Error while committing transaction: %s")
}

func TestSavepoint(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)
	checkNoError(t, db.Savepoint("1"), "Error while creating savepoint: %s")
	checkNoError(t, db.Savepoint("2"), "Error while creating savepoint: %s")
	checkNoError(t, db.RollbackSavepoint("2"), "Error while creating savepoint: %s")
	checkNoError(t, db.ReleaseSavepoint("1"), "Error while creating savepoint: %s")
}

func TestExists(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)
	b, err := db.Exists("SELECT 1 WHERE 1 = 0")
	checkNoError(t, err, "%s")
	assert.T(t, !b, "no row expected")
	b, err = db.Exists("SELECT 1 WHERE 1 = 1")
	checkNoError(t, err, "%s")
	assert.T(t, b, "one row expected")

	_, err = db.Exists("SELECT 1", 1)
	assert.T(t, err != nil)
	//println(err.Error())

	_, err = db.Exists("SELECT 1 FROM test")
	assert.T(t, err != nil)
	//println(err.Error())

	_, err = db.Exists("PRAGMA shrink_memory")
	assert.T(t, err != nil)
	//println(err.Error())
}

func TestInsert(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)
	createTable(db, t)
	db.Begin()
	for i := 0; i < 1000; i++ {
		ierr := db.Exec("INSERT INTO test (float_num, int_num, a_string) VALUES (?, ?, ?)", float64(i)*3.14, i, "hello")
		checkNoError(t, ierr, "insert error: %s")
		c := db.Changes()
		assert.Equal(t, 1, c, "changes")
	}
	checkNoError(t, db.Commit(), "Error: %s")

	lastID := db.LastInsertRowid()
	assert.Equal(t, int64(1000), lastID, "last insert row id")

	cs, _ := db.Prepare("SELECT COUNT(*) FROM test")
	defer checkFinalize(cs, t)

	paramCount := cs.BindParameterCount()
	assert.Equal(t, 0, paramCount, "bind parameter count")
	columnCount := cs.ColumnCount()
	assert.Equal(t, 1, columnCount, "column count")

	assert.T(t, checkStep(t, cs))
	assert.Equal(t, columnCount, cs.DataCount(), "column & data count expected to be equal")
	var i int
	checkNoError(t, cs.Scan(&i), "error scanning count: %s")
	assert.Equal(t, 1000, i, "count")
	if checkStep(t, cs) {
		t.Fatal("Only one row expected")
	}
	assert.T(t, !cs.Busy(), "expected statement to be reset")
}

/*
func TestLoadExtension(t *testing.T) { // OMIT_LOAD_EXTENSION
	db := open(t)

	db.EnableLoadExtension(true)

	err := db.LoadExtension("/tmp/myext.so")
	checkNoError(t, err, "load extension error: %s")
}
*/

func TestOpenSameMemoryDb(t *testing.T) {
	db1, err := Open("file:dummy.db?mode=memory&cache=shared", OpenURI, OpenReadWrite, OpenCreate, OpenFullMutex)
	checkNoError(t, err, "open error: %s")
	defer checkClose(db1, t)
	err = db1.Exec("CREATE TABLE test (data TEXT)")
	checkNoError(t, err, "exec error: %s")

	db2, err := Open("file:dummy.db?mode=memory&cache=shared", OpenURI, OpenReadWrite, OpenCreate, OpenFullMutex)
	checkNoError(t, err, "open error: %s")
	defer checkClose(db2, t)
	_, err = db2.Exists("SELECT 1 FROM test")
	checkNoError(t, err, "exists error: %s")
}

func TestConnExecWithSelect(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)

	err := db.Exec("SELECT 1")
	assert.T(t, err != nil, "error expected")
	if serr, ok := err.(StmtError); ok {
		assert.Equal(t, ErrSpecific, serr.Code())
	} else {
		t.Errorf("got %s; want StmtError", reflect.TypeOf(err))
	}
}

func TestConnInitialState(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)
	autoCommit := db.GetAutocommit()
	assert.T(t, autoCommit, "autocommit expected to be active by default")
	totalChanges := db.TotalChanges()
	assert.Equal(t, 0, totalChanges, "total changes")
	err := db.LastError()
	assert.Equal(t, nil, err, "expected last error to be nil")
	readonly, err := db.Readonly("main")
	checkNoError(t, err, "Readonly status error: %s")
	assert.T(t, !readonly, "readonly expected to be unset by default")
}

func TestReadonlyMisuse(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)
	_, err := db.Readonly("doesnotexist")
	assert.T(t, err != nil, "error expected")
	_ = err.Error()
	//println(err.Error())
}

func TestComplete(t *testing.T) {
	c, err := Complete("SELECT 1;")
	checkNoError(t, err, "error: %s")
	assert.T(t, c, "expected complete statement")
}

func TestExecMisuse(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)
	createTable(db, t)
	err := db.Exec("INSERT INTO test VALUES (?, ?, ?, ?); INSERT INTO test VALUES (?, ?, ?, ?)", 0, 273.1, 1, "test")
	assert.T(t, err != nil, "exec misuse expected")
	//println(err.Error())

	err = db.Exec("CREATE FUNCTION incr(i INT) RETURN i+1;")
	assert.T(t, err != nil, "syntax error expected")
	//println(err.Error())
}

func TestExecTwice(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)
	createTable(db, t)
	err := db.Exec("INSERT INTO test VALUES (?, ?, ?, ?); INSERT INTO test VALUES (?, ?, ?, ?)",
		0, 273.1, 1, "test",
		1, 2257, 2, nil)
	checkNoError(t, err, "Exec error: %s")
}

func TestTransaction(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)
	createTable(db, t)
	err := db.Transaction(Immediate, func(_ *Conn) error {
		err := db.Transaction(Immediate, func(__ *Conn) error {
			return db.Exec("INSERT INTO test VALUES (?, ?, ?, ?)", 0, 273.1, 1, "test")
		})
		checkNoError(t, err, "error: %s")
		return err
	})
	checkNoError(t, err, "error: %s")
}

func TestCommitMisuse(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)

	err := db.Commit()
	assert.T(t, err != nil, "error expected")
	if cerr, ok := err.(ConnError); ok {
		assert.Equal(t, ErrError, cerr.Code())
		assert.Equal(t, 1, cerr.ExtendedCode())
	} else {
		t.Errorf("got %s; want ConnError", reflect.TypeOf(err))
	}
	assert.Equal(t, err, db.LastError())
}

func TestNilDb(t *testing.T) {
	var db *Conn
	err := db.Exec("DROP TABLE IF EXISTS test")
	assert.T(t, err != nil)
	//println(err.Error())

	err = db.Close()
	assert.T(t, err != nil)
	//println(err.Error())

	err = db.LastError()
	assert.T(t, err != nil)
	//println(err.Error())

	_, err = db.ForeignKeyCheck("", "")
	assert.T(t, err != nil)

	_, err = db.Databases()
	assert.T(t, err != nil)
}

func TestError(t *testing.T) {
	err := ErrMisuse
	assert.T(t, err.Error() != "")
}

func TestOneValueMisuse(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)

	var value interface{}
	err := db.OneValue("SELECT 1", &value, 1)
	assert.T(t, err != nil)
	//println(err.Error())

	err = db.OneValue("SELECT 1 FROM test", &value)
	assert.T(t, err != nil)
	//println(err.Error())

	err = db.OneValue("PRAGMA shrink_memory", &value)
	assert.T(t, err != nil)
	//println(err.Error())

	err = db.OneValue("SELECT 1 LIMIT 0", &value)
	assert.T(t, err != nil)
	//println(err.Error())

}
