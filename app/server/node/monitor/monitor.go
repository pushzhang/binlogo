package monitor

import (
	"context"
	"sync"

	"github.com/jin06/binlogo/pkg/store/dao/dao_node"
	"github.com/jin06/binlogo/pkg/store/dao/dao_pipe"
	"github.com/jin06/binlogo/pkg/watcher"
	"github.com/jin06/binlogo/pkg/watcher/node"
	"github.com/jin06/binlogo/pkg/watcher/pipeline"
)

// Monitor monitor the operation of pipelines, nodes and other resources
type Monitor struct {
	status string
	lock   sync.Mutex
	cancel context.CancelFunc
	ctx    context.Context
	//nodeWatcher     *watcher.General
	//pipeWatcher     *watcher.General
	//registerWatcher *watcher.General
}

const STATUS_RUN = "run"
const STATUS_STOP = "stop"

// NewMonitor returns a new Monitor
func NewMonitor() (m *Monitor, err error) {
	m = &Monitor{
		status: STATUS_STOP,
	}
	//m.pipeWatcher, err = pipeline.New(dao_pipe.PipelinePrefix())
	//m.nodeWatcher, err = node.New(dao_node.NodePrefix())
	//m.registerWatcher, err = node.New(dao_cluster.RegisterPrefix())
	return
}

func (m *Monitor) init() (err error) {
	//m.pipeWatcher, err = pipeline.New(dao_pipe.PipelinePrefix())
	//if err != nil {
	//	return
	//}
	//m.nodeWatcher, err = node.New(dao_node.NodePrefix())
	//if err != nil {
	//	return
	//}
	//m.registerWatcher, err = node.New(dao_cluster.RegisterPrefix())
	//if err != nil {
	//	return
	//}
	return
}

// Run start working
func (m *Monitor) Run(ctx context.Context) (err error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	if m.status == STATUS_RUN {
		return
	}
	err = m.init()
	if err != nil {
		return
	}
	myCtx, cancel := context.WithCancel(ctx)
	m.ctx = myCtx
	defer func() {
		if err != nil {
			cancel()
		} else {
			m.status = STATUS_RUN
		}
	}()
	m.cancel = cancel
	nodeCtx, err := m.monitorNode(myCtx)
	if err != nil {
		return
	}
	pipeCtx, err := m.monitorPipe(myCtx)
	if err != nil {
		return
	}
	statusCtx, err := m.monitorStatus(myCtx)
	if err != nil {
		return
	}
	go func() {
		defer func() {
			cancel()
			m.status = STATUS_STOP
		}()
		select {
		case <-nodeCtx.Done():
			{
				return
			}
		case <-pipeCtx.Done():
			{
				return
			}
		case <-statusCtx.Done():
			{
				return
			}

		}
	}()

	return nil
}

func (m *Monitor) Stop(ctx context.Context) {
	m.lock.Lock()
	defer m.lock.Unlock()
	if m.status == STATUS_STOP {
		//blog.Debug("Monitor is stopped, do nothing")
		return
	}
	m.cancel()
	m.status = STATUS_STOP
	return
}

func (m *Monitor) newNodeWatcherCh(ctx context.Context) (ch chan *watcher.Event, err error) {
	//wa, err := node.New(dao_node.NodePrefix())
	//if err != nil {
	//	return
	//}
	//ch, err = wa.WatchEtcdList(ctx)
	return node.WatchList(ctx, dao_node.NodePrefix())
}

func (m *Monitor) newNodeRegWatcherCh(ctx context.Context) (ch chan *watcher.Event, err error) {
	//nodeRegWatcher, err := node.New(dao_node.NodeRegisterPrefix())
	//if err != nil {
	//	return
	//}
	//ch, err = nodeRegWatcher.WatchEtcdList(ctx)
	return node.WatchList(ctx, dao_node.NodeRegisterPrefix())
}

func (m *Monitor) newPipeWatcherCh(ctx context.Context) (ch chan *watcher.Event, err error) {
	//pipeWatcher, err := pipeline.New(dao_pipe.PipelinePrefix())
	//if err != nil {
	//	return
	//}
	//ch, err = pipeWatcher.WatchEtcdList(ctx)
	return pipeline.WatchList(ctx, dao_pipe.PipelinePrefix())
}

// GetStatus get monitor status
func (m *Monitor) Status() string {
	return m.status
}
