// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sqlite_test

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/bmizerany/assert"
	"github.com/gwenn/gosqlite"
)

const (
	ddl = "DROP TABLE IF EXISTS test;" +
		"CREATE TABLE test (id INTEGER PRIMARY KEY NOT NULL," +
		" name TEXT);"
	dml = "INSERT INTO test (name) VALUES ('Bart');" +
		"INSERT INTO test (name) VALUES ('Lisa');" +
		"UPDATE test SET name = 'El Barto' WHERE name = 'Bart';" +
		"DELETE FROM test WHERE name = 'Bart';"
	insert       = "INSERT INTO test (name) VALUES (?)"
	insert_named = "INSERT INTO test (name) VALUES (:name)"
	query        = "SELECT * FROM test WHERE name LIKE ?"
)

func sqlOpen(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", "file:dummy.db?mode=memory&cache=shared")
	checkNoError(t, err, "Error opening database: %s")
	return db
}

func checkSqlDbClose(db *sql.DB, t *testing.T) {
	checkNoError(t, db.Close(), "Error closing connection: %s")
}

func checkSqlStmtClose(stmt *sql.Stmt, t *testing.T) {
	checkNoError(t, stmt.Close(), "Error closing statement: %s")
}

func checkSqlRowsClose(rows *sql.Rows, t *testing.T) {
	checkNoError(t, rows.Close(), "Error closing rows: %s")
}

func sqlCreate(ddl string, t *testing.T) *sql.DB {
	db := sqlOpen(t)
	_, err := db.Exec(ddl)
	checkNoError(t, err, "Error creating table: %s")
	return db
}

func TestSqlOpen(t *testing.T) {
	db := sqlOpen(t)
	checkNoError(t, db.Close(), "Error closing database: %s")

	db, err := sql.Open("sqlite3", "file:data.db?mode=readonly")
	checkNoError(t, err, "Error opening database: %s")
	defer checkSqlDbClose(db, t)
	err = db.Ping()
	assert.T(t, err != nil)
	//println(err.Error())
}

func TestSqlDdl(t *testing.T) {
	db := sqlOpen(t)
	defer checkSqlDbClose(db, t)
	result, err := db.Exec(ddl)
	checkNoError(t, err, "Error creating table: %s")
	_, err = result.LastInsertId() // FIXME Error expected
	if err == nil {
		t.Logf("Error expected when calling LastInsertId after DDL")
	}
	_, err = result.RowsAffected() // FIXME Error expected
	if err == nil {
		t.Logf("Error expected when calling RowsAffected after DDL")
	}
}

func TestSqlDml(t *testing.T) {
	db := sqlCreate(ddl, t)
	defer checkSqlDbClose(db, t)
	result, err := db.Exec(dml)
	checkNoError(t, err, "Error updating data: %s")
	id, err := result.LastInsertId()
	checkNoError(t, err, "Error while calling LastInsertId: %s")
	assert.Equal(t, int64(2), id, "lastInsertId")
	changes, err := result.RowsAffected()
	checkNoError(t, err, "Error while calling RowsAffected: %s")
	assert.Equal(t, int64(0), changes, "rowsAffected")
}

func TestSqlInsert(t *testing.T) {
	db := sqlCreate(ddl, t)
	defer checkSqlDbClose(db, t)
	result, err := db.Exec(insert, "Bart")
	checkNoError(t, err, "Error updating data: %s")
	id, err := result.LastInsertId()
	checkNoError(t, err, "Error while calling LastInsertId: %s")
	assert.Equal(t, int64(1), id, "lastInsertId")
	changes, err := result.RowsAffected()
	checkNoError(t, err, "Error while calling RowsAffected: %s")
	assert.Equal(t, int64(1), changes, "rowsAffected")
}

func TestSqlInsertNamed(t *testing.T) {
	db := sqlCreate(ddl, t)
	defer checkSqlDbClose(db, t)
	result, err := db.Exec(insert_named, sql.Named("name", "Bart"))
	checkNoError(t, err, "Error updating data: %s")
	id, err := result.LastInsertId()
	checkNoError(t, err, "Error while calling LastInsertId: %s")
	assert.Equal(t, int64(1), id, "lastInsertId")
	changes, err := result.RowsAffected()
	checkNoError(t, err, "Error while calling RowsAffected: %s")
	assert.Equal(t, int64(1), changes, "rowsAffected")
}

