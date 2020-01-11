// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sqlite_test

import (
	"fmt"
	"math"
	"os"
	"path"
	"reflect"
	"runtime"
	"testing"
	"time"
	"unsafe"

	"github.com/bmizerany/assert"
	. "github.com/gwenn/gosqlite"
)

func checkFinalize(s *Stmt, t *testing.T) {
	checkNoError(t, s.Finalize(), "Error finalizing statement: %s")
}

func checkStep(t *testing.T, s *Stmt) bool {
	b, err := s.Next()
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		t.Fatalf("\n%s:%d: %s", path.Base(file), line, fmt.Sprintf("step error: %s", err))
	}
	return b
}

func TestInsertWithStatement(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)
	createTable(db, t)
	s, serr := db.Prepare("INSERT INTO test (float_num, int_num, a_string) VALUES (:f, :i, :s)")
	checkNoError(t, serr, "prepare error: %s")
	if s == nil {
		t.Fatal("statement is nil")
	}
	defer checkFinalize(s, t)

	assert.T(t, !s.ReadOnly(), "update statement should not be readonly")

	paramCount := s.BindParameterCount()
	assert.Equal(t, 3, paramCount, "bind parameter count")
	firstParamName, berr := s.BindParameterName(1)
	checkNoError(t, berr, "error binding: %s")
	assert.Equal(t, ":f", firstParamName, "bind parameter name")
	lastParamIndex, berr := s.BindParameterIndex(":s")
	checkNoError(t, berr, "error binding: %s")
	assert.Equal(t, 3, lastParamIndex, "bind parameter index")
	columnCount := s.ColumnCount()
	assert.Equal(t, 0, columnCount, "column count")

	db.Begin()
	for i := 0; i < 1000; i++ {
		c, ierr := s.ExecDml(float64(i)*3.14, i, "hello")
		checkNoError(t, ierr, "insert error: %s")
		assert.Equal(t, 1, c, "changes")
		assert.T(t, !s.Busy(), "statement not busy")
	}

	checkNoError(t, db.Commit(), "Error: %s")

	cs, _ := db.Prepare("SELECT COUNT(*) FROM test")
	defer checkFinalize(cs, t)
	assert.T(t, cs.ReadOnly(), "SELECT statement should be readonly")
	assert.T(t, checkStep(t, cs))
	var i int
	checkNoError(t, cs.Scan(&i), "error scanning count: %s")
	assert.Equal(t, 1000, i, "count")

	rs, _ := db.Prepare("SELECT float_num, int_num, a_string FROM test WHERE a_string LIKE ? ORDER BY int_num LIMIT 2", "hel%")
	defer checkFinalize(rs, t)
	columnCount = rs.ColumnCount()
	assert.Equal(t, 3, columnCount, "column count")
	assert.Equal(t, "int_num", rs.ColumnName(1), "column name")

	if checkStep(t, rs) {
		var fnum float64
		var inum int64
		var sstr string
		rs.Scan(&fnum, &inum, &sstr)
		assert.Equal(t, float64(0), fnum)
		assert.Equal(t, int64(0), inum)
		assert.Equal(t, "hello", sstr)
	}
	if checkStep(t, rs) {
		var fnum float64
		var inum int64
		var sstr string
		rs.NamedScan("a_string", &sstr, "float_num", &fnum, "int_num", &inum)
		assert.Equal(t, float64(3.14), fnum)
		assert.Equal(t, int64(1), inum)
		assert.Equal(t, "hello", sstr)
	}
	assert.T(t, 999 == rs.Status(StmtStatusFullScanStep, false), "expected full scan")
	assert.T(t, 1 == rs.Status(StmtStatusSort, false), "expected one sort")
	assert.T(t, 0 == rs.Status(StmtStatusAutoIndex, false), "expected no auto index")
}

func TestScanColumn(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)

	s, err := db.Prepare("SELECT 1, null, 0")
	checkNoError(t, err, "prepare error: %s")
	defer checkFinalize(s, t)
	assert.T(t, checkStep(t, s))
	var i1, i2, i3 int
	null, err := s.ScanByIndex(0, &i1)
	checkNoError(t, err, "%s")
	assert.T(t, !null, "expected not null value")
	assert.Equal(t, 1, i1)
	null, err = s.ScanByIndex(1, &i2)
	checkNoError(t, err, "%s")
	assert.T(t, null, "expected null value")
	assert.Equal(t, 0, i2)
	null, err = s.ScanByIndex(2, &i3)
	checkNoError(t, err, "%s")
	assert.T(t, !null, "expected not null value")
	assert.Equal(t, 0, i3)

	err = s.Scan(i1, i2, i3, nil)
	assert.T(t, err != nil)
	//println(err.Error())
}

