// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sqlite_test

import (
	"testing"
	"time"

	"github.com/bmizerany/assert"
	. "github.com/gwenn/gosqlite"
)

func TestJulianDay(t *testing.T) {
	utc := JulianDayToUTC(2440587.5)
	if utc.Unix() != 0 {
		t.Errorf("got %d; want %d ", utc.Unix(), 0)
	}
	now := time.Now()
	r := JulianDayToLocalTime(JulianDay(now))
	if r.Unix() != now.Unix() { // FIXME Rounding problem?
		t.Errorf("got %#v; want %#v", r, now)
	}
}

func TestBindTime(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)
	var delta int
	//err := db.OneValue("SELECT CAST(strftime('%s', 'now') AS NUMERIC) - ?", &delta, time.Now())
	err := db.OneValue("SELECT datetime('now') - datetime(?)", &delta, time.Now())
	checkNoError(t, err, "Error reading date: %#v")
	if delta != 0 {
		t.Errorf("Delta between Go and SQLite timestamps: %d", delta)
	}
}

func TestScanTime(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)
	var dt time.Time
	err := db.OneValue("SELECT date('now')", &dt)
	checkNoError(t, err, "Error reading date: %#v")
	if dt.IsZero() {
		t.Error("Unexpected zero date")
	}

	var tm time.Time
	err = db.OneValue("SELECT time('now')", &tm)
	checkNoError(t, err, "Error reading date: %#v")
	if tm.IsZero() {
		t.Error("Unexpected zero time")
	}

	var dtm time.Time
	err = db.OneValue("SELECT strftime('%Y-%m-%dT%H:%M:%f', 'now')", &dtm)
	checkNoError(t, err, "Error reading date: %#v")
	if dtm.IsZero() {
		t.Error("Unexpected zero datetime")
	}

	var jd time.Time
	err = db.OneValue("SELECT CAST(strftime('%J', 'now') AS NUMERIC)", &jd)
	checkNoError(t, err, "Error reading date: %#v")
	if jd.IsZero() {
		t.Error("Unexpected zero julian day")
	}

	var unix time.Time
	err = db.OneValue("SELECT CAST(strftime('%s', 'now') AS NUMERIC)", &unix)
	checkNoError(t, err, "Error reading date: %#v")
	if unix.IsZero() {
		t.Error("Unexpected zero julian day")
	}
}

func TestScanNullTime(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)

	var unix UnixTime
	err := db.OneValue("SELECT NULL", &unix)
	checkNoError(t, err, "Error scanning null time: %#v")
	if !unix.IsZero() {
		t.Error("Expected zero time")
	}
}

func TestBindTimeAsString(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)
	err := db.Exec("CREATE TABLE test (time TEXT)")
	checkNoError(t, err, "exec error: %s")

	is, err := db.Prepare("INSERT INTO test (time) VALUES (?)")
	checkNoError(t, err, "prepare error: %s")
	defer checkFinalize(is, t)

	now := time.Now()
	//id1, err := is.Insert(YearMonthDay(now))
	//checkNoError(t, err, "error inserting YearMonthDay: %s")
	id2, err := is.Insert(TimeStamp{Time: now})
	checkNoError(t, err, "error inserting TimeStamp: %s")

	// The format used to persist has a max precision of 1ms.
	now = now.Truncate(time.Millisecond)

	var tim time.Time
	//err = db.OneValue("SELECT /*date(*/time/*)*/ FROM test where ROWID = ?", &tim, id1)
	//checkNoError(t, err, "error selecting YearMonthDay: %s")
	//assert.Equal(t, now.Year(), tim.Year(), "year")
	//assert.Equal(t, now.YearDay(), tim.YearDay(), "yearDay")

	err = db.OneValue("SELECT /*datetime(*/time/*)*/ FROM test where ROWID = ?", &tim, id2)
	checkNoError(t, err, "error selecting TimeStamp: %s")
	if !now.Equal(tim) {
		t.Errorf("got timeStamp: %s; want %s", tim, now)
	}
}