func TestSqlExecWithIllegalCmd(t *testing.T) {
	db := sqlCreate(ddl+dml, t)
	defer checkSqlDbClose(db, t)

	_, err := db.Exec(query, "%")
	if err == nil {
		t.Fatalf("Error expected when calling Exec with a SELECT")
	}
}

func TestSqlQuery(t *testing.T) {
	db := sqlCreate(ddl+dml, t)
	defer checkSqlDbClose(db, t)

	rows, err := db.Query(query, "%")
	defer checkSqlRowsClose(rows, t)
	var id int
	var name string
	for rows.Next() {
		err = rows.Scan(&id, &name)
		checkNoError(t, err, "Error while scanning: %s")
	}
}

func TestSqlTx(t *testing.T) {
	db := sqlCreate(ddl, t)
	defer checkSqlDbClose(db, t)

	tx, err := db.Begin()
	checkNoError(t, err, "Error while beginning tx: %s")
	err = tx.Rollback()
	checkNoError(t, err, "Error while rollbacking tx: %s")

	tx, err = db.Begin()
	checkNoError(t, err, "Error while beginning tx: %s")
	err = tx.Commit()
	checkNoError(t, err, "Error while committing tx: %s")
}

func TestSqlPrepare(t *testing.T) {
	db := sqlCreate(ddl+dml, t)
	defer checkSqlDbClose(db, t)

	stmt, err := db.Prepare(insert)
	checkNoError(t, err, "Error while preparing stmt: %s")
	defer checkSqlStmtClose(stmt, t)
	result, err := stmt.Exec("Bart")
	checkNoError(t, err, "Error while executing stmt: %s")
	id, err := result.LastInsertId()
	checkNoError(t, err, "Error while calling LastInsertId: %s")
	assert.Equal(t, int64(3), id, "lastInsertId")
	changes, err := result.RowsAffected()
	checkNoError(t, err, "Error while calling RowsAffected: %s")
	assert.Equal(t, int64(1), changes, "rowsAffected")
}

func TestSqlPrepareNamed(t *testing.T) {
	db := sqlCreate(ddl+dml, t)
	defer checkSqlDbClose(db, t)

	stmt, err := db.Prepare(insert_named)
	checkNoError(t, err, "Error while preparing stmt: %s")
	defer checkSqlStmtClose(stmt, t)
	result, err := stmt.Exec(sql.Named("name", "Bart"))
	checkNoError(t, err, "Error while executing stmt: %s")
	id, err := result.LastInsertId()
	checkNoError(t, err, "Error while calling LastInsertId: %s")
	assert.Equal(t, int64(3), id, "lastInsertId")
	changes, err := result.RowsAffected()
	checkNoError(t, err, "Error while calling RowsAffected: %s")
	assert.Equal(t, int64(1), changes, "rowsAffected")
}

func TestRowsWithStmtClosed(t *testing.T) {
	db := sqlCreate(ddl+dml, t)
	defer checkSqlDbClose(db, t)

	stmt, err := db.Prepare(query)
	checkNoError(t, err, "Error while preparing stmt: %s")
	//defer stmt.Close()

	rows, err := stmt.Query("%")
	checkSqlStmtClose(stmt, t)
	defer checkSqlRowsClose(rows, t)
	var id int
	var name string
	for rows.Next() {
		err = rows.Scan(&id, &name)
		checkNoError(t, err, "Error while scanning: %s")
	}
}

func TestUnwrap(t *testing.T) {
	db := sqlOpen(t)
	defer checkSqlDbClose(db, t)
	conn := sqlite.Unwrap(db)
	assert.Tf(t, conn != nil, "got %#v; want *sqlite.Conn", conn)
	// fmt.Printf("%#v\n", conn)
	conn.TotalChanges()
}