func TestNamedScanColumn(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)

	s, err := db.Prepare("SELECT 1 AS i1, null AS i2, 0 AS i3")
	checkNoError(t, err, "prepare error: %s")
	defer checkFinalize(s, t)
	assert.T(t, checkStep(t, s))
	var i1, i2, i3 int
	null, err := s.ScanByName("i1", &i1)
	checkNoError(t, err, "%s")
	assert.T(t, !null, "expected not null value")
	assert.Equal(t, 1, i1)
	null, err = s.ScanByName("i2", &i2)
	checkNoError(t, err, "%s")
	assert.T(t, null, "expected null value")
	assert.Equal(t, 0, i2)
	null, err = s.ScanByName("i3", &i3)
	checkNoError(t, err, "%s")
	assert.T(t, !null, "expected not null value")
	assert.Equal(t, 0, i3)

	_, err = s.ScanByName("invalid", &i1)
	assert.T(t, err != nil, "expected invalid name")
	//println(err.Error())

	err = s.NamedScan("invalid", &i1)
	assert.T(t, err != nil, "expected invalid name")
	//println(err.Error())

	err = s.NamedScan("i1", i1)
	assert.T(t, err != nil, "expected invalid type")
	//println(err.Error())
}

func TestScanCheck(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)

	s, err := db.Prepare("SELECT 'hello'")
	checkNoError(t, err, "prepare error: %s")
	defer checkFinalize(s, t)
	assert.T(t, checkStep(t, s))
	var i int
	_, err = s.ScanByIndex(0, &i)
	if serr, ok := err.(StmtError); ok {
		assert.Equal(t, "", serr.Filename())
		assert.Equal(t, ErrSpecific, serr.Code())
		assert.Equal(t, s.SQL(), serr.SQL())
	} else {
		t.Errorf("got %s; want StmtError", reflect.TypeOf(err))
	}
}

func TestScanNull(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)

	s, err := db.Prepare("SELECT null")
	checkNoError(t, err, "prepare error: %s")
	defer checkFinalize(s, t)
	assert.T(t, checkStep(t, s))
	var pi = new(int)
	null, err := s.ScanByIndex(0, &pi)
	checkNoError(t, err, "%s")
	assert.T(t, null, "expected null value")
	assert.Equal(t, (*int)(nil), pi, "expected nil")
	var ps = new(string)
	null, err = s.ScanByIndex(0, &ps)
	checkNoError(t, err, "%s")
	assert.T(t, null, "expected null value")
	assert.Equal(t, (*string)(nil), ps, "expected nil")

	i, null, err := s.ScanInt64(0)
	checkNoError(t, err, "scan error: %s")
	assert.T(t, null, "expected null value")
	assert.Equal(t, (int64)(0), i, "expected zero")

	var i32 int32
	null, err = s.ScanByIndex(0, &i32)
	checkNoError(t, err, "scan error: %s")
	assert.T(t, null, "expected null value")
	assert.Equal(t, (int32)(0), i32, "expected zero")

	f, null, err := s.ScanDouble(0)
	checkNoError(t, err, "scan error: %s")
	assert.T(t, null, "expected null value")
	assert.Equal(t, (float64)(0), f, "expected zero")

	b, null, err := s.ScanByte(0)
	checkNoError(t, err, "scan error: %s")
	assert.T(t, null, "expected null value")
	assert.Equal(t, (byte)(0), b, "expected zero")

	bo, null, err := s.ScanBool(0)
	checkNoError(t, err, "scan error: %s")
	assert.T(t, null, "expected null value")
	assert.Equal(t, false, bo, "expected false")

	rb, null := s.ScanRawBytes(0)
	assert.T(t, null, "expected null value")
	assert.Equal(t, 0, len(rb), "expected empty")
}

