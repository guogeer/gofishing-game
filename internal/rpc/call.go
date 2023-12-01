package rpc

type rpcHandler func()

type rpcQueue struct {
	q chan rpcHandler
}

var mainQueue = &rpcQueue{
	q: make(chan rpcHandler, 8<<10),
}

func (rq *rpcQueue) RunOnce() {
	for i := 0; i < 3; i++ {
		select {
		case h := <-rq.q:
			h()
		default:
			return
		}
	}
}

func (rq *rpcQueue) Push(h func()) {
	rq.q <- (rpcHandler)(h)
}

func OnResponse(h func()) {
	mainQueue.Push(h)
}

func RunOnce() {
	mainQueue.RunOnce()
}
