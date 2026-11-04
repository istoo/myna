// Package all registers every built-in plugin. Only myna's main imports
// it: core never imports a plugin, and a plugin never imports another.
package all

import (
	_ "github.com/istoo/myna/plugins/budget"
	_ "github.com/istoo/myna/plugins/gateway"
	_ "github.com/istoo/myna/plugins/groups"
	_ "github.com/istoo/myna/plugins/library"
	_ "github.com/istoo/myna/plugins/middleware"
	_ "github.com/istoo/myna/plugins/otel"
	_ "github.com/istoo/myna/plugins/redact"
	_ "github.com/istoo/myna/plugins/rotation"
	_ "github.com/istoo/myna/plugins/routing"
	_ "github.com/istoo/myna/plugins/search"
	_ "github.com/istoo/myna/plugins/sessions"
	_ "github.com/istoo/myna/plugins/tools"
	_ "github.com/istoo/myna/plugins/usage"
	_ "github.com/istoo/myna/plugins/warmups"
)