func TestCustomRegister(t *testing.T) {
	sql.Register("sqlite3ReadOnly", sqlite.NewDriver(func(name string) (*sqlite.Conn, error) {
		c, err := sqlite.Open(name, sqlite.OpenURI, sqlite.OpenNoMutex, sqlite.OpenReadOnly)
		if err != nil {
			return nil, err
		}
		c.BusyTimeout(10 * time.Second)
		return c, nil
	}, nil))
	// readlonly memory db is useless but...
	db, err := sql.Open("sqlite3ReadOnly", ":memory:")
	checkNoError(t, err, "Error while opening customized db: %s")
	defer checkSqlDbClose(db, t)
	conn := sqlite.Unwrap(db)
	ro, err := conn.Readonly("main")
	checkNoError(t, err, "Error while reading readonly status: %s")
	assert.Tf(t, ro, "readonly = %t; want %t", ro, true)
}

func TestCustomRegister2(t *testing.T) {
	sql.Register("sqlite3FK", sqlite.NewDriver(nil, func(c *sqlite.Conn) error {
		_, err := c.EnableFKey(true)
		return err
	}))
	db, err := sql.Open("sqlite3FK", ":memory:")
	checkNoError(t, err, "Error while opening customized db: %s")
	defer checkSqlDbClose(db, t)
	conn := sqlite.Unwrap(db)
	fk, err := conn.IsFKeyEnabled()
	checkNoError(t, err, "Error while reading foreign_keys status: %s")
	assert.Tf(t, fk, "foreign_keys = %t; want %t", fk, true)
}

// sql: Scan error on column index 0: unsupported driver -> Scan pair: []uint8 -> *time.Time
func TestScanTimeFromView(t *testing.T) {
	db := sqlCreate("CREATE VIEW v AS SELECT strftime('%Y-%m-%d %H:%M:%f', 'now') AS tic", t)
	defer checkSqlDbClose(db, t)

	conn := sqlite.Unwrap(db)
	conn.DefaultTimeLayout = "2006-01-02 15:04:05.000"

	row := db.QueryRow("SELECT * FROM v")
	var tic time.Time
	err := row.Scan(&tic)
	//checkNoError(t, err, "Error while scanning view: %s")
	assert.Tf(t, err != nil, "scan error expected")
}

func TestScanNumericalAsTime(t *testing.T) {
	db := sqlOpen(t)
	defer checkSqlDbClose(db, t)
	now := time.Now()
	_, err := db.Exec("CREATE TABLE test (ms TIMESTAMP); INSERT INTO test VALUES (?)", now)
	checkNoError(t, err, "%s")
	row := db.QueryRow("SELECT ms FROM test")

	now = now.Truncate(time.Millisecond)

	var ms time.Time
	err = row.Scan(&ms)
	checkNoError(t, err, "%s")
	if !now.Equal(ms) {
		t.Errorf("got timeStamp: %s; want %s", ms, now)
	}

	_, err = db.Exec("DELETE FROM test; INSERT INTO test VALUES (?)", "bim")
	checkNoError(t, err, "%s")
	row = db.QueryRow("SELECT ms FROM test")
	err = row.Scan(&ms)
	assert.T(t, err != nil)
	//println(err.Error())
}

// Adapted from https://github.com/bradfitz/go-sql-test/blob/master/src/sqltest/sql_test.go
func TestBlobs(t *testing.T) {
	db := sqlOpen(t)
	defer checkSqlDbClose(db, t)

	var blob = []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
	result, err := db.Exec("CREATE TABLE foo (id INTEGER PRIMARY KEY, bar BLOB); INSERT INTO foo (id, bar) VALUES (?, ?)", 0, blob)
	checkNoError(t, err, "Error inserting BLOB: %s")
	id, err := result.LastInsertId()
	checkNoError(t, err, "Error while calling LastInsertId: %s")
	assert.Equal(t, int64(0), id, "lastInsertId")
	changes, err := result.RowsAffected()
	checkNoError(t, err, "Error while calling RowsAffected: %s")
	assert.Equal(t, int64(1), changes, "rowsAffected")

	var b []byte
	err = db.QueryRow("SELECT bar FROM foo WHERE id = ?", 0).Scan(&b)
	checkNoError(t, err, "Error selecting BLOB: %s")
	assert.Equalf(t, blob, b, "blob = %v; want %v", b, blob)
	var s string
	err = db.QueryRow("SELECT bar FROM foo WHERE id = ?", 0).Scan(&s)
	checkNoError(t, err, "Error selecting BLOB: %s")
	assert.Equalf(t, string(blob), s, "blob = %s; want %s", s, string(blob))
}

