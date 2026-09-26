package outbox

import (
	"github.com/tertua/invoiceman/pkg/logger"
)

// recordErr logs best-effort bookkeeping write failures. Callers already
// branched on the primary outcome, so a failed counter or status touch must
// not fail the tick — but it must not vanish silently either.
func recordErr(op string, err error, args ...any) {
	if err != nil {
		logger.L().Warn("outbox bookkeeping write failed", append([]any{"op", op, "err", err}, args...)...)
	}
}
