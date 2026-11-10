// Package all registers every built-in plugin. Only owl's main imports
// it: core never imports a plugin, and a plugin never imports another.
package all

import (
	_ "github.com/istoo/owl/plugins/budget"
	_ "github.com/istoo/owl/plugins/gateway"
	_ "github.com/istoo/owl/plugins/groups"
	_ "github.com/istoo/owl/plugins/library"
	_ "github.com/istoo/owl/plugins/middleware"
	_ "github.com/istoo/owl/plugins/otel"
	_ "github.com/istoo/owl/plugins/redact"
	_ "github.com/istoo/owl/plugins/rotation"
	_ "github.com/istoo/owl/plugins/routing"
	_ "github.com/istoo/owl/plugins/search"
	_ "github.com/istoo/owl/plugins/sessions"
	_ "github.com/istoo/owl/plugins/tools"
	_ "github.com/istoo/owl/plugins/usage"
	_ "github.com/istoo/owl/plugins/warmups"
)
