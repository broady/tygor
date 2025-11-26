package tygor

import "reflect"

// ExportedRoute contains metadata about a registered route for code generation.
type ExportedRoute struct {
	Name       string
	Request    reflect.Type
	Response   reflect.Type
	HTTPMethod string
}

// ExportRoutes returns all registered routes for code generation purposes.
// This is used by the generator package.
func (r *App) ExportRoutes() map[string]ExportedRoute {
	r.mu.RLock()
	defer r.mu.RUnlock()

	exported := make(map[string]ExportedRoute)
	for k, v := range r.routes {
		meta := v.Metadata()
		exported[k] = ExportedRoute{
			Name:       k,
			Request:    meta.Request,
			Response:   meta.Response,
			HTTPMethod: meta.HTTPMethod,
		}
	}
	return exported
}