func TestManyQueryRow(t *testing.T) {
	db := sqlOpen(t)
	defer checkSqlDbClose(db, t)

	_, err := db.Exec("CREATE TABLE foo (id INTEGER PRIMARY KEY, name TEXT); INSERT INTO foo (id, name) VALUES (?, ?)", 1, "bob")
	checkNoError(t, err, "Error inserting BLOB: %s")
	var name string
	for i := 0; i < 10000; i++ {
		err := db.QueryRow("SELECT name FROM foo where id = ?", 1).Scan(&name)
		if err != nil || name != "bob" {
			t.Fatalf("on query %d: err=%v, name=%q", i, err, name)
		}
	}
}

// Adapted from https://github.com/bradfitz/go-sql-test/blob/master/src/sqltest/sql_test.go
func TestTxQuery(t *testing.T) {
	db := sqlOpen(t)
	defer checkSqlDbClose(db, t)

	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()

	_, err = db.Exec("CREATE TABLE foo (id INTEGER PRIMARY KEY, name TEXT)")
	if err != nil {
		t.Logf("cannot create table foo: %s", err)
	}

	_, err = tx.Exec("INSERT INTO foo (id, name) VALUES (?,?)", 1, "bob")
	if err != nil {
		t.Fatal(err)
	}

	r, err := tx.Query("SELECT name FROM foo WHERE id = ?", 1)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	if !r.Next() {
		if r.Err() != nil {
			t.Fatal(err)
		}
		t.Fatal("expected one rows")
	}

	var name string
	err = r.Scan(&name)
	if err != nil {
		t.Fatal(err)
	}
}

// Adapted from https://github.com/bradfitz/go-sql-test/blob/master/src/sqltest/sql_test.go
func _TestPreparedStmt(t *testing.T) {
	db := sqlOpen(t)
	defer checkSqlDbClose(db, t)

	_, err := db.Exec("CREATE TABLE t (count INT)")
	checkNoError(t, err, "Error creating table: %s")
	sel, err := db.Prepare("SELECT count FROM t ORDER BY count DESC")
	if err != nil {
		t.Fatalf("prepare 1: %v", err)
	}
	ins, err := db.Prepare("INSERT INTO t (count) VALUES (?)")
	if err != nil {
		t.Fatalf("prepare 2: %v", err)
	}

	for n := 1; n <= 3; n++ {
		if _, err := ins.Exec(n); err != nil {
			t.Fatalf("insert(%d) = %v", n, err)
		}
	}

	const nRuns = 10
	ch := make(chan bool)
	for i := 0; i < nRuns; i++ {
		go func() {
			defer func() {
				ch <- true
			}()
			for j := 0; j < 10; j++ {
				count := 0
				if err := sel.QueryRow().Scan(&count); err != nil && err != sql.ErrNoRows {
					t.Errorf("Query: %v", err)
					return
				}
				if _, err := ins.Exec(rand.Intn(100)); err != nil {
					t.Errorf("Insert: %v", err)
					return
				}
			}
		}()
	}
	for i := 0; i < nRuns; i++ {
		<-ch
	}
}

func TestCancel(t *testing.T) {
	skipIfCgoCheckActive(t)

	db := sqlCreate(ddl, t)
	defer checkSqlDbClose(db, t)

	for i := 0; i < 100; i++ {
		_, err := db.Exec(insert, "Bart "+strconv.Itoa(i))
		checkNoError(t, err, "Error updating data: %s")
	}

	conn := sqlite.Unwrap(db)
	assert.Tf(t, conn != nil, "got %#v; want *sqlite.Conn", conn)
	conn.CreateScalarFunction("sleep", 0, false, nil, func(ctx *sqlite.ScalarContext, nArg int) {
		time.Sleep(5 * time.Millisecond)
		ctx.ResultText("ok")
	}, nil)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()
	_, err := db.ExecContext(ctx, "UPDATE test SET name = sleep()")
	if err != context.Canceled {
		t.Errorf("ExecContext expected to fail with Cancelled but it returned %v", err)
	}

	// connection should be usable after timeout
	row := db.QueryRow("SELECT 1")
	var val int64
	err = row.Scan(&val)
	if err != nil {
		t.Fatal("Scan failed with", err)
	}
}

