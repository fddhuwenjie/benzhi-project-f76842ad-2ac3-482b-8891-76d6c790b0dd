package sqlitedriver

/*
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
int benzhi_bind_text_copy(sqlite3_stmt*, int, const char*, int);
int benzhi_bind_blob_copy(sqlite3_stmt*, int, const void*, int);
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
*/
import "C"

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"time"
	"unsafe"
)

func init() { sql.Register("benzhi_sqlite", &sqliteDriver{}) }

type sqliteDriver struct{}
type conn struct {
	db     *C.sqlite3
	closed bool
}
type statement struct {
	conn   *conn
	stmt   *C.sqlite3_stmt
	query  string
	closed bool
}
type transaction struct {
	conn *conn
	done bool
}
type rows struct {
	stmt    *statement
	columns []string
	done    bool
}
type result struct {
	lastID   int64
	affected int64
}

func (d *sqliteDriver) Open(name string) (driver.Conn, error) {
	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))
	var db *C.sqlite3
	if code := C.sqlite3_open(cname, &db); code != sqliteOK {
		message := "无法打开 SQLite"
		if db != nil {
			message = C.GoString(C.sqlite3_errmsg(db))
			C.sqlite3_close(db)
		}
		return nil, errors.New(message)
	}
	return &conn{db: db}, nil
}

func (c *conn) Prepare(query string) (driver.Stmt, error) {
	if c.closed {
		return nil, driver.ErrBadConn
	}
	cquery := C.CString(query)
	defer C.free(unsafe.Pointer(cquery))
	var stmt *C.sqlite3_stmt
	if code := C.sqlite3_prepare_v2(c.db, cquery, -1, &stmt, nil); code != sqliteOK {
		return nil, c.err(code)
	}
	return &statement{conn: c, stmt: stmt, query: query}, nil
}
func (c *conn) Close() error {
	if c.closed {
		return nil
	}
	if code := C.sqlite3_close(c.db); code != sqliteOK {
		return c.err(code)
	}
	c.closed = true
	return nil
}
func (c *conn) Begin() (driver.Tx, error) { return c.begin(context.Background()) }
func (c *conn) BeginTx(ctx context.Context, _ driver.TxOptions) (driver.Tx, error) {
	return c.begin(ctx)
}
func (c *conn) begin(ctx context.Context) (driver.Tx, error) {
	if err := c.execDirect(ctx, "BEGIN"); err != nil {
		return nil, err
	}
	return &transaction{conn: c}, nil
}
func (c *conn) Ping(ctx context.Context) error { return c.execDirect(ctx, "SELECT 1") }
func (c *conn) execDirect(ctx context.Context, query string) error {
	stmt, err := c.Prepare(query)
	if err != nil {
		return err
	}
	defer stmt.Close()
	_, err = stmt.(*statement).ExecContext(ctx, nil)
	return err
}
func (c *conn) err(code C.int) error {
	return fmt.Errorf("sqlite code %d: %s", int(code), C.GoString(C.sqlite3_errmsg(c.db)))
}

func (t *transaction) Commit() error {
	if t.done {
		return errors.New("事务已结束")
	}
	t.done = true
	return t.conn.execDirect(context.Background(), "COMMIT")
}
func (t *transaction) Rollback() error {
	if t.done {
		return nil
	}
	t.done = true
	return t.conn.execDirect(context.Background(), "ROLLBACK")
}

