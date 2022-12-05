package balancer

import (
	"log"
	"net/http"
	"strconv"
	"sync/atomic"
)

type balancer struct {
	handlers []http.Handler
	counter  int32
}

func New(handlers ...http.Handler) *balancer {
	return &balancer{
		handlers: handlers,
	}
}

func (b *balancer) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	counter := atomic.AddInt32(&b.counter, 1)
	if counter < 0 {
		counter = -counter
	}
	index := int(counter-1) % len(b.handlers)
	log.Println("using backend #" + strconv.Itoa(index))
	b.handlers[index].ServeHTTP(writer, request)
}
