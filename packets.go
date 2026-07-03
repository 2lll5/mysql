package main

import "io"

type packetConn interface {
	Write([]byte) (int, error)
}

// writePacket marks the current command as having written to the wire before
// delegating to net.Conn.Write. If the write fails, handleError will preserve
// the original network error rather than returning driver.ErrBadConn and
// causing database/sql to retry an operation that may have reached the server.
func (mc *mysqlConn) writePacket(conn packetConn, packet []byte) error {
	mc.markWriteStarted()
	_, err := conn.Write(packet)
	return mc.handleError(err)
}

// readPacket may safely translate idle/pre-write EOF into driver.ErrBadConn;
// after a command write has started, handleError returns the raw error.
func (mc *mysqlConn) readPacket(r io.Reader, buf []byte) error {
	_, err := io.ReadFull(r, buf)
	return mc.handleError(err)
}
