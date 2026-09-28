package sqljs

import (
	"database/sql"
	"database/sql/driver"
	_ "fmt"
	"io"
	"strconv"
	"syscall/js"
)

const (
	driverName = "sqljs"
)

type Driver struct{}

type conn struct {
	db js.Value
}

type tx struct {
	c *conn
}

type stmt struct {
	c       *conn
	columns []string
	s       js.Value
}

type rows struct {
	s       *stmt
	hasRows bool
}

func init() {
	sql.Register(driverName, &Driver{})
}

func (d *Driver) Open(name string) (c driver.Conn, err error) {
	window := js.Global().Get("window")
	db := window.Get(name)
	c = &conn{db}
	return c, nil
}

func (c *conn) Begin() (dt driver.Tx, err error) {
	c.db.Call("run", "BEGIN;")
	return &tx{c}, nil
}

func (c *conn) Close() error {
	return nil
}

func (c *conn) Prepare(query string) (ds driver.Stmt, err error) {
	s := c.db.Call("prepare", query)
	return &stmt{c: c, s: s}, nil
}

func (t *tx) Commit() (err error) {
	t.c.db.Call("run", "COMMIT;")
	return nil
}

func (t *tx) Rollback() (err error) {
	t.c.db.Call("run", "ROLLBACK;")
	return nil
}

func (s *stmt) Close() (err error) {
	return nil
}

func (s *stmt) Exec(args []driver.Value) (driver.Result, error) {
	new_args := make([]any, len(args))
	for i, arg := range args {
		new_args[i] = js.ValueOf(arg)
	}
	s.s.Call("bind", new_args)
	s.s.Call("step")
	return nil, nil
}

func (s *stmt) NumInput() (n int) {
	return -1
}

func (s *stmt) Query(args []driver.Value) (driver.Rows, error) {
	new_args := make([]any, len(args))
	for i, arg := range args {
		new_args[i] = js.ValueOf(arg)
	}
	s.s.Call("bind", new_args)
	hasRows := s.s.Call("step").Bool()
	if s.columns == nil {
		columns_value := s.s.Call("getColumnNames")
		s.columns = make([]string, columns_value.Length())
		for i, _ := range s.columns {
			s.columns[i] = columns_value.Get(strconv.Itoa(i)).String()
		}
	}
	r := &rows{s, hasRows}
	return r, nil
}

func (r *rows) Close() (err error) {
	return nil
}

func (r *rows) Columns() (c []string) {
	return r.s.columns
}

func (r *rows) Next(dest []driver.Value) (err error) {
	if !r.hasRows {
		return io.EOF
	}

	row := r.s.s.Call("get")
	for i := 0; i < row.Length(); i++ {
		value := row.Get(strconv.Itoa(i))
		switch val_type := value.Type(); val_type {
		case js.TypeNumber:
			dest[i] = value.Float()
		case js.TypeString:
			dest[i] = value.String()
		case js.TypeBoolean:
			dest[i] = value.Bool()
		}
	}
	r.hasRows = r.s.s.Call("step").Bool()

	return nil
}
