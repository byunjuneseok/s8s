// Package broker defines the Broker interface that the application depends on
// for all securities operations (accounts, holdings, market data, orders).
//
// Concrete implementations live in subpackages (e.g. broker/toss) and translate
// provider-specific APIs into the broker-neutral domain models.
package broker