func TestScanNotNull(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)

	s, err := db.Prepare("SELECT 1")
	checkNoError(t, err, "prepare error: %s")
	defer checkFinalize(s, t)
	assert.T(t, checkStep(t, s))
	var pi = new(int)
	null, err := s.ScanByIndex(0, &pi)
	checkNoError(t, err, "%s")
	assert.T(t, !null, "expected not null value")
	assert.Equal(t, 1, *pi)
	var ps = new(string)
	null, err = s.ScanByIndex(0, &ps)
	checkNoError(t, err, "%s")
	assert.T(t, !null, "expected not null value")
	assert.Equal(t, "1", *ps)

	rb, null := s.ScanRawBytes(0)
	assert.T(t, !null, "expected not null value")
	assert.Equal(t, 1, len(rb), "expected not empty")

	var i32 int32
	null, err = s.ScanByIndex(0, &i32)
	checkNoError(t, err, "scan error: %s")
	assert.T(t, !null, "expected not null value")
	assert.Equal(t, (int32)(1), i32)
}

/*
func TestScanError(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)

	s, err := db.Prepare("SELECT 1")
	checkNoError(t, err, "prepare error: %s")
	defer checkFinalize(s, t)
	assert.T(t, checkStep(t, s))
	var pi *int
	null, err := s.ScanByIndex(0, &pi)
	t.Errorf("(%t,%s)", null, err)
}*/

func TestCloseTwice(t *testing.T) {
	db := open(t)
	s, err := db.Prepare("SELECT 1")
	checkNoError(t, err, "prepare error: %s")
	err = s.Finalize()
	checkNoError(t, err, "finalize error: %s")
	err = s.Finalize()
	checkNoError(t, err, "finalize error: %s")
	err = db.Close()
	checkNoError(t, err, "close error: %s")
	err = db.Close()
	checkNoError(t, err, "close error: %s")
}

func TestStmtMisuse(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)
	createTable(db, t)

	s, err := db.Prepare("MISUSE")
	assert.T(t, s == nil && err != nil, "error expected")
	//println(err.Error())
	err = s.Finalize()
	assert.T(t, err != nil, "error expected")

	_, err = db.Prepare("INSERT INTO test VALUES (?, ?, ?, ?)", os.ErrInvalid, nil, nil, nil)
	assert.T(t, err != nil)
	//println(err.Error())
}

func TestStmtWithClosedDb(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)

	db.SetCacheSize(0)

	s, err := db.Prepare("SELECT 1")
	checkNoError(t, err, "prepare error: %s")
	assert.Equal(t, db, s.Conn(), "conn")

	err = db.Close()
	checkNoError(t, err, "close error: %s")

	err = s.Finalize()
	assert.T(t, err != nil, "error expected")
	//println(err.Error())
}

func TestStmtExecWithSelect(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)

	s, err := db.Prepare("SELECT 1")
	checkNoError(t, err, "prepare error: %s")
	defer s.Finalize()

	err = s.Exec()
	assert.T(t, err != nil, "error expected")
	//println(err.Error())
	if serr, ok := err.(StmtError); ok {
		assert.Equal(t, ErrSpecific, serr.Code())
	} else {
		t.Errorf("got %s; want StmtError", reflect.TypeOf(err))
	}

	s1, err := db.Prepare("SELECT 1 LIMIT 0")
	checkNoError(t, err, "prepare error: %s")
	defer s1.Finalize()

	err = s1.Exec()
	assert.T(t, err != nil, "error expected")
	//println(err.Error())
}

func TestSelectOneRow(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)

	s, err := db.Prepare("SELECT 1")
	checkNoError(t, err, "prepare error: %s")
	defer s.Finalize()

	exists, err := s.SelectOneRow(nil)
	checkNoError(t, err, "select error: %s")
	assert.T(t, exists)

	exists, err = s.SelectOneRow(nil)
	checkNoError(t, err, "select error: %s")
	assert.T(t, !exists)
}

