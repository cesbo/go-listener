package listener

import (
	"io"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_Listener(t *testing.T) {
	t.Run("Dial limit", func(t *testing.T) {
		assert := assert.New(t)

		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if !assert.NoError(err) {
			return
		}

		ln = NewConnlimitListener(ln, 1)
		defer ln.Close()

		network := ln.Addr().Network()
		address := ln.Addr().String()

		queueCh := make(chan struct{}, 2)

		go func(l net.Listener) {
			for i := 0; i < 2; i++ {
				_, err := l.Accept()
				if err != nil {
					break
				}

				queueCh <- struct{}{}
			}
		}(ln)

		// First connection should be accepted
		conn1, err := net.Dial(network, address)
		if !assert.NoError(err) {
			return
		}
		defer conn1.Close()

		// Second connection should be queued
		conn2, err := net.Dial(network, address)
		if !assert.NoError(err) {
			return
		}
		defer conn2.Close()

		select {
		case <-queueCh:
		case <-time.After(time.Millisecond * 100):
			assert.Fail("timeout")
			return
		}

		select {
		case <-queueCh:
			assert.Fail("unexpected connection")
			return
		case <-time.After(time.Millisecond * 100):
		}
	})

	t.Run("Dial in series", func(t *testing.T) {
		assert := assert.New(t)

		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if !assert.NoError(err) {
			return
		}
		ln = NewConnlimitListener(ln, 1)
		defer ln.Close()

		network := ln.Addr().Network()
		address := ln.Addr().String()

		queueCh := make(chan struct{}, 2)

		go func(l net.Listener) {
			for i := 0; i < 2; i++ {
				conn, err := l.Accept()
				if err != nil {
					break
				}

				var out [1]byte
				if _, err := conn.Read(out[:]); !assert.ErrorIs(err, io.EOF) {
					break
				} else {
					conn.Close()
				}

				queueCh <- struct{}{}
			}
		}(ln)

		// First connection should be accepted
		conn1, err := net.Dial(network, address)
		if !assert.NoError(err) {
			return
		}
		conn1.Close()

		// Second connection should be accepted
		conn2, err := net.Dial(network, address)
		if !assert.NoError(err) {
			return
		}
		conn2.Close()

		select {
		case <-queueCh:
		case <-time.After(time.Millisecond * 100):
			assert.Fail("timeout conn1")
			return
		}

		select {
		case <-queueCh:
		case <-time.After(time.Millisecond * 100):
			assert.Fail("timeout conn2")
			return
		}
	})
}
