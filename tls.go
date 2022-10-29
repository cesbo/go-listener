package listener

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
)

type tlsListener struct {
	net.Listener

	cert string
	key  string

	lockCrt sync.RWMutex
	crt     *tls.Certificate

	once    sync.Once
	closeCh chan struct{}
}

func NewTlsListener(inner net.Listener, cert, key string) net.Listener {
	t := &tlsListener{
		cert:    cert,
		key:     key,
		closeCh: make(chan struct{}),
	}

	go t.watcher()

	t.Listener = tls.NewListener(
		inner, &tls.Config{
			GetCertificate: t.getCertificate,
		},
	)

	return t
}

func (t *tlsListener) getCertificate(_ *tls.ClientHelloInfo) (*tls.Certificate, error) {
	t.lockCrt.RLock()
	defer t.lockCrt.RUnlock()

	if t.crt == nil {
		return nil, fmt.Errorf("tls listener: certificate is not set")
	}

	return t.crt, nil
}

func (t *tlsListener) update() {
	crt, err := tls.LoadX509KeyPair(t.cert, t.key)
	if err != nil {
		log.Printf("tls listener: load certificate: %s", err)
		return
	}

	t.lockCrt.Lock()
	defer t.lockCrt.Unlock()

	t.crt = &crt
}

func certModified(event fsnotify.Event, cert string) bool {
	return filepath.Clean(event.Name) == cert && event.Has(fsnotify.Create|fsnotify.Write)
}

func symlinkModified(cur, prev string) bool {
	return cur != "" && cur != prev
}

func (t *tlsListener) watcher() {
	t.update()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Printf("tls listener: watcher: %s", err)
		return
	}
	defer watcher.Close()

	// https://github.com/cloudflare/certinel/
	certPath := filepath.Clean(t.cert)
	certDir, _ := filepath.Split(certPath)
	realCertPath, _ := filepath.EvalSymlinks(certPath)

	if err := watcher.Add(certDir); err != nil {
		log.Printf("tls listener: watcher: %s", err)
		return
	}

	for {
		select {
		case <-t.closeCh:
			return

		case event := <-watcher.Events:
			currentPath, err := filepath.EvalSymlinks(certPath)
			if err != nil {
				continue
			}

			if certModified(event, certPath) || symlinkModified(currentPath, realCertPath) {
				realCertPath = currentPath
				t.update()
				log.Println("tls listener: watcher: certificate updated")
			}

		case err := <-watcher.Errors:
			log.Printf("tls listener: watcher error: %s", err)
		}
	}
}

func (t *tlsListener) Close() error {
	err := t.Listener.Close()
	t.once.Do(func() {
		close(t.closeCh)
	})

	return err
}
