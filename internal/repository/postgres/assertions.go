package postgres

import (
	"github.com/Nuu-maan/video-streaming-service/internal/service"
)

// The service layer declares narrow, consumer-side interfaces for the slice of
// each store it actually uses. Nothing in the wiring path proves a concrete
// repository still satisfies them, so the two sides silently drifted apart
// before: every one of these interfaces named methods that no implementation
// had, and it compiled only because no service was ever constructed.
//
// These assertions cost nothing at runtime and turn that drift back into a
// build failure.
//
// They live in this package, not in service, because the dependency only runs
// one way: postgres imports service, service imports repository. Asserting from
// the service side would require service to import postgres and close the cycle.
var (
	_ service.ModerationRepository  = (*ReportRepository)(nil)
	_ service.VideoRepository       = (*PostgresVideoRepository)(nil)
	_ service.UserRepository        = (*UserRepository)(nil)
	_ service.AuditLogRepository    = (*AuditLogRepository)(nil)
	_ service.AnalyticsRepository   = (*AnalyticsRepository)(nil)
	_ service.ViewTrackerRepository = (*AnalyticsRepository)(nil)
)
