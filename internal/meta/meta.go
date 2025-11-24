package meta

import (
	"reflect"
	"time"
)

// MethodMetadata holds the runtime metadata for a registered RPC method.
// This type is internal so it cannot be instantiated by external packages,
// which allows us to seal the RPCMethod interface.
type MethodMetadata struct {
	Method   string
	Path     string
	Request  reflect.Type
	Response reflect.Type
	CacheTTL time.Duration
}