func TestNilAndEmptyBytes(t *testing.T) {
	db := sqlOpen(t)
	defer checkSqlDbClose(db, t)
	actualNil := []byte("use this to use an actual nil not a reference to nil")
	emptyBytes := []byte{}
	for tsti, tst := range []struct {
		name          string
		columnType    string
		insertBytes   []byte
		expectedBytes []byte
	}{
		{"actual nil blob", "blob", actualNil, nil},
		// In Go, []byte cannot be nil (only empty).
		// And, the return value from sqlite3_column_blob() for a zero-length BLOB is a NULL pointer.
		{"referenced nil blob", "blob", nil, emptyBytes},
		{"empty blob", "blob", emptyBytes, emptyBytes},
		//{"actual nil text", "text", actualNil, nil},
		//{"referenced nil text", "text", nil, nil},
		//{"empty text", "text", emptyBytes, emptyBytes},
	} {
		if _, err := db.Exec(fmt.Sprintf("create table tbl%d (txt %s)", tsti, tst.columnType)); err != nil {
			t.Fatal(tst.name, err)
		}
		if bytes.Equal(tst.insertBytes, actualNil) {
			if _, err := db.Exec(fmt.Sprintf("insert into tbl%d (txt) values (?)", tsti), nil); err != nil {
				t.Fatal(tst.name, err)
			}
		} else {
			if _, err := db.Exec(fmt.Sprintf("insert into tbl%d (txt) values (?)", tsti), &tst.insertBytes); err != nil {
				t.Fatal(tst.name, err)
			}
		}
		rows, err := db.Query(fmt.Sprintf("select txt from tbl%d", tsti))
		if err != nil {
			t.Fatal(tst.name, err)
		}
		defer checkSqlRowsClose(rows, t)
		if !rows.Next() {
			t.Fatal(tst.name, "no rows")
		}
		var scanBytes []byte
		if err = rows.Scan(&scanBytes); err != nil {
			t.Fatal(tst.name, err)
		}
		if err = rows.Err(); err != nil {
			t.Fatal(tst.name, err)
		}
		if tst.expectedBytes == nil && scanBytes != nil {
			t.Errorf("%s: %#v != %#v", tst.name, scanBytes, tst.expectedBytes)
		} else if !bytes.Equal(scanBytes, tst.expectedBytes) {
			t.Errorf("%s: %#v != %#v", tst.name, scanBytes, tst.expectedBytes)
		}
		checkSqlRowsClose(rows, t)
	}
}

func TestMultipleResultSets(t *testing.T) {
	db := sqlOpen(t)
	defer checkSqlDbClose(db, t)

	rows, err := db.Query("SELECT 1; SELECT 2; SELECT 3")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	var i int
	for rows.Next() {
		if err := rows.Scan(&i); err != nil {
			t.Fatal(err)
		}
		if i != 1 {
			t.Fatalf("expected 1, got %d", i)
		}
	}
	if !rows.NextResultSet() {
		t.Fatal("expected more result sets", rows.Err())
	}
	for rows.Next() {
		if err := rows.Scan(&i); err != nil {
			t.Fatal(err)
		}
		if i != 2 {
			t.Fatalf("expected 2, got %d", i)
		}
	}

	// Make sure that if we ignore a result we can still query.

	rows, err = db.Query("select 4; select 5")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		if err := rows.Scan(&i); err != nil {
			t.Fatal(err)
		}
		if i != 4 {
			t.Fatalf("expected 4, got %d", i)
		}
	}
	if !rows.NextResultSet() {
		t.Fatal("expected more result sets", rows.Err())
	}
	for rows.Next() {
		if err := rows.Scan(&i); err != nil {
			t.Fatal(err)
		}
		if i != 5 {
			t.Fatalf("expected 5, got %d", i)
		}
	}
	if rows.NextResultSet() {
		t.Fatal("unexpected result set")
	}
}
