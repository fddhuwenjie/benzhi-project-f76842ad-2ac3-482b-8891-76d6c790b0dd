package sqlitedriver

/*
#cgo LDFLAGS: -l:libsqlite3.so.0
#include <stdlib.h>
typedef struct sqlite3 sqlite3;
typedef struct sqlite3_stmt sqlite3_stmt;
int sqlite3_open(const char*, sqlite3**);
int sqlite3_close(sqlite3*);
const char *sqlite3_errmsg(sqlite3*);
int sqlite3_prepare_v2(sqlite3*, const char*, int, sqlite3_stmt**, const char**);
int sqlite3_finalize(sqlite3_stmt*);
int sqlite3_reset(sqlite3_stmt*);
int sqlite3_clear_bindings(sqlite3_stmt*);
int sqlite3_bind_parameter_count(sqlite3_stmt*);
int sqlite3_bind_null(sqlite3_stmt*, int);
int sqlite3_bind_int64(sqlite3_stmt*, int, long long);
int sqlite3_bind_double(sqlite3_stmt*, int, double);
int sqlite3_bind_text(sqlite3_stmt*, int, const char*, int, void(*)(void*));
int sqlite3_bind_blob(sqlite3_stmt*, int, const void*, int, void(*)(void*));
int sqlite3_step(sqlite3_stmt*);
int sqlite3_column_count(sqlite3_stmt*);
const char *sqlite3_column_name(sqlite3_stmt*, int);
int sqlite3_column_type(sqlite3_stmt*, int);
long long sqlite3_column_int64(sqlite3_stmt*, int);
double sqlite3_column_double(sqlite3_stmt*, int);
const unsigned char *sqlite3_column_text(sqlite3_stmt*, int);
const void *sqlite3_column_blob(sqlite3_stmt*, int);
int sqlite3_column_bytes(sqlite3_stmt*, int);
long long sqlite3_last_insert_rowid(sqlite3*);
int sqlite3_changes(sqlite3*);

int benzhi_bind_text_copy(sqlite3_stmt *s, int n, const char *v, int len) {
    return sqlite3_bind_text(s, n, v, len, (void(*)(void*))-1);
}
int benzhi_bind_blob_copy(sqlite3_stmt *s, int n, const void *v, int len) {
    return sqlite3_bind_blob(s, n, v, len, (void(*)(void*))-1);
}
*/
import "C"

const (
	sqliteOK      = 0
	sqliteInteger = 1
	sqliteFloat   = 2
	sqliteText    = 3
	sqliteBlob    = 4
	sqliteNull    = 5
	sqliteRow     = 100
	sqliteDone    = 101
)
