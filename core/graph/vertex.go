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
type WorkerFunc[K any] func(ctx context.Context, input K, yield func(elem K) error) error

func (wf WorkerFunc[K]) Work(ctx context.Context, input K, yield func(elem K) error) error {
	return wf(ctx, input, yield)
}

type RunnerFunc[K any] func(ctx context.Context, input <-chan K, yield func(elem K, err error) error) error

func (rf RunnerFunc[K]) Run(ctx context.Context, input <-chan K, yield func(elem K, err error) error) error {
	return rf(ctx, input, yield)
}

type Runner[K any] interface {
	Run(ctx context.Context, input <-chan K, yield func(elem K, err error) error) error
}

type Producer[K any] interface {
	Produce(ctx context.Context, yield func(elem K) error) error
}
type ProducerFunc[K any] func(ctx context.Context, yield func(elem K) error) error

func (pf ProducerFunc[K]) Produce(ctx context.Context, yield func(elem K) error) error {
	return pf(ctx, yield)
}

type Consumer[K any] interface {
	Consume(ctx context.Context, input K) error
}
type ConsumerFunc[K any] func(ctx context.Context, input K) error

func (cf ConsumerFunc[K]) Consume(ctx context.Context, input K) error {
	return cf(ctx, input)
}

type IOWorker[K any] interface {
	Run(ctx SyncContext) <-chan error
	SetInput(input <-chan K)
	Output() <-chan K
}

type VertexSelfDescribe[K comparable] interface {
	GetName() K
	GetParents() []K
}
type IOWorkerVertex[K any] interface {
	IOWorker[K]
	VertexSelfDescribe[string]
}

type DefaultIOWorkerVertex[K any] struct {
	IOWorker[K]
	name    string
	parents []string
}

