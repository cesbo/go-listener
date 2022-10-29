package listener

import (
	"net"
	"sync"
)

type connlimitListener struct {
	net.Listener
	once    sync.Once
	queueCh chan struct{}
	closeCh chan struct{}
}

func NewConnlimitListener(inner net.Listener, connlimit int) net.Listener {
	return &connlimitListener{
		Listener: inner,
		queueCh:  make(chan struct{}, connlimit),
		closeCh:  make(chan struct{}),
	}
}

func (c *connlimitListener) acquire() bool {
	select {
	case <-c.closeCh:
		return false
	case c.queueCh <- struct{}{}:
		return true
	}
}

func (c *connlimitListener) release() {
	<-c.queueCh
}

type connlimitConn struct {
	net.Conn
	once    sync.Once
	release func()
}

func (c *connlimitConn) Close() error {
	err := c.Conn.Close()
	c.once.Do(c.release)
	return err
}

func (c *connlimitListener) Accept() (net.Conn, error) {
	isAcquire := c.acquire()

	conn, err := c.Listener.Accept()
	if err != nil {
		if isAcquire {
			c.release()
		}
		return nil, err
	}

	return &connlimitConn{
		Conn:    conn,
		release: c.release,
	}, nil
}

func (c *connlimitListener) Close() error {
	err := c.Listener.Close()
	c.once.Do(func() {
		close(c.closeCh)
	})

	return err
}