func (s *statement) Close() error {
	if s.closed {
		return nil
	}
	s.closed = true
	if code := C.sqlite3_finalize(s.stmt); code != sqliteOK {
		return s.conn.err(code)
	}
	return nil
}
func (s *statement) NumInput() int { return int(C.sqlite3_bind_parameter_count(s.stmt)) }
func (s *statement) Exec(values []driver.Value) (driver.Result, error) {
	args := make([]driver.NamedValue, len(values))
	for i, value := range values {
		args[i] = driver.NamedValue{Ordinal: i + 1, Value: value}
	}
	return s.ExecContext(context.Background(), args)
}
func (s *statement) Query(values []driver.Value) (driver.Rows, error) {
	args := make([]driver.NamedValue, len(values))
	for i, value := range values {
		args[i] = driver.NamedValue{Ordinal: i + 1, Value: value}
	}
	return s.QueryContext(context.Background(), args)
}
func (s *statement) ExecContext(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
	if err := s.prepareRun(ctx, args); err != nil {
		return nil, err
	}
	code := C.sqlite3_step(s.stmt)
	if code != sqliteDone {
		return nil, s.conn.err(code)
	}
	return result{lastID: int64(C.sqlite3_last_insert_rowid(s.conn.db)), affected: int64(C.sqlite3_changes(s.conn.db))}, nil
}
func (s *statement) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	if err := s.prepareRun(ctx, args); err != nil {
		return nil, err
	}
	count := int(C.sqlite3_column_count(s.stmt))
	columns := make([]string, count)
	for i := range columns {
		columns[i] = C.GoString(C.sqlite3_column_name(s.stmt, C.int(i)))
	}
	return &rows{stmt: s, columns: columns}, nil
}
func (s *statement) prepareRun(ctx context.Context, args []driver.NamedValue) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	if s.closed {
		return errors.New("语句已关闭")
	}
	C.sqlite3_reset(s.stmt)
	C.sqlite3_clear_bindings(s.stmt)
	if len(args) != s.NumInput() {
		return fmt.Errorf("参数数量为 %d，期望 %d", len(args), s.NumInput())
	}
	for _, arg := range args {
		if err := s.bind(arg.Ordinal, arg.Value); err != nil {
			return err
		}
	}
	return nil
}
func (s *statement) bind(index int, value any) error {
	var code C.int
	switch v := value.(type) {
	case nil:
		code = C.sqlite3_bind_null(s.stmt, C.int(index))
	case int64:
		code = C.sqlite3_bind_int64(s.stmt, C.int(index), C.longlong(v))
	case float64:
		code = C.sqlite3_bind_double(s.stmt, C.int(index), C.double(v))
	case bool:
		integer := int64(0)
		if v {
			integer = 1
		}
		code = C.sqlite3_bind_int64(s.stmt, C.int(index), C.longlong(integer))
	case string:
		ptr := C.CString(v)
		defer C.free(unsafe.Pointer(ptr))
		code = C.benzhi_bind_text_copy(s.stmt, C.int(index), ptr, C.int(len(v)))
	case []byte:
		if len(v) == 0 {
			code = C.benzhi_bind_blob_copy(s.stmt, C.int(index), nil, 0)
		} else {
			ptr := C.CBytes(v)
			defer C.free(ptr)
			code = C.benzhi_bind_blob_copy(s.stmt, C.int(index), ptr, C.int(len(v)))
		}
	case time.Time:
		text := v.UTC().Format(time.RFC3339Nano)
		ptr := C.CString(text)
		defer C.free(unsafe.Pointer(ptr))
		code = C.benzhi_bind_text_copy(s.stmt, C.int(index), ptr, C.int(len(text)))
	default:
		return fmt.Errorf("不支持的 SQLite 参数类型 %T", value)
	}
	if code != sqliteOK {
		return s.conn.err(code)
	}
	return nil
}

func (r result) LastInsertId() (int64, error) { return r.lastID, nil }
func (r result) RowsAffected() (int64, error) { return r.affected, nil }
func (r *rows) Columns() []string             { return r.columns }
func (r *rows) Close() error                  { r.done = true; return nil }
func (r *rows) Next(dest []driver.Value) error {
	if r.done {
		return io.EOF
	}
	code := C.sqlite3_step(r.stmt.stmt)
	if code == sqliteDone {
		r.done = true
		return io.EOF
	}
	if code != sqliteRow {
		return r.stmt.conn.err(code)
	}
	for index := range dest {
		kind := int(C.sqlite3_column_type(r.stmt.stmt, C.int(index)))
		switch kind {
		case sqliteNull:
			dest[index] = nil
		case sqliteInteger:
			dest[index] = int64(C.sqlite3_column_int64(r.stmt.stmt, C.int(index)))
		case sqliteFloat:
			dest[index] = float64(C.sqlite3_column_double(r.stmt.stmt, C.int(index)))
		case sqliteText:
			length := C.sqlite3_column_bytes(r.stmt.stmt, C.int(index))
			dest[index] = C.GoStringN((*C.char)(unsafe.Pointer(C.sqlite3_column_text(r.stmt.stmt, C.int(index)))), length)
		case sqliteBlob:
			length := C.sqlite3_column_bytes(r.stmt.stmt, C.int(index))
			dest[index] = C.GoBytes(C.sqlite3_column_blob(r.stmt.stmt, C.int(index)), length)
		default:
			return fmt.Errorf("未知 SQLite 列类型 %d", kind)
		}
	}
	return nil
}

var _ driver.Conn = (*conn)(nil)
var _ driver.ConnBeginTx = (*conn)(nil)
var _ driver.Pinger = (*conn)(nil)
var _ driver.StmtExecContext = (*statement)(nil)
var _ driver.StmtQueryContext = (*statement)(nil)
