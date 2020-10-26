package utils

import "fmt"

//QueueWithCapacity for represent FIFO queue with defined capacity
type QueueWithCapacity struct {
	channel chan interface{}
}

//InitQueueWithCapacity initialises queue
func InitQueueWithCapacity(capacity int) *QueueWithCapacity {
	return &QueueWithCapacity{channel: make(chan interface{}, capacity)}
}

//Enqueue add element into queue
// if queue is full, it drop last element
func (queue *QueueWithCapacity) Enqueue(item interface{}) error {
	var ok bool
	select {
	case queue.channel <- item:
		ok = true
	default:
		ok = false
	}
	if !ok {
		<-queue.channel
		return queue.Enqueue(item)
	}
	return nil
}

//Dequeue get last element from queue
func (queue *QueueWithCapacity) Dequeue() (item interface{}, err error) {
	var ok bool
	select {
	case item = <-queue.channel:
		ok = true
	default:
		ok = false
	}
	if !ok {
		return nil, fmt.Errorf("no elements in queue")
	}
	return item, nil
}
