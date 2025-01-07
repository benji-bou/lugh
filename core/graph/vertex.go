package graph

import (
	"context"
	"log/slog"
	"reflect"

	"github.com/benji-bou/diwo"
)

type Worker[K any] interface {
	Work(ctx context.Context, input K, yield func(elem K) error) error
}

type Runner[K any] interface {
	Run(ctx context.Context, input <-chan K, yield func(elem K) error) error
}

type Producer[K any] interface {
	Produce(ctx context.Context, yield func(elem K) error) error
}

type Consumer[K any] interface {
	Consume(ctx context.Context, input <-chan K) error
}

type IOWorker[K any] interface {
	Run(ctx SyncContext) <-chan error
	SetInput(input <-chan K)
	Output() <-chan K
}

type IOWorkerVertex[K any] interface {
	IOWorker[K]
	GetName() string
	GetParents() []string
}

type DefaultIOWorkerVertex[K any] struct {
	IOWorker[K]
	name    string
	parents []string
}

func NewDefaultIOWorkerVertex[K any](name string, parents []string, decorated IOWorker[K]) *DefaultIOWorkerVertex[K] {
	return &DefaultIOWorkerVertex[K]{
		IOWorker: decorated,
		name:     name,
		parents:  parents,
	}
}

func (dwv *DefaultIOWorkerVertex[K]) GetName() string {
	return dwv.name
}

func (dwv *DefaultIOWorkerVertex[K]) GetParents() []string {
	return dwv.parents
}

type ioWorker[K any] struct {
	inputC  <-chan K
	outputC chan K
}

func (v *ioWorker[K]) SetInput(input <-chan K) {
	v.inputC = input
}

func (v *ioWorker[K]) Output() <-chan K {
	return v.outputC
}

type syncWorker[K any] struct {
	ioWorker[K]
	worker Worker[K]
}

func NewIOWorkerFromWorker[K any](worker Worker[K]) IOWorker[K] {
	s := &syncWorker[K]{
		ioWorker: ioWorker[K]{outputC: make(chan K)},
		worker:   worker,
	}

	return s
}

func (s *syncWorker[K]) Run(ctx SyncContext) <-chan error {
	ctx.Initializing()
	return diwo.New(func(errC chan<- error) {
		typeName := reflect.TypeOf(s.worker)
		slog.Debug("start", "object", "syncWorker", "function", "Run", "name", typeName)
		defer func() {
			slog.Debug("end, output close", "object", "syncWorker", "function", "Run", "name", typeName)
			close(s.outputC)
		}()
		ctx.Initialized()
		for data := range s.inputC {
			slog.Debug("received data start work", "data", data, "object", "syncWorker", "function", "Run", "name", typeName)
			err := s.worker.Work(ctx, data, func(elem K) error {
				s.outputC <- elem
				return nil
			})
			if err != nil {
				slog.Error("work failed", "error", err, "object", "syncWorker", "function", "Run", "name", typeName)
				errC <- err
				continue
			}
		}
	}, diwo.WithName(reflect.TypeOf(s.worker).String()))
}

type producerWorker[K any] struct {
	ioWorker[K]
	producer Producer[K]
}

func NewIOWorkerFromProducer[K any](producer Producer[K]) IOWorker[K] {
	v := &producerWorker[K]{
		producer: producer,
		ioWorker: ioWorker[K]{outputC: make(chan K)},
	}
	return v
}

func (p *producerWorker[K]) Run(ctx SyncContext) <-chan error {
	ctx.Initializing()
	return diwo.New(func(c chan<- error) {
		defer close(p.outputC)
		slog.Debug("producer initialized")
		ctx.Initialized()
		slog.Debug("producer start producing")
		err := p.producer.Produce(ctx, func(elem K) error {
			p.outputC <- elem
			return nil
		})
		if err != nil {
			c <- err
		}
	})
}

type consumerWorker[K any] struct {
	ioWorker[K]
	consumer Consumer[K]
}

func NewIOWorkerFromConsumer[K any](consumer Consumer[K]) IOWorker[K] {
	v := &consumerWorker[K]{
		consumer: consumer,
		ioWorker: ioWorker[K]{outputC: make(chan K)},
	}
	return v
}

func (c *consumerWorker[K]) Run(ctx SyncContext) <-chan error {
	ctx.Initializing()
	return diwo.New(func(eC chan<- error) {
		close(c.outputC)
		ctx.Initialized()
		err := c.consumer.Consume(ctx, c.inputC)
		if err != nil {
			eC <- err
		}
	})
}

type runWorker[K any] struct {
	ioWorker[K]
	runner Runner[K]
}

func NewIOWorkerFromRunner[K any](runner Runner[K]) IOWorker[K] {
	v := &runWorker[K]{
		runner:   runner,
		ioWorker: ioWorker[K]{outputC: make(chan K)},
	}
	return v
}

func (v *runWorker[K]) Run(ctx SyncContext) <-chan error {
	ctx.Initializing()
	return diwo.New(func(c chan<- error) {
		defer close(v.outputC)
		slog.Debug("start", "object", "runWorker", "function", "Run", "name", reflect.TypeOf(v.runner))
		ctx.Initialized()
		err := v.runner.Run(ctx, v.inputC, func(elem K) error {
			v.outputC <- elem
			return nil
		})
		if err != nil {
			c <- err
		}
	})
}