func TestBindTimeAsNumeric(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)
	err := db.Exec("CREATE TABLE test (time NUMERIC)")
	checkNoError(t, err, "exec error: %s")

	is, err := db.Prepare("INSERT INTO test (time) VALUES (?)")
	checkNoError(t, err, "prepare error: %s")

	now := time.Now()
	id1, err := is.Insert(UnixTime{Time: now})
	checkNoError(t, err, "error inserting UnixTime: %s")
	id2, err := is.Insert(JulianTime{Time: now})
	checkNoError(t, err, "error inserting JulianTime: %s")
	checkFinalize(is, t)

	// the format used to persist has a max precision of 1s.
	now = now.Truncate(time.Second)

	var tim time.Time
	err = db.OneValue("SELECT /*datetime(*/ time/*, 'unixepoch')*/ FROM test where ROWID = ?", &tim, id1)
	checkNoError(t, err, "error selecting UnixTime: %s")
	assert.Equal(t, now, tim)

	err = db.OneValue("SELECT /*julianday(*/time/*)*/ FROM test where ROWID = ?", &tim, id2)
	checkNoError(t, err, "error selecting JulianTime: %s")
	assert.Equal(t, now, tim)
}

func TestJulianTime(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)
	err := db.Exec("CREATE TABLE test (time NUMERIC)")
	checkNoError(t, err, "exec error: %s")

	is, err := db.Prepare("INSERT INTO test (time) VALUES (?)")
	checkNoError(t, err, "prepare error: %s")

	now := time.Now()
	id, err := is.Insert(JulianTime{Time: now})
	checkNoError(t, err, "error inserting JulianTime: %s")
	_, err = is.Insert(JulianTime{})
	checkNoError(t, err, "error inserting JulianTime: %s")
	checkFinalize(is, t)

	// the format used to persist has a max precision of 1s.
	now = now.Truncate(time.Second)

	var jt JulianTime
	err = db.OneValue("SELECT time FROM test where ROWID = ?", &jt, id)
	checkNoError(t, err, "error selecting JulianTime: %s")
	assert.Equal(t, now, jt.Time)

	err = db.OneValue("SELECT null", &jt)
	checkNoError(t, err, "%s")
	assert.T(t, jt.IsZero())

	err = db.OneValue("SELECT 0", &jt)
	checkNoError(t, err, "%s")

	err = db.OneValue("SELECT 'bim'", &jt)
	assert.T(t, err != nil)
	//println(err.Error())
}

func TestTimeStamp(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)
	err := db.Exec("CREATE TABLE test (time NUMERIC)")
	checkNoError(t, err, "exec error: %s")

	is, err := db.Prepare("INSERT INTO test (time) VALUES (?)")
	checkNoError(t, err, "prepare error: %s")

	now := time.Now()
	id, err := is.Insert(TimeStamp{Time: now})
	checkNoError(t, err, "error inserting TimeStamp: %s")
	_, err = is.Insert(TimeStamp{})
	checkNoError(t, err, "error inserting TimeStamp: %s")
	checkFinalize(is, t)

	// the format used to persist has a max precision of 1ms.
	now = now.Truncate(time.Millisecond)

	var ts TimeStamp
	err = db.OneValue("SELECT time FROM test where ROWID = ?", &ts, id)
	checkNoError(t, err, "error selecting TimeStamp: %s")
	if !now.Equal(ts.Time) {
		t.Errorf("got timeStamp: %s; want %s", ts, now)
	}

	err = db.OneValue("SELECT null", &ts)
	checkNoError(t, err, "%s")
	assert.T(t, ts.IsZero())

	err = db.OneValue("SELECT 'bim'", &ts)
	assert.T(t, err != nil)
	//println(err.Error())

	err = db.OneValue("SELECT 0", &ts)
	assert.T(t, err != nil)
	//println(err.Error())
}

func TestUnixTime(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)
	err := db.Exec("CREATE TABLE test (time NUMERIC)")
	checkNoError(t, err, "exec error: %s")

	is, err := db.Prepare("INSERT INTO test (time) VALUES (?)")
	checkNoError(t, err, "prepare error: %s")

	now := time.Now()
	id, err := is.Insert(UnixTime{Time: now})
	checkNoError(t, err, "error inserting UnixTime: %s")
	_, err = is.Insert(UnixTime{})
	checkNoError(t, err, "error inserting UnixTime: %s")
	checkFinalize(is, t)

	// the format used to persist has a max precision of 1s.
	now = now.Truncate(time.Second)

	var ut UnixTime
	err = db.OneValue("SELECT time FROM test where ROWID = ?", &ut, id)
	checkNoError(t, err, "error selecting UnixTime: %s")
	assert.Equal(t, now, ut.Time)

	err = db.OneValue("SELECT null", &ut)
	checkNoError(t, err, "%s")
	assert.T(t, ut.IsZero())

	err = db.OneValue("SELECT 'bim'", &ut)
	assert.T(t, err != nil)
	//println(err.Error())
}
