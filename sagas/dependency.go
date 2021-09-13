package sagas

import (
	"context"
	"fmt"
	"time"

	"github.com/DoNewsCode/core/contract"
	"github.com/DoNewsCode/core/di"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/oklog/run"
)

/*
Providers returns a set of dependency providers.
	Depends On:
		contract.ConfigAccessor
		log.Logger
		Store   `optional:"true"`
		[]*Step `group:"saga"`
	Provide:
		*Registry
		SagaEndpoints
*/
func Providers(opts ...ProvidersOptionFunc) di.Deps {
	option := providersOption{
		storeConstructor: newDefaultStore,
	}
	for _, f := range opts {
		f(&option)
	}
	return []interface{}{provide(&option), provideConfig}
}

// in is the injection parameter for saga module.
type in struct {
	di.In

	Conf      contract.ConfigAccessor
	Logger    log.Logger
	Steps     []*Step `group:"saga"`
	Populator contract.DIPopulator
}

type recoverInterval time.Duration

// SagaEndpoints is a collection of all registered endpoint in the saga registry
type SagaEndpoints map[string]endpoint.Endpoint

type out struct {
	di.Out

	Registry      *Registry
	Interval      recoverInterval
	SagaEndpoints SagaEndpoints
}

// provide creates a new saga module.
func provide(option *providersOption) func(in in) (out, error) {
	if option.storeConstructor == nil {
		option.storeConstructor = newDefaultStore
	}
	return func(in in) (out, error) {
		var (
			store Store
			err   error
		)
		store = option.store
		if option.store == nil {
			store, err = option.storeConstructor(StoreArgs{Populator: in.Populator})
			if err != nil {
				return out{}, fmt.Errorf("fails to construct Store: %w", err)
			}
		}
		var conf configuration
		err = in.Conf.Unmarshal("sagas", &conf)
		if err != nil {
			level.Warn(in.Logger).Log("err", err)
		}
		timeout := conf.getSagaTimeout().Duration
		recoverVal := conf.getRecoverInterval().Duration

		registry := NewRegistry(
			store,
			WithLogger(in.Logger),
			WithTimeout(timeout),
		)
		eps := make(SagaEndpoints)

		for i := range in.Steps {
			eps[in.Steps[i].Name] = registry.AddStep(in.Steps[i])
		}

		return out{
			Registry:      registry,
			Interval:      recoverInterval(recoverVal),
			SagaEndpoints: eps,
		}, nil
	}

}

// ProvideRunGroup implements the RunProvider.
func (m out) ProvideRunGroup(group *run.Group) {
	ctx, cancel := context.WithCancel(context.Background())
	ticker := time.NewTicker(time.Duration(m.Interval))
	group.Add(func() error {
		m.Registry.Recover(ctx)
		for {
			select {
			case <-ticker.C:
				m.Registry.Recover(ctx)
			case <-ctx.Done():
				return nil
			}
		}
	}, func(err error) {
		cancel()
		ticker.Stop()
	})
}

func (m out) ModuleSentinel() {}

func (m out) Module() interface{} { return m }

func newDefaultStore(args StoreArgs) (Store, error) {

	var storeHolder struct {
		di.In
		Store Store `optional:"true"`
	}
	if err := args.Populator.Populate(&storeHolder); err != nil {
		return nil, err
	}
	if storeHolder.Store == nil {
		storeHolder.Store = NewInProcessStore()
	}
	return storeHolder.Store, nil

}