func TestStmtSelectWithInsert(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)
	err := db.Exec("CREATE TABLE test (data TEXT)")
	checkNoError(t, err, "exec error: %s")

	s, err := db.Prepare("INSERT INTO test VALUES ('...')")
	checkNoError(t, err, "prepare error: %s")
	defer s.Finalize()

	exists, err := s.SelectOneRow()
	assert.T(t, err != nil, "error expected")
	//println(err.Error())
	if serr, ok := err.(StmtError); ok {
		assert.Equal(t, ErrSpecific, serr.Code())
	} else {
		t.Errorf("got %s; want StmtError", reflect.TypeOf(err))
	}
	assert.T(t, !exists, "false expected")

	err = s.Select(nil)
	assert.T(t, err != nil)
	//println(err.Error())
}

func TestNamedBind(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)
	err := db.Exec("CREATE TABLE test (data BLOB, byte INT)")
	checkNoError(t, err, "exec error: %s")

	is, err := db.Prepare("INSERT INTO test (data, byte) VALUES (:blob, :b)")
	checkNoError(t, err, "prepare error: %s")
	bc := is.BindParameterCount()
	assert.Equal(t, 2, bc, "parameter count")
	for i := 1; i <= bc; i++ {
		_, err := is.BindParameterName(i)
		checkNoError(t, err, "bind parameter name error: %s")
	}

	blob := []byte{'h', 'e', 'l', 'l', 'o'}
	var byt byte = '!'
	err = is.NamedBind(":b", byt, ":blob", blob)
	checkNoError(t, err, "named bind error: %s")
	checkStep(t, is)

	err = is.NamedBind(":b", byt, ":invalid", nil)
	assert.T(t, err != nil, "invalid param name expected")
	err = is.NamedBind(":b")
	assert.T(t, err != nil, "missing params")
	err = is.NamedBind(byt, ":b")
	assert.T(t, err != nil, "invalid param name")
	err = is.NamedBind(":b", byt, ":blob", os.ErrInvalid)
	assert.T(t, err != nil, "invalid param type")
	checkFinalize(is, t)

	s, err := db.Prepare("SELECT data AS bs, byte AS b FROM test")
	checkNoError(t, err, "prepare error: %s")
	defer checkFinalize(s, t)
	assert.T(t, checkStep(t, s))
	var bs []byte
	var b byte
	err = s.NamedScan("b", &b, "bs", &bs)
	checkNoError(t, err, "named scan error: %s")
	assert.Equal(t, len(blob), len(bs), "blob length")
	assert.Equal(t, byt, b, "byte")

	err = s.NamedScan("b")
	assert.T(t, err != nil, "missing params")
	err = s.NamedScan(&b, "b")
	assert.T(t, err != nil, "invalid param name")
}

func TestBind(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)
	err := db.Exec("CREATE TABLE test (data TEXT, bool INT)")
	checkNoError(t, err, "exec error: %s")

	is, err := db.Prepare("INSERT INTO test (data, bool) VALUES (?, ?)")
	defer checkFinalize(is, t)
	checkNoError(t, err, "prepare error: %s")
	err = is.Bind(nil, true)
	checkNoError(t, err, "bind error: %s")
	checkStep(t, is)

	err = is.Bind(int32(1), float32(273.1))
	checkNoError(t, err, "bind error: %s")

	err = is.Bind(nil, os.ErrInvalid)
	assert.T(t, err != nil, "unsupported type error expected")
}

func TestInsertMisuse(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)
	err := db.Exec("CREATE TABLE test (data TEXT, bool INT)")
	checkNoError(t, err, "exec error: %s")

	is, err := db.Prepare("INSERT INTO test (data, bool) VALUES (?, ?)")
	checkNoError(t, err, "prepare error: %s")
	defer checkFinalize(is, t)

	_, err = is.Insert()
	assert.T(t, err != nil, "missing bind parameters expected")

	//db.Exec("DELETE FROM test") // to reset sqlite3_changes counter

	ois, err := db.Prepare("PRAGMA shrink_memory")
	checkNoError(t, err, "prepare error: %s")
	defer checkFinalize(ois, t)
	rowID, err := ois.Insert()
	checkNoError(t, err, "insert error: %s")
	assert.Equal(t, int64(-1), rowID)
}

func TestScanValues(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)

	s, err := db.Prepare("SELECT 1, null, 0")
	checkNoError(t, err, "prepare error: %s")
	defer checkFinalize(s, t)
	assert.T(t, checkStep(t, s))
	values := make([]interface{}, 3)
	s.ScanValues(values)
	assert.Equal(t, int64(1), values[0])
	assert.Equal(t, nil, values[1])
	assert.Equal(t, int64(0), values[2])
}

