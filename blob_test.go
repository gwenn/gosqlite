// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sqlite_test

import (
	"io"
	"testing"

	"github.com/bmizerany/assert"
	. "github.com/gwenn/gosqlite"
)

func TestBlob(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)

	err := db.Exec("CREATE TABLE test (content BLOB);")
	checkNoError(t, err, "error creating table: %s")
	s, err := db.Prepare("INSERT INTO test VALUES (?)")
	checkNoError(t, err, "prepare error: %s")
	if s == nil {
		t.Fatal("statement is nil")
	}
	defer checkFinalize(s, t)
	err = s.Exec(ZeroBlobLength(10))
	checkNoError(t, err, "insert error: %s")
	rowid := db.LastInsertRowid()

	bw, err := db.NewBlobReadWriter("main", "test", "content", rowid)
	checkNoError(t, err, "blob open error: %s")
	defer bw.Close()
	content := []byte("Clob")
	n, err := bw.Write(content)
	checkNoError(t, err, "blob write error: %s")
	//bw.Close()

	_, err = bw.Write([]byte("5678901"))
	assert.T(t, err != nil)
	//println(err.Error())

	err = bw.Reopen(rowid)
	checkNoError(t, err, "blob reopen error: %s")
	bw.Close()

	br, err := db.NewBlobReader("main", "test", "content", rowid)
	checkNoError(t, err, "blob open error: %s")
	defer br.Close()
	size, err := br.Size()
	checkNoError(t, err, "blob size error: %s")

	content = make([]byte, size+5)
	n, err = br.Read(content[:5])
	checkNoError(t, err, "blob read error: %s")
	assert.Equal(t, 5, n, "bytes")

	n, err = br.Read(content[5:])
	checkNoError(t, err, "blob read error: %s")
	assert.Equal(t, 5, n, "bytes")
	//fmt.Printf("%#v\n", content)

	n, err = br.Read(content[10:])
	assert.T(t, n == 0 && err == io.EOF, "error expected")

	err = br.Reopen(rowid)
	checkNoError(t, err, "blob reopen error: %s")
	_, err = br.Seek(0, io.SeekStart)
	checkNoError(t, err, "blob seek error: %s")

	err = br.Reopen(-1)
	assert.T(t, err != nil)
	//println(err.Error())

	n, err = br.Read(nil)
	checkNoError(t, err, "blob read error: %s")
	assert.Equal(t, 0, n)

	_, err = br.Read(content)
	assert.T(t, err != nil)

	err = br.Close()
	checkNoError(t, err, "blob close error: %s")

	_, err = br.Size()
	assert.T(t, err != nil)

	_, err = br.Read(content)
	assert.T(t, err != nil)
	//println(err.Error())

	err = bw.Reopen(-1)
	assert.T(t, err != nil)
	assert.T(t, err.Error() != "")
	//println(err.Error())

	_, err = bw.Write(content)
	assert.T(t, err != nil)
	//println(err.Error())
}

func TestBlobMisuse(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)

	bw, err := db.NewBlobReadWriter("main", "test", "content", 0)
	assert.T(t, bw == nil && err != nil, "error expected")
	//println(err.Error())
	/*err = bw.Close()
	assert.T(t, err != nil, "error expected")*/
}

func TestZeroLengthBlob(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)

	err := db.Exec("CREATE TABLE test (content BLOB);")
	checkNoError(t, err, "error creating table: %s")

	err = db.Exec("INSERT INTO test VALUES (?)", ZeroBlobLength(0))
	checkNoError(t, err, "insert error: %s")
	rowid := db.LastInsertRowid()

	var blob []byte
	err = db.OneValue("SELECT content FROM test WHERE rowid = ?", &blob, rowid)
	checkNoError(t, err, "select error: %s")
	assert.T(t, blob == nil, "nil blob expected")
}
