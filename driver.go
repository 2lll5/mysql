package main

import (
	"fmt"
	"sort"
	"strings"
)

type sessionConn interface {
	Exec(query string) error
	Close() error
}

type connector struct {
	systemVars map[string]string
	dial       func() (sessionConn, error)
}

func (c *connector) Connect() (sessionConn, error) {
	conn, err := c.dial()
	if err != nil {
		return nil, err
	}
	if err := c.initializeSession(conn); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return conn, nil
}

// initializeSession applies all DSN-configured session state before the
// connection is returned to the pool. If any statement fails, the partially
// initialized connection is closed and discarded instead of being reused with
// default server settings.
func (c *connector) initializeSession(conn sessionConn) error {
	for _, stmt := range c.sessionInitStatements() {
		if err := conn.Exec(stmt); err != nil {
			return fmt.Errorf("initialize mysql session: %w", err)
		}
	}
	return nil
}

func (c *connector) sessionInitStatements() []string {
	if len(c.systemVars) == 0 {
		return nil
	}
	keys := make([]string, 0, len(c.systemVars))
	for key := range c.systemVars {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	stmts := make([]string, 0, len(keys))
	for _, key := range keys {
		stmts = append(stmts, fmt.Sprintf("SET SESSION %s = %s", quoteIdentifier(key), c.systemVars[key]))
	}
	return stmts
}

func quoteIdentifier(name string) string {
	parts := strings.Split(name, ".")
	for i, part := range parts {
		parts[i] = "`" + strings.ReplaceAll(part, "`", "``") + "`"
	}
	return strings.Join(parts, ".")
}