func NewDefaultIOWorkerVertex[K any](name string, parents []string, decorated IOWorker[K]) *DefaultIOWorkerVertex[K] {
	return &DefaultIOWorkerVertex[K]{
		IOWorker: NewBroadcasterIOWorker(decorated),
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
	slog.Debug("Setting input for worker")
	v.inputC = input
}

func (v *ioWorker[K]) Output() <-chan K {
	if v.outputC == nil {
		v.outputC = make(chan K)
	}
	return v.outputC
}

func (v *ioWorker[K]) SendOutput(output K) {
	if v.outputC != nil {
		v.outputC <- output
	}
}

func (v *ioWorker[K]) Close() {
	if v.outputC != nil {
		close(v.outputC)
	}
}

type syncWorker[K any] struct {
	ioWorker[K]
	worker Worker[K]
}

func NewIOWorkerFromWorker[K any](worker Worker[K]) IOWorker[K] {
	s := &syncWorker[K]{
		ioWorker: ioWorker[K]{},
		worker:   worker,
	}

	return s
}

func (s *syncWorker[K]) Run(ctx SyncContext) <-chan error {
	ctx.Initializing()
	slog.Debug("Initializing Worker started", "worker", reflect.TypeOf(s.worker).String())
	return diwo.New(func(errC chan<- error) {
		typeWorker := reflect.TypeOf(s.worker).String()
		workerCtx, workerCancel := context.WithCancel(ctx)
		defer func() {
			defer workerCancel()
			s.Close()
			slog.Debug("Worker exited Closed output chan", "worker", typeWorker)
		}()
		slog.Debug("Worker initialized wait for sync", "worker", reflect.TypeOf(s.worker).String())
		ctx.Initialized()
		slog.Debug("Worker initialized and sync", "worker", reflect.TypeOf(s.worker).String())
		for {
			select {
			case <-ctx.Done():
				return
			case data, ok := <-s.inputC:
				slog.Debug("Worker Received", "worker", typeWorker)
				if !ok {
					return
				}
				err := s.worker.Work(workerCtx, data, func(elem K) error {
					if workerCtx.Err() != nil {
						return workerCtx.Err()
					}
					slog.Debug("Worker Yielding to output chan", "worker", typeWorker)
					s.SendOutput(elem)
					slog.Debug("Worker Yielded to output chan", "worker", typeWorker)
					return nil
				})
				if err != nil {
					errC <- err
				}
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
		ioWorker: ioWorker[K]{},
	}
	return v
}

func (p *producerWorker[K]) Run(ctx SyncContext) <-chan error {
	ctx.Initializing()
	slog.Debug("Producer Initialization started", "producer", reflect.TypeOf(p.producer).String())
	return diwo.New(func(c chan<- error) {
		typeProducer := reflect.TypeOf(p.producer).String()
		producerCtx, producerCancel := context.WithCancel(ctx)
		defer func() {
			defer producerCancel()
			p.Close()
			slog.Debug("Producer exited Closed output chan", "producer", typeProducer)
		}()
		slog.Debug("Producer initialized wait for sync", "producer", typeProducer)
		ctx.Initialized()
		slog.Debug("Producer initialized and sync", "producer", typeProducer)
		err := p.producer.Produce(ctx, func(elem K) error {
			if producerCtx.Err() != nil {
				return producerCtx.Err()
			}
			slog.Debug("Producer Yielding to output chan", "producer", typeProducer)
			p.SendOutput(elem)
			slog.Debug("Producer Yielded to output chan", "producer", typeProducer)
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
		ioWorker: ioWorker[K]{},
	}
	return v
}

func (c *consumerWorker[K]) Run(ctx SyncContext) <-chan error {
	ctx.Initializing()
	slog.Debug("Consummer Initialization started", "consummer", reflect.TypeOf(c.consumer).String())

	return diwo.New(func(eC chan<- error) {
		typeConsumer := reflect.TypeOf(c.consumer).String()

		defer func() {
			c.Close()
			slog.Debug("Consumer exited Closed output chan", "worker", typeConsumer)
		}()
		slog.Debug("Consumer initialized wait for sync", "consumer", typeConsumer)
		ctx.Initialized()
		slog.Debug("Consumer initialized and sync", "consumer", typeConsumer)
		for {
			select {
			case <-ctx.Done():
				return
			case input, ok := <-c.inputC:
				slog.Debug("Consumer Received", "consumer", typeConsumer, "elem", input, "isClosed", !ok)
				if !ok {
					return
				}
				err := c.consumer.Consume(ctx, input)
				slog.Debug("Consumer consummed", "consumer", typeConsumer, "elem", input)
				if err != nil {
					eC <- err
				}
			}
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
		ioWorker: ioWorker[K]{},
	}
	return v
}

func (v *runWorker[K]) Run(ctx SyncContext) <-chan error {
	ctx.Initializing()
	slog.Debug("Initializing Runner started", "runner", reflect.TypeOf(v.runner).String())

	return diwo.New(func(c chan<- error) {
		typeRunner := reflect.TypeOf(v.runner).String()
		runnerCtx, runnerCancel := context.WithCancel(ctx)
		defer func() {
			defer runnerCancel()
			v.Close()
			slog.Debug("Runner exited Closed output chan", "runner", typeRunner)
		}()
		slog.Debug("Runner initialized wait for sync", "runner", typeRunner)
		ctx.Initialized()
		slog.Debug("Runner initialized and sync", "runner", typeRunner)
		err := v.runner.Run(ctx, v.inputC, func(elem K, err error) error {
			if runnerCtx.Err() != nil {
				return runnerCtx.Err()
			}
			if err != nil {
				slog.Debug("Runner Yielding to error chan", "runner", typeRunner, "error", err)
				c <- err
				slog.Debug("Runner Yielded to error chan", "runner", typeRunner, "error", err)
				return nil
			}
			slog.Debug("Runner Yielding to output chan", "runner", typeRunner)
			v.SendOutput(elem)
			slog.Debug("Runner Yielded to output chan", "runner", typeRunner)
			return nil
		})
		if err != nil {
			c <- err
		}
	})
}

type BroadcasterIOWorker[K any] struct {
	IOWorker[K]
	broker *diwo.Broker[K]
}

func NewBroadcasterIOWorker[K any](worker IOWorker[K]) *BroadcasterIOWorker[K] {
	return &BroadcasterIOWorker[K]{
		IOWorker: worker,
	}
}

func (v *BroadcasterIOWorker[K]) Output() <-chan K {
	if v.broker == nil {
		v.broker = diwo.NewBroker(v.IOWorker.Output())
	}
	return v.broker.Subscribe()
}
