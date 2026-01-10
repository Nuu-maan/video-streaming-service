# Phase 6: Admin Dashboard & System Monitoring

## Overview
This PR implements comprehensive admin dashboard functionality with advanced analytics, content moderation, audit logging, and system monitoring capabilities for the video streaming platform.

## Changes Summary

### Domain Models
- **Analytics**: Dashboard statistics, video analytics, user analytics, and time-series data models
- **Content Reports**: Structured reporting system for user-generated content moderation
- **Audit Logs**: Comprehensive logging of all admin actions and system events
- **System Metrics**: CPU, memory, disk, database, Redis, and queue health monitoring

### Database Schema
- **Analytics Tables**: 
  - `video_views` - Track all video view events with timestamps
  - `video_analytics` - Aggregated video performance metrics
  - `user_analytics` - User engagement and activity metrics
- **Moderation Tables**:
  - `content_reports` - User-submitted reports with status tracking
  - `audit_logs` - Complete audit trail of admin actions
- **User Enhancements**:
  - Added `is_banned`, `ban_reason`, `banned_at`, `banned_by` fields for user moderation

### Repository Layer
- **AnalyticsRepository**: Complex SQL queries for dashboard stats, video/user analytics with time-series aggregations
- **ReportRepository**: CRUD operations for content reports with filtering and pagination
- **AuditLogRepository**: Audit log persistence with queryable history

### Service Layer
- **AnalyticsService**: 
  - Dashboard statistics with Redis caching (TTL: 5 minutes)
  - Video analytics with trend analysis
  - User engagement metrics
  - Cache invalidation on data updates
- **ViewTracker**: 
  - Real-time view counting using Redis sorted sets
  - Deduplication using user+video keys
  - Periodic database sync
- **ModerationService**: 
  - Content report management
  - User ban/unban functionality
  - Report review and resolution
- **AuditService**: 
  - Structured logging of all admin actions
  - Contextual metadata capture
- **MonitoringService**: 
  - System health metrics (CPU, memory, disk usage)
  - Database connection pool monitoring
  - Redis health checks
  - Queue depth tracking

### Technical Implementation

**Redis Integration:**
- Analytics caching with 5-minute TTL for dashboard stats
- Real-time view tracking with sorted sets
- Automatic cache invalidation on writes

**Performance Optimizations:**
- Complex SQL queries with proper indexing
- Redis caching layer for frequently accessed data
- Efficient time-series aggregations
- Connection pool monitoring

**Error Handling:**
- Comprehensive error types for analytics, moderation, and audit operations
- Graceful fallbacks when Redis is unavailable
- Proper logging and error propagation

**System Monitoring:**
- Uses gopsutil v3 for cross-platform system metrics
- Health check endpoints ready for integration
- Queue and database pool monitoring

## Database Migrations
- `000005_create_analytics_tables`: Analytics tracking tables with proper indexes
- `000006_create_reports_and_audit`: Content moderation and audit logging tables
- `000007_add_user_ban_fields`: User ban functionality fields

## Dependencies Added
- `github.com/shirou/gopsutil/v3`: System monitoring (CPU, memory, disk metrics)

## Testing Recommendations
1. Run database migrations in test environment
2. Test analytics queries with various date ranges
3. Verify Redis caching behavior and invalidation
4. Test user ban/unban workflows
5. Validate audit log capture for all admin actions
6. Monitor system metrics under load

## Next Steps
- Wire up admin HTTP handlers
- Create admin dashboard UI templates
- Implement authentication middleware for admin routes
- Add role-based access control
- Create admin API documentation
- Deploy and monitor in production

## Breaking Changes
None. All changes are additive and backward compatible.

## Security Considerations
- Audit logs capture all sensitive admin actions
- User ban functionality requires proper authorization
- System metrics endpoints should be admin-only
- Content reports contain user-submitted data that requires sanitization
