package meta

import (
	"reflect"
)

// MethodMetadata holds the runtime metadata for a registered RPC method.
// This type is internal so it cannot be instantiated by external packages,
// which allows us to seal the RPCMethod interface.
type MethodMetadata struct {
	HTTPMethod string
	Request    reflect.Type
	Response   reflect.Type
}
