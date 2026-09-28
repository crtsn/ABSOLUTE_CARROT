package libsqlite3

/*
#cgo LDFLAGS: -lsqlite3

#include <sqlite3.h>
#include <stdlib.h>

// These wrappers are necessary because SQLITE_TRANSIENT
// is a pointer constant, and cgo doesn't translate them correctly.
// The definition in sqlite3.h is:
//
// typedef void (*sqlite3_destructor_type)(void*);
// #define SQLITE_STATIC      ((sqlite3_destructor_type)0)
// #define SQLITE_TRANSIENT   ((sqlite3_destructor_type)-1)

static int my_bind_text(sqlite3_stmt *stmt, int n, char *p, int np) {
	return sqlite3_bind_text(stmt, n, p, np, SQLITE_TRANSIENT);
}
*/
import "C"

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"strings"
	"unsafe"
)

const (
	driverName = "libsqlite3"
)

type Driver struct{}

type conn struct {
	db *C.sqlite3
}

type tx struct {
	c *conn
}

type stmt struct {
	c        *conn
	s        *C.sqlite3_stmt
	columns  []string
	coltypes []string
}

type rows struct {
	s *stmt
}

type errno int

func (e errno) Error() string {
	s := errText[e]
	if s == "" {
		return fmt.Sprintf("errno %d", int(e))
	}
	return s
}

var (
	errError      error = errno(1)   //    /* SQL error or missing database */
	errInternal   error = errno(2)   //    /* Internal logic error in SQLite */
	errPerm       error = errno(3)   //    /* Access permission denied */
	errAbort      error = errno(4)   //    /* Callback routine requested an abort */
	errBusy       error = errno(5)   //    /* The database file is locked */
	errLocked     error = errno(6)   //    /* A table in the database is locked */
	errNoMem      error = errno(7)   //    /* A malloc() failed */
	errReadOnly   error = errno(8)   //    /* Attempt to write a readonly database */
	errInterrupt  error = errno(9)   //    /* Operation terminated by sqlite3_interrupt()*/
	errIOErr      error = errno(10)  //    /* Some kind of disk I/O error occurred */
	errCorrupt    error = errno(11)  //    /* The database disk image is malformed */
	errFull       error = errno(13)  //    /* Insertion failed because database is full */
	errCantOpen   error = errno(14)  //    /* Unable to open the database file */
	errEmpty      error = errno(16)  //    /* Database is empty */
	errSchema     error = errno(17)  //    /* The database schema changed */
	errTooBig     error = errno(18)  //    /* String or BLOB exceeds size limit */
	errConstraint error = errno(19)  //    /* Abort due to constraint violation */
	errMismatch   error = errno(20)  //    /* Data type mismatch */
	errMisuse     error = errno(21)  //    /* Library used incorrectly */
	errNolfs      error = errno(22)  //    /* Uses OS features not supported on host */
	errAuth       error = errno(23)  //    /* Authorization denied */
	errFormat     error = errno(24)  //    /* Auxiliary database format error */
	errRange      error = errno(25)  //    /* 2nd parameter to sqlite3_bind out of range */
	errNotDB      error = errno(26)  //    /* File opened that is not a database file */
	stepRow             = errno(100) //   /* sqlite3_step() has another row ready */
	stepDone            = errno(101) //   /* sqlite3_step() has finished executing */
)

var errText = map[errno]string{
	1:   "SQL error or missing database",
	2:   "Internal logic error in SQLite",
	3:   "Access permission denied",
	4:   "Callback routine requested an abort",
	5:   "The database file is locked",
	6:   "A table in the database is locked",
	7:   "A malloc() failed",
	8:   "Attempt to write a readonly database",
	9:   "Operation terminated by sqlite3_interrupt()*/",
	10:  "Some kind of disk I/O error occurred",
	11:  "The database disk image is malformed",
	12:  "NOT USED. Table or record not found",
	13:  "Insertion failed because database is full",
	14:  "Unable to open the database file",
	15:  "NOT USED. Database lock protocol error",
	16:  "Database is empty",
	17:  "The database schema changed",
	18:  "String or BLOB exceeds size limit",
	19:  "Abort due to constraint violation",
	20:  "Data type mismatch",
	21:  "Library used incorrectly",
	22:  "Uses OS features not supported on host",
	23:  "Authorization denied",
	24:  "Auxiliary database format error",
	25:  "2nd parameter to sqlite3_bind out of range",
	26:  "File opened that is not a database file",
	100: "sqlite3_step() has another row ready",
	101: "sqlite3_step() has finished executing",
}

func init() {
	sql.Register(driverName, &Driver{})
}

func (d *Driver) Open(name string) (c driver.Conn, err error) {
	var db *C.sqlite3
	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))
	rv := C.sqlite3_open_v2(cname, &db,
		C.SQLITE_OPEN_FULLMUTEX|
			C.SQLITE_OPEN_READWRITE|
			C.SQLITE_OPEN_CREATE,
		nil)
	if rv != 0 {
		return nil, errno(rv)
	}
	if db == nil {
		return nil, errors.New("sqlite succeeded without returning a database")
	}
	return &conn{db}, nil
}

func (c *conn) exec(cmd string) error {
	cstring := C.CString(cmd)
	defer C.free(unsafe.Pointer(cstring))
	rv := C.sqlite3_exec(c.db, cstring, nil, nil, nil)
	return c.error(rv)
}

