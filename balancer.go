package grpcx

import (
	"errors"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/naming"
)

var ErrWatcherClose = errors.New("watcher has been closed")

func NewDnsBalancer() grpc.Balancer {
	r, _ := naming.NewDNSResolver()
	return grpc.RoundRobin(r)
}

// Example
// conn, err := grpc.Dial(
// 	"shark",
// 	grpc.WithInsecure(),
// 	grpc.WithBalancer(grpc.RoundRobin((NewHelpResolver.serverAddrs))),
// )

func NewMultiAddrBalancer(addrs []string) grpc.Balancer {
	r := NewHelpResolver(addrs)
	return grpc.RoundRobin(r)
}

// NewHelpResolver creates a new pseudo resolver which returns fixed addrs.
func NewHelpResolver(addrs []string) naming.Resolver {
	return &HelpResolver{
		addrs: addrs,
	}
}

type HelpResolver struct {
	addrs   []string
	watcher *helpWatcher
	sync.Mutex
}

// Add dynamic add new target
func (r *HelpResolver) Add(target string) error {
	r.Lock()
	defer r.Unlock()

	for _, addr := range r.addrs {
		if addr == target {
			return errors.New("target is existed")
		}
	}

	updates := []*naming.Update{&naming.Update{Op: naming.Add, Addr: target}}
	r.watcher.updatesChan <- updates
	return nil
}

// Resolve
func (r *HelpResolver) Resolve(target string) (naming.Watcher, error) {
	r.Lock()
	defer r.Unlock()

	w := &helpWatcher{
		updatesChan: make(chan []*naming.Update, 1),
	}
	updates := []*naming.Update{}
	for _, addr := range r.addrs {
		updates = append(updates, &naming.Update{Op: naming.Add, Addr: addr})
	}
	w.updatesChan <- updates
	r.watcher = w
	return w, nil
}

// This watcher is implemented based on ipwatcher below
// https://github.com/grpc/grpc-go/blob/30fb59a4304034ce78ff68e21bd25776b1d79488/naming/dns_resolver.go#L151-L171
type helpWatcher struct {
	updatesChan chan []*naming.Update
}

func (w *helpWatcher) Next() ([]*naming.Update, error) {
	us, ok := <-w.updatesChan
	if !ok {
		return nil, ErrWatcherClose
	}
	return us, nil
}

func (w *helpWatcher) Close() {
	close(w.updatesChan)
}
