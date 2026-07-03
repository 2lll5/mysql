package main

import (
	"database/sql/driver"
	"errors"
	"io"
	"net"
	"strings"
	"sync"
	"syscall"
)

// mysqlConn contains the pieces of connection state that must be considered
// before translating a connection failure to driver.ErrBadConn. Returning
// driver.ErrBadConn tells database/sql that it may transparently retry the
// operation on a different connection, so it is only safe before the current
// command has put bytes on the wire and while no transaction is active.
type mysqlConn struct {
	mu            sync.Mutex
	closed        bool
	writeStarted  bool
	inTransaction bool
	prepared      map[uint32]*preparedStmt
	flags         uint32
}

type preparedStmt struct {
	id    uint32
	query string
}

func newMysqlConn() *mysqlConn {
	return &mysqlConn{prepared: make(map[uint32]*preparedStmt)}
}

func (mc *mysqlConn) markCommandStart() {
	mc.mu.Lock()
	mc.writeStarted = false
	mc.mu.Unlock()
}

func (mc *mysqlConn) markWriteStarted() {
	mc.mu.Lock()
	mc.writeStarted = true
	mc.mu.Unlock()
}

func (mc *mysqlConn) beginTx() {
	mc.mu.Lock()
	mc.inTransaction = true
	mc.mu.Unlock()
}

func (mc *mysqlConn) endTx() {
	mc.mu.Lock()
	mc.inTransaction = false
	mc.mu.Unlock()
}

func (mc *mysqlConn) addPrepared(id uint32, query string) {
	mc.mu.Lock()
	if mc.prepared == nil {
		mc.prepared = make(map[uint32]*preparedStmt)
	}
	mc.prepared[id] = &preparedStmt{id: id, query: query}
	mc.mu.Unlock()
}

// cleanup invalidates all driver-side state associated with a bad connection.
// It is called before surfacing a connection failure, ensuring stale prepared
// statement handles and transaction flags cannot be reused accidentally.
func (mc *mysqlConn) cleanup() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.closed = true
	mc.writeStarted = false
	mc.inTransaction = false
	mc.flags = 0
	mc.prepared = make(map[uint32]*preparedStmt)
}

func (mc *mysqlConn) canRetryLocked() bool {
	return !mc.writeStarted && !mc.inTransaction
}

// handleError maps connection-reset style errors to driver.ErrBadConn only
// when database/sql can safely retry: idle connection failure or before any
// bytes for the current command were written. Once a write starts, or while a
// transaction is active, the original error is returned to avoid duplicating a
// non-idempotent operation on a fresh connection.
func (mc *mysqlConn) handleError(err error) error {
	if err == nil {
		return nil
	}
	if !isBadConnError(err) {
		return err
	}

	mc.mu.Lock()
	safeToRetry := mc.canRetryLocked()
	mc.mu.Unlock()
	mc.cleanup()

	if safeToRetry {
		return driver.ErrBadConn
	}
	return err
}

func isBadConnError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, driver.ErrBadConn) || errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, net.ErrClosed) || errors.Is(err, syscall.ECONNRESET) || errors.Is(err, syscall.EPIPE) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) && !netErr.Timeout() {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "connection reset") || strings.Contains(msg, "broken pipe") || strings.Contains(msg, "use of closed network connection")
}
