package http

import "sync/atomic"

// upgradeProtocols counts RegisterUpgradeProtocol calls in this process.
// The count only increases: importing an upgrade package opts the process in.
var upgradeProtocols atomic.Int32

// RegisterUpgradeProtocol records that an upgrade handler is linked into this
// process. Application listen turns Server.AllowUpgrade on when the count is
// non-zero, unless ListenOptions or HTTP_ALLOW_UPGRADE set an explicit value.
// packages/websocket registers through the addon registry; this hook is for
// other upgrade protocols and tests.
func RegisterUpgradeProtocol() {
	upgradeProtocols.Add(1)
}

// UpgradeProtocolRegistered reports whether RegisterUpgradeProtocol ran.
func UpgradeProtocolRegistered() bool {
	return upgradeProtocols.Load() > 0
}
