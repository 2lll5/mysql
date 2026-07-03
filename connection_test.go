package main

import (
	"database/sql/driver"
	"errors"
	"io"
	"reflect"
	"testing"
)

func TestHandleErrorReturnsErrBadConnOnlyBeforeWrite(t *testing.T) {
	mc := newMysqlConn()
	mc.markCommandStart()

	if err := mc.handleError(io.EOF); !errors.Is(err, driver.ErrBadConn) {
		t.Fatalf("pre-write EOF = %v, want driver.ErrBadConn", err)
	}

	mc = newMysqlConn()
	mc.markCommandStart()
	mc.markWriteStarted()
	if err := mc.handleError(io.ErrUnexpectedEOF); errors.Is(err, driver.ErrBadConn) {
		t.Fatalf("mid-write failure returned driver.ErrBadConn; want original error")
	} else if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("mid-write failure = %v, want original io.ErrUnexpectedEOF", err)
	}
}

func TestHandleErrorDoesNotRetryInsideTransaction(t *testing.T) {
	mc := newMysqlConn()
	mc.beginTx()

	if err := mc.handleError(io.EOF); errors.Is(err, driver.ErrBadConn) {
		t.Fatalf("transaction failure returned driver.ErrBadConn; want original error")
	} else if !errors.Is(err, io.EOF) {
		t.Fatalf("transaction failure = %v, want EOF", err)
	}
}

func TestCleanupInvalidatesState(t *testing.T) {
	mc := newMysqlConn()
	mc.beginTx()
	mc.markWriteStarted()
	mc.addPrepared(1, "SELECT 1")
	mc.flags = 99

	_ = mc.handleError(io.EOF)

	if !mc.closed {
		t.Fatal("connection was not marked closed")
	}
	if mc.inTransaction || mc.writeStarted || mc.flags != 0 {
		t.Fatalf("state was not reset: inTx=%v writeStarted=%v flags=%d", mc.inTransaction, mc.writeStarted, mc.flags)
	}
	if len(mc.prepared) != 0 {
		t.Fatalf("prepared statement cache was not cleared: %v", mc.prepared)
	}
}

func TestConnectorAppliesSessionVarsBeforeReturningConnection(t *testing.T) {
	fake := &fakeSessionConn{}
	c := &connector{
		systemVars: map[string]string{
			"sql_mode":  "'ANSI'",
			"time_zone": "'+00:00'",
		},
		dial: func() (sessionConn, error) { return fake, nil },
	}

	conn, err := c.Connect()
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	if conn != fake {
		t.Fatalf("Connect() returned unexpected connection")
	}
	want := []string{"SET SESSION `sql_mode` = 'ANSI'", "SET SESSION `time_zone` = '+00:00'"}
	if !reflect.DeepEqual(fake.queries, want) {
		t.Fatalf("session init queries = %#v, want %#v", fake.queries, want)
	}
	if fake.closed {
		t.Fatal("initialized connection was closed")
	}
}

func TestConnectorClosesPartiallyInitializedConnectionOnError(t *testing.T) {
	fake := &fakeSessionConn{failAfter: 1, failErr: io.ErrUnexpectedEOF}
	c := &connector{
		systemVars: map[string]string{"sql_mode": "'ANSI'", "time_zone": "'+00:00'"},
		dial:       func() (sessionConn, error) { return fake, nil },
	}

	if _, err := c.Connect(); err == nil {
		t.Fatal("Connect() succeeded; want init error")
	}
	if !fake.closed {
		t.Fatal("partially initialized connection was not closed")
	}
}

type fakeSessionConn struct {
	queries   []string
	closed    bool
	failAfter int
	failErr   error
}

func (f *fakeSessionConn) Exec(query string) error {
	f.queries = append(f.queries, query)
	if f.failAfter > 0 && len(f.queries) >= f.failAfter {
		return f.failErr
	}
	return nil
}

func (f *fakeSessionConn) Close() error {
	f.closed = true
	return nil
}