func TestScanBytes(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)

	s, err := db.Prepare("SELECT 'test'")
	checkNoError(t, err, "prepare error: %s")
	defer checkFinalize(s, t)
	assert.T(t, checkStep(t, s))
	blob, _ := s.ScanBlob(0)
	assert.Equal(t, "test", string(blob))
}

func TestBindEmptyZero(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)

	var zero time.Time
	s, err := db.Prepare("SELECT ?, ?", "", zero)
	checkNoError(t, err, "prepare error: %s")
	defer checkFinalize(s, t)
	assert.T(t, checkStep(t, s))

	var ps *string
	var zt time.Time
	err = s.Scan(&ps, &zt)
	checkNoError(t, err, "scan error: %s")
	assert.T(t, ps == nil && zt.IsZero(), "null pointers expected")
	_, null := s.ScanValue(0)
	assert.T(t, null, "null string expected")
	_, null = s.ScanValue(1)
	assert.T(t, null, "null time expected")
}

func TestBindEmptyZeroNotTransformedToNull(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)

	NullIfEmptyString = false
	NullIfZeroTime = false
	defer func() {
		NullIfEmptyString = true
		NullIfZeroTime = true
	}()

	var zero time.Time
	s, err := db.Prepare("SELECT ?, ?", "", zero)
	checkNoError(t, err, "prepare error: %s")
	defer checkFinalize(s, t)
	assert.T(t, checkStep(t, s))

	var st string
	var zt time.Time
	err = s.Scan(&st, &zt)
	checkNoError(t, err, "scan error: %s")
	assert.T(t, len(st) == 0 && zt.IsZero(), "null pointers expected")
	_, null := s.ScanValue(0)
	assert.T(t, !null, "empty string expected")
	_, null = s.ScanValue(1)
	assert.T(t, !null, "zero time expected")
}

func TestColumnType(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)

	createTable(db, t)
	s, err := db.Prepare("SELECT * from test")
	checkNoError(t, err, "prepare error: %s")
	defer checkFinalize(s, t)

	expectedAffinities := []Affinity{Integral, Real, Integral, Textual}
	for col := 0; col < s.ColumnCount(); col++ {
		//println(col, s.ColumnName(col), s.ColumnOriginName(col), s.ColumnType(col), s.ColumnDeclaredType(col))
		assert.Equal(t, Null, s.ColumnType(col), "column type")
		assert.Equal(t, expectedAffinities[col], s.ColumnTypeAffinity(col), "column type affinity")
	}
}

func TestIntOnArch64(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)

	createTable(db, t)
	if unsafe.Sizeof(int(0)) > 4 {
		var i = math.MaxInt64
		err := db.Exec("INSERT INTO test (int_num) VALUES (?)", i)
		checkNoError(t, err, "insert error: %s")
		var r int
		err = db.OneValue("SELECT int_num FROM test", &r)
		checkNoError(t, err, "select error: %s")
		assert.Equal(t, i, r, "int truncated")
	}
}

func TestBlankQuery(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)

	s, err := db.Prepare("")
	checkNoError(t, err, "prepare error: %s")
	defer checkFinalize(s, t)
	assert.T(t, s.Empty(), "empty stmt expected")
	assert.Equal(t, "", s.Tail(), "empty tail expected")

	_, err = s.SelectOneRow()
	assert.T(t, err != nil, "error expected")
	//println(err.Error())
}

func TestNilStmt(t *testing.T) {
	var s *Stmt
	err := s.Finalize()
	assert.T(t, err != nil, "error expected")
}

