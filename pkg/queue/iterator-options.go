package queue

// QueueIteratorOption is a function that modifies the QueueIterator
// configuration. It can be used to set various options for the iterator.
type QueueIteratorOption func(*QueueIterator)

// WithReaderFactor is needed for allowing the users to change the algorithm
// used to read messages from the iterator internal queues. The goal is to
// allow flexibility.
func WithReaderFactory(factory ReaderFactory) QueueIteratorOption {
	return func(q *QueueIterator) {
		q.readerFactory = factory
	}
}
