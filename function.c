// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
#include <sqlite3.h>
#include <stdlib.h>

// These wrappers are necessary because SQLITE_TRANSIENT
// is a pointer constant, and cgo doesn't translate them correctly.

void my_result_text(sqlite3_context *ctx, char *p, int np) {
	sqlite3_result_text(ctx, p, np, SQLITE_TRANSIENT);
}
void my_result_blob(sqlite3_context *ctx, void *p, int np) {
	sqlite3_result_blob(ctx, p, np, SQLITE_TRANSIENT);
}
