package sagas

import "github.com/DoNewsCode/core/contract"

type providersOption struct {
	store            Store
	storeConstructor func(args StoreArgs) (Store, error)
}

// ProvidersOptionFunc is the type of functional providersOption for Providers. Use this type to change how Providers work.
type ProvidersOptionFunc func(options *providersOption)

// WithStore instructs the Providers to accept a queue driver
// different from the default one. This option supersedes the
// WithStoreConstructor option.
func WithStore(store Store) ProvidersOptionFunc {
	return func(options *providersOption) {
		options.store = store
	}
}

// WithStoreConstructor instructs the Providers to accept an alternative Store for dtx.
// If the WithDriver option is set, this option becomes an no-op.
func WithStoreConstructor(f func(args StoreArgs) (Store, error)) ProvidersOptionFunc {
	return func(options *providersOption) {
		options.storeConstructor = f
	}
}

// StoreArgs are arguments to construct the Store. See WithStoreConstructor.
type StoreArgs struct {
	Populator contract.DIPopulator
}
