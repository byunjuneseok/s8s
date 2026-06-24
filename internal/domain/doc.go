// Package domain defines the broker-neutral securities domain models
// (positions, accounts, quotes, orderbooks, orders, money).
//
// Types here are independent of any brokerage API. Broker adapters map their
// provider-specific DTOs into these types, so the rest of the application never
// depends on a particular brokerage. Monetary amounts and quantities are
// modeled as decimals, never floats.
package domain
