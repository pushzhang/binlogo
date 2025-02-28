package node

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/raft"
	"github.com/jin06/binlogo/app/server/node/election"
	"github.com/jin06/binlogo/app/server/node/manager"
	"github.com/jin06/binlogo/app/server/node/manager/manager_event"
	"github.com/jin06/binlogo/app/server/node/manager/manager_pipe"
	"github.com/jin06/binlogo/app/server/node/manager/manager_status"
	"github.com/jin06/binlogo/app/server/node/monitor"
	"github.com/jin06/binlogo/app/server/node/scheduler"
	"github.com/jin06/binlogo/pkg/node/role"
	"github.com/jin06/binlogo/pkg/register"
	"github.com/jin06/binlogo/pkg/store/dao/dao_node"
	"github.com/jin06/binlogo/pkg/store/model/node"
	"github.com/sirupsen/logrus"
)

// Node represents a node instance
// Running pipeline, reporting status, etc.
// if it becomes the master node, it will run tasks such as scheduling pipeline, monitoring, event management, etc
type Node struct {
	Mode     *NodeMode
	Options  *Options
	Name     string
	Register *register.Register
	//election       *election.Election
	electionManager *election.Manager
	Scheduler       *scheduler.Scheduler
	StatusManager   *manager_status.Manager
	monitor         *monitor.Monitor
	leaderRunMutex  sync.Mutex
	pipeManager     *manager_pipe.Manager
	eventManager    *manager_event.Manager
	raft            *raft.Raft
}

type NodeMode byte

const (
	MODE_CLUSTER NodeMode = 1
	MODE_SINGLE  NodeMode = 2
)

// NodeMode todo
func (n NodeMode) String() string {
	switch n {
	case MODE_CLUSTER:
		{
			return "cluster"
		}
	case MODE_SINGLE:
		{
			return "single"
		}
	}
	return ""
}

// New return a new node
func New(opts ...Option) (node *Node, err error) {
	options := &Options{}
	node = &Node{
		Options: options,
	}
	for _, v := range opts {
		v(options)
	}
	err = node.init()
	return
}

func (n *Node) init() (err error) {
	logrus.Debug("---->", n.Options.Node)
	return
}

func (n *Node) refreshNode() (err error) {
	var newest *node.Node
	newest, err = dao_node.GetNode(n.Options.Node.Name)
	if err != nil {
		return
	}
	if newest == nil {
		err = errors.New("unexpected, node is null")
		return
	}
	n.Options.Node = newest
	return
}

// Run start working
func (n *Node) Run(ctx context.Context) (err error) {
	myCtx, cancel := context.WithCancel(ctx)
	defer func() {
		if re := recover(); re != nil {
			err = errors.New(fmt.Sprintf("panic %v", re))
		}
		cancel()
	}()
	err = n.refreshNode()
	if err != nil {
		return
	}
	nodeCtx := n._mustRun(myCtx)
	n._leaderRun(myCtx)
	select {
	case <-ctx.Done():
		{
			return
		}
	case <-nodeCtx.Done():
		{
			return
		}
	}
}

// Role returns current role
func (n *Node) Role() role.Role {
	return n.electionManager.Role()
	//return n.election.Role()
}

// _leaderRun run when node is leader
func (n *Node) _leaderRun(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				{
					return
				}
			case r := <-n.electionManager.RoleCh():
				{
					n.leaderRun(ctx, r)
				}
			case <-time.Tick(time.Second * 3):
				{
					n.leaderRun(ctx, n.Role())
				}
			}
		}
	}()
}

func (n *Node) leaderRun(ctx context.Context, r role.Role) {
	//if r == "" {
	//	r = n.Role()
	//}
	switch r {
	case role.LEADER:
		{
			var err error
			if n.Scheduler == nil || n.Scheduler.Status() == scheduler.SCHEDULER_STOP {
				n.Scheduler = scheduler.New()
				err = n.Scheduler.Run(ctx)
				if err != nil {
					return
				}
			}
			if n.monitor == nil || n.monitor.Status() == monitor.STATUS_STOP {
				n.monitor, err = monitor.NewMonitor()
				if err != nil {
					return
				}
				err = n.monitor.Run(ctx)
				if err != nil {
					return
				}
			}
			if n.eventManager == nil || n.eventManager.Status == manager.STOP {
				n.eventManager = manager_event.New()
				err = n.eventManager.Run(ctx)
				if err != nil {
					return
				}
			}
		}
	default:
		{
			if n.Scheduler != nil {
				n.Scheduler.Stop()
			}
			if n.monitor != nil {
				n.monitor.Stop(ctx)
			}
			if n.eventManager != nil {
				n.eventManager.Stop()
			}
		}
	}
}

// _mustRun run manager that every node must run
// such as election, node register, node status
func (n *Node) _mustRun(ctx context.Context) (resCtx context.Context) {
	resCtx, cancel := context.WithCancel(ctx)
	n.Register = register.New(
		register.WithTTL(5),
		register.WithKey(dao_node.NodeRegisterPrefix()+"/"+n.Options.Node.Name),
		register.WithData(n.Options.Node),
	)
	n.Register.Run(resCtx)
	//n.election = election.New(
	//	election.OptionNode(n.Options.Node),
	//	election.OptionTTL(5),
	//)
	//n.election.Run(ctx)
	n.electionManager = election.NewManager(n.Options.Node)
	n.electionManager.Run(resCtx)
	n.pipeManager = manager_pipe.New(n.Options.Node)
	n.pipeManager.Run(resCtx)
	n.StatusManager = manager_status.NewManager(n.Options.Node)
	n.StatusManager.Run(resCtx)
	go func() {
		defer cancel()
		select {
		case <-ctx.Done():
			{
				return
			}
		case <-n.Register.Context().Done():
			{
				return
			}
			//case <-n.election.Context().Done():
		case <-n.electionManager.Context().Done():
			{
				return
			}
		case <-n.pipeManager.Context().Done():
			{
				return
			}
		case <-n.StatusManager.Context().Done():
			{
				return
			}
		}
	}()
	return
}
