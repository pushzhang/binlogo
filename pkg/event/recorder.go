package event

import (
	"context"
	"strings"
	"time"

	"github.com/golang/groupcache/lru"
	"github.com/jin06/binlogo/pkg/store/dao/dao_event"
	"github.com/jin06/binlogo/pkg/store/model/event"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// Recorder record event.
// Aggregate events and delete old events regularly
type Recorder struct {
	incomeChan chan *event.Event
	flushChan  chan *event.Event
	cache      *lru.Cache
	nodeName   string
	flushMap   map[string]*event.Event
}

// New Returns a new Recorder
func New() (*Recorder, error) {
	r := &Recorder{}
	r.incomeChan = make(chan *event.Event, 16384)
	r.flushChan = make(chan *event.Event, 4096)
	r.cache = lru.New(4096)
	r.nodeName = viper.GetString("node.name")
	r.flushMap = map[string]*event.Event{}
	return r, nil
}

// Loop start event record loop
func (r Recorder) Loop(ctx context.Context) {
	go r._dispatch(ctx)
	go r._send(ctx)
}

func (r Recorder) _dispatch(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			{
				return
			}
		case e := <-r.incomeChan:
			{
				r.dispatch(e)
			}
		}
	}
}

func (r Recorder) _send(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			{
				return
			}
		case e := <-r.flushChan:
			{
				r.flushMap[e.Key] = e
				r.flush(false)
			}
		case <-time.Tick(time.Second * 10):
			{
				r.flush(true)
			}
		}
	}
}

func (r Recorder) flush(force bool) {
	if force || len(r.flushMap) >= 100 {
		for _, v := range r.flushMap {
			er := dao_event.Update(v)
			if er != nil {
				logrus.Errorln("Write event to etcd failed: ", er)
			}
		}
		r.flushMap = map[string]*event.Event{}
	}
}

// Event pass event to income chan
func (r Recorder) Event(e *event.Event) {
	r.incomeChan <- e
	return
}

func (r Recorder) dispatch(new *event.Event) {
	aggKey := aggregatorKey(new)
	oldInter, ok := r.cache.Get(aggKey)
	if ok {
		if old, is := oldInter.(*event.Event); is {
			if isExceedTime(time.Now(), old.FirstTime) {
				old.Count = old.Count + 1
				old.LastTime = time.Now()
				r.cache.Add(aggKey, old)
				r.update(old)
				return
			} else {
				r.cache.Remove(aggKey)
			}
		}
	}
	new.Count = 1
	//new.Key = key.GetKey()
	new.FirstTime = time.Now()
	new.LastTime = time.Now()
	new.NodeName = r.nodeName
	r.cache.Add(aggKey, new)
	r.update(new)
}

func (r Recorder) update(e *event.Event) {
	r.flushChan <- e
}

func aggregatorKey(e *event.Event) string {
	return strings.Join([]string{
		string(e.Type),
		string(e.ResourceType),
		e.ResourceName,
		e.Message,
		e.NodeName,
	},
		".",
	)
}

func isExceedTime(newTime time.Time, oldTime time.Time) bool {
	exceedTime := time.Minute * 5
	return oldTime.Add(exceedTime).Before(newTime)
}