func TestBindAndScanReflect(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)

	s, err := db.Prepare("SELECT 1")
	checkNoError(t, err, "prepare error: %s")
	defer checkFinalize(s, t)
	assert.T(t, checkStep(t, s))

	is, err := db.Prepare("SELECT ?")
	checkNoError(t, err, "prepare error: %s")
	defer checkFinalize(is, t)

	type Flag bool
	var bo Flag
	null, err := s.ScanReflect(0, bo)
	assert.T(t, err != nil, "scan error expected")
	null, err = s.ScanReflect(0, &bo)
	checkNoError(t, err, "scan error: %s")
	assert.T(t, !null)
	assert.Equal(t, Flag(true), bo)

	checkNoError(t, is.BindReflect(1, bo), "bind error: %s")

	type Type string
	var typ Type
	null, err = s.ScanReflect(0, &typ)
	checkNoError(t, err, "scan error: %s")
	assert.Equal(t, Type("1"), typ)

	checkNoError(t, is.BindReflect(1, typ), "bind error: %s")

	type Code int
	var code Code
	null, err = s.ScanReflect(0, &code)
	checkNoError(t, err, "scan error: %s")
	assert.Equal(t, Code(1), code)

	checkNoError(t, is.BindReflect(1, code), "bind error: %s")

	type Enum uint
	var enum Enum
	_, err = s.ScanReflect(0, &enum)
	checkNoError(t, err, "scan error: %s")
	assert.Equal(t, Enum(1), enum)

	checkNoError(t, is.BindReflect(1, enum), "bind error: %s")

	type Amount float64
	var amount Amount
	_, err = s.ScanReflect(0, &amount)
	checkNoError(t, err, "scan error: %s")
	assert.Equal(t, Amount(1), amount)

	checkNoError(t, is.BindReflect(1, amount), "bind error: %s")

	checkNoError(t, is.BindReflect(1, -1), "bind error: %s")
	assert.T(t, checkStep(t, is))

	_, err = is.ScanReflect(0, &enum)
	assert.T(t, err != nil)
	//println(err.Error())

	var ut error
	_, err = is.ScanReflect(0, &ut)
	assert.T(t, err != nil)
	//println(err.Error())

	var ui uint64 = math.MaxUint64
	err = is.BindReflect(1, ui)
	assert.T(t, err != nil)
	//println(err.Error())
}

func TestSelect(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)

	s, err := db.Prepare("SELECT 1 LIMIT ?")
	checkNoError(t, err, "prepare error: %s")
	defer checkFinalize(s, t)

	err = s.Select(func(s *Stmt) error {
		return s.Scan(nil)
	}, 1)
	checkNoError(t, err, "select error: %s")

	err = s.Select(func(s *Stmt) error {
		return os.ErrInvalid
	}, os.ErrInvalid)
	assert.T(t, err != nil)

	err = s.Select(func(s *Stmt) error {
		return os.ErrInvalid
	}, 1)
	assert.Equal(t, os.ErrInvalid, err)
}

func TestStmtCache(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)

	s, err := db.Prepare("SELECT 1 LIMIT ?")
	checkNoError(t, err, "prepare error: %s")
	s.Finalize()

	s, err = db.Prepare("SELECT 1 LIMIT ?", 0)
	checkNoError(t, err, "prepare error: %s")
	s.Finalize()

	_, err = db.Prepare("SELECT 1 LIMIT ?", os.ErrInvalid)
	assert.T(t, err != nil)
	//println(err.Error())
}

func TestCheckTypeMismatch(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)

	var i int
	err := db.OneValue("SELECT 3.14", &i)
	assert.T(t, err != nil)
	//println(err.Error())

	var f float64
	err = db.OneValue("SELECT 'bim'", &f)
	assert.T(t, err != nil)
	//println(err.Error())

	err = db.OneValue("SELECT X'53514C697465'", &f)
	assert.T(t, err != nil)
	//println(err.Error())
}

func TestReadOnly(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)

	s, err := db.Prepare("DROP TABLE IF EXISTS test")
	checkNoError(t, err, "prepare error: %s")
	assert.T(t, s.ReadOnly())
	//checkNoError(t, s.Exec(), "exe error: %s")
	checkFinalize(s, t)

	s, err = db.Prepare("CREATE TABLE test (data TEXT)")
	checkNoError(t, err, "prepare error: %s")
	assert.T(t, !s.ReadOnly())
	checkNoError(t, s.Exec(), "exe error: %s")
	checkFinalize(s, t)

	s, err = db.Prepare("DROP TABLE IF EXISTS test")
	//s, err = db.Prepare("DROP TABLE test")
	checkNoError(t, err, "prepare error: %s")
	assert.T(t, s.ReadOnly()) // FIXME
	checkNoError(t, s.Exec(), "exe error: %s")
	checkFinalize(s, t)
	assert.T(t, !s.ReadOnly())
}
