package watcher

import (
	"context"
	"github.com/jin06/binlogo/pkg/etcdclient"
	"github.com/sirupsen/logrus"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// General a base watcher
// this will be use for most watchers
type General struct {
	key string
	//EventHandler func(*clientv3.Event, bool) (*Event, error)
	EventHandler Handler
}

// Handler function of handle event
type Handler func(*clientv3.Event) (*Event, error)

// NewGeneral returns a new General watcher
func NewGeneral(key string) (w *General, err error) {
	w = &General{
		key: key,
	}
	w.EventHandler = func(event *clientv3.Event) (*Event, error) {
		return nil, nil
	}
	return
}

// New returns a new General watcher with handler
func New(key string, handler Handler) (w *General, err error) {
	w = &General{key: key, EventHandler: handler}
	return
}

// GetKey returns General key
func (w *General) GetKey() string {
	return w.key
}

// WatchEtcd start watch etcd changes
func (w *General) WatchEtcd(ctx context.Context, opts ...clientv3.OpOption) (ch chan *Event, err error) {
	ch = make(chan *Event, 1000)
	defer func() {
		if err != nil {
			close(ch)
		}
	}()
	go func() {
		watchCh := etcdclient.Default().Watch(ctx, w.key, opts...)
		defer func() {
			close(ch)
		}()
		//LOOP:
		for {
			select {
			case resp, ok := <-watchCh:
				{
					if !ok {
						return
						//break LOOP
					}
					if resp.Err() != nil {
						logrus.Error(resp.Err())
					}
					for _, val := range resp.Events {
						ev, err2 := w.EventHandler(val)
						if err2 != nil {
							logrus.Error("Event handle error: ", err2)
							continue
						}
						ch <- ev
					}
				}
			case <-ctx.Done():
				{
					return
				}
			}
		}

	}()
	return
}

// WatchEtcdList start watch etcd changes for list
func (w *General) WatchEtcdList(ctx context.Context) (ch chan *Event, err error) {
	return w.WatchEtcd(ctx, clientv3.WithPrefix())
}