func (c *conn) Begin() (dt driver.Tx, err error) {
	if err := c.exec("BEGIN TRANSACTION"); err != nil {
		return nil, err
	}
	return &tx{c}, nil
}

func (c *conn) Close() error {
	return nil
}

func (c *conn) Prepare(query string) (ds driver.Stmt, err error) {
	querystr := C.CString(query)
	defer C.free(unsafe.Pointer(querystr))
	var s *C.sqlite3_stmt
	var tail *C.char
	rv := C.sqlite3_prepare_v2(c.db, querystr, C.int(len(query)+1), &s, &tail)
	if rv != 0 {
		return nil, c.error(rv)
	}
	return &stmt{c: c, s: s}, nil
}

func (c *conn) error(rv C.int) error {
	if rv == 0 {
		return nil
	}
	if rv == 21 {
		return fmt.Errorf("library used incorrectly: %d", rv)
	}
	return errors.New(errno(rv).Error() + ": " + C.GoString(C.sqlite3_errmsg(c.db)))
}

func (t *tx) Commit() (err error) {
	err = t.c.exec("COMMIT TRANSACTION")
	return err
}

func (t *tx) Rollback() (err error) {
	err = t.c.exec("ROLLBACK")
	return err
}

func (s *stmt) Close() (err error) {
	return nil
}

func (s *stmt) NumInput() (n int) {
	return -1
}

func (s *stmt) start(args []driver.Value) error {
	n := int(C.sqlite3_bind_parameter_count(s.s))
	if n != len(args) {
		return fmt.Errorf("incorrect argument count for command: have %d want %d", len(args), n)
	}

	for i, v := range args {
		var str string
		switch v := v.(type) {
		case nil:
			if rv := C.sqlite3_bind_null(s.s, C.int(i+1)); rv != 0 {
				return s.c.error(rv)
			}
			continue

		case float64:
			if rv := C.sqlite3_bind_double(s.s, C.int(i+1), C.double(v)); rv != 0 {
				return s.c.error(rv)
			}
			continue

		case int64:
			if rv := C.sqlite3_bind_int64(s.s, C.int(i+1), C.sqlite3_int64(v)); rv != 0 {
				return s.c.error(rv)
			}
			continue

		case bool:
			var vi int64
			if v {
				vi = 1
			}
			if rv := C.sqlite3_bind_int64(s.s, C.int(i+1), C.sqlite3_int64(vi)); rv != 0 {
				return s.c.error(rv)
			}
			continue

		case string:
			str = v

		default:
			str = fmt.Sprint(v)
		}

		cstr := C.CString(str)
		rv := C.my_bind_text(s.s, C.int(i+1), cstr, C.int(len(str)))
		C.free(unsafe.Pointer(cstr))
		if rv != 0 {
			return s.c.error(rv)
		}
	}

	return nil
}

func (s *stmt) Exec(args []driver.Value) (driver.Result, error) {
	err := s.start(args)
	if err != nil {
		return nil, err
	}

	rv := C.sqlite3_step(s.s)
	if errno(rv) != stepDone {
		if rv == 0 {
			rv = 21 // errMisuse
		}
		return nil, s.c.error(rv)
	}

	return nil, nil
}

func (s *stmt) Query(args []driver.Value) (driver.Rows, error) {
	err := s.start(args)
	if err != nil {
		return nil, err
	}

	if s.columns == nil {
		n := int64(C.sqlite3_column_count(s.s))
		s.columns = make([]string, n)
		s.coltypes = make([]string, n)
		for i := range s.columns {
			s.columns[i] = C.GoString(C.sqlite3_column_name(s.s, C.int(i)))
			s.coltypes[i] = strings.ToLower(C.GoString(C.sqlite3_column_decltype(s.s, C.int(i))))
		}
	}
	return &rows{s: s}, nil
}

func (r *rows) Close() (err error) {
	rv := C.sqlite3_reset(r.s.s)
	return r.s.c.error(rv)
}

func (r *rows) Columns() (c []string) {
	return r.s.columns
}

func (r *rows) Next(dest []driver.Value) (err error) {
	rv := C.sqlite3_step(r.s.s)
	if errno(rv) == stepDone {
		return io.EOF
	}
	if errno(rv) != stepRow {
		return r.s.c.error(rv)
	}

	for i := range dest {
		column := C.int(i)
		switch C.sqlite3_column_type(r.s.s, column) {
		case C.SQLITE_INTEGER:
			dest[i] = int64(C.sqlite3_column_int64(r.s.s, column))
		case C.SQLITE_FLOAT:
			dest[i] = float64(C.sqlite3_column_double(r.s.s, column))
		case C.SQLITE_TEXT:
			text := C.sqlite3_column_text(r.s.s, column)
			length := C.sqlite3_column_bytes(r.s.s, column)
			dest[i] = string(C.GoBytes(unsafe.Pointer(text), length))
		case C.SQLITE_BLOB:
			blob := C.sqlite3_column_blob(r.s.s, column)
			length := C.sqlite3_column_bytes(r.s.s, column)
			dest[i] = C.GoBytes(blob, length)
		case C.SQLITE_NULL:
			dest[i] = nil
		default:
			return fmt.Errorf("unsupported SQLite column type at index %d", i)
		}
	}
	return nil
}
