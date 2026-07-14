/**
 * Response and request types for the video streaming service API.
 *
 * Field names mirror the server's JSON tags exactly (snake_case) so a payload
 * can be assigned to these interfaces without any mapping layer.
 * Canonical API prefix: /api/v1
 */

// ---------------------------------------------------------------------------
// Enums / unions
// ---------------------------------------------------------------------------

export type Role = "guest" | "user" | "premium" | "moderator" | "admin";

export type VideoStatus = "uploading" | "processing" | "ready" | "failed";

export type VideoVisibility = "public" | "private" | "unlisted";

export type VideoQuality = "360p" | "480p" | "720p" | "1080p";

export type TrendingWindow = "24h" | "7d" | "30d";

export type SearchSort = "relevance" | "newest" | "views" | "likes";

export type NotificationType =
  | "new_video"
  | "comment"
  | "reply"
  | "like"
  | "subscriber"
  | "mention";

export type ReportType =
  | "spam"
  | "harassment"
  | "hate_speech"
  | "violence"
  | "copyright"
  | "nudity"
  | "misinformation"
  | "other";

export type ReportStatus = "pending" | "reviewing" | "resolved" | "dismissed";

export type ReportReviewAction =
  | "delete_video"
  | "ban_user"
  | "warn_user"
  | "dismiss";

export type TimeSeriesInterval = "hour" | "day" | "week" | "month";

/** Error codes observed in the handlers. The server may add more over time. */
export type ApiErrorCode =
  | "VALIDATION_ERROR"
  | "BAD_REQUEST"
  | "UNAUTHORIZED"
  | "FORBIDDEN"
  | "NOT_FOUND"
  | "INTERNAL_ERROR"
  | "ALREADY_EXISTS"
  | "USER_BANNED"
  | "DUPLICATE_REPORT"
  | "EMAIL_ALREADY_VERIFIED"
  | "INVALID_TOKEN"
  | "INVALID_CURRENT_PASSWORD"
  | "FILE_TOO_LARGE"
  | "INVALID_FORMAT"
  | "HLS_NOT_READY"
  | "PLAYLIST_NOT_FOUND"
  | "SEGMENT_NOT_FOUND"
  | "VIDEO_NOT_READY"
  | "FILE_NOT_FOUND"
  | "ALREADY_IN_PLAYLIST"
  /** 503: token-revocation store unreachable, server failing closed. Retry; do NOT discard tokens. */
  | "AUTH_UNAVAILABLE"
  | (string & {});

// ---------------------------------------------------------------------------
// Envelopes
// ---------------------------------------------------------------------------

export interface ErrorDetail {
  code: ApiErrorCode;
  message: string;
}

export interface SuccessEnvelope<T> {
  success: true;
  data: T;
}

export interface ErrorEnvelope {
  success: false;
  error: ErrorDetail;
}

export interface PaginationMeta {
  total: number;
  page: number;
  limit: number;
  total_pages: number;
  has_next: boolean;
  has_previous: boolean;
}

export interface ListEnvelope<T> {
  success: true;
  data: T[];
  pagination: PaginationMeta;
}

/** What paginated client methods return: the unwrapped list plus its metadata. */
export interface Page<T> {
  items: T[];
  pagination: PaginationMeta;
}

// ---------------------------------------------------------------------------
// Auth
// ---------------------------------------------------------------------------

export interface User {
  id: string;
  username: string;
  email: string;
  full_name?: string | null;
  bio?: string | null;
  avatar_url?: string | null;
  role: Role;
  email_verified: boolean;
  last_login_at?: string | null;
  oauth_provider?: string | null;
  oauth_avatar_url?: string | null;
  is_banned: boolean;
  ban_reason?: string | null;
  ban_expiry?: string | null;
  banned_at?: string | null;
  created_at: string;
  updated_at: string;
}

export interface TokenPair {
  access_token: string;
  refresh_token: string;
  token_type: "Bearer";
  /** Access-token lifetime in seconds (900). */
  expires_in: number;
  /** Refresh-token lifetime in seconds (604800). */
  refresh_expires_in: number;
  user: User;
}

export interface RegisterRequest {
  username: string;
  email: string;
  password: string;
}

export interface LoginRequest {
  /** A username OR an email — the server accepts either. */
  identifier: string;
  password: string;
}

// ---------------------------------------------------------------------------
// Videos
// ---------------------------------------------------------------------------

export interface Video {
  id: string;
  user_id?: string | null;
  title: string;
  description: string;
  filename: string;
  file_size: number;
  /** Seconds. */
  duration: number;
  status: VideoStatus;
  visibility: VideoVisibility;
  mime_type: string;
  original_resolution?: string;
  /** 0-100. */
  transcoding_progress: number;
  available_qualities: string[];
  hls_ready: boolean;
  streaming_protocol?: string;
  category?: string;
  tags?: string[];
  language?: string;
  view_count: number;
  like_count: number;
  comment_count: number;
  created_at: string;
  updated_at: string;
  processed_at?: string | null;
  /**
   * Absolute PATH (not a full URL), e.g. /api/v1/videos/{id}/thumbnail.
   * Resolve against your API origin — see client.media.resolve().
   */
  thumbnail_url?: string;
  /**
   * Absolute PATH (not a full URL), e.g. /api/v1/videos/{id}/hls/master.m3u8.
   * Present once hls_ready is true.
   */
  hls_url?: string;
}

export interface VideoStatusInfo {
  id: string;
  status: VideoStatus;
  /** 0-100. */
  progress: number;
  available_qualities: string[];
  message: string;
  /** Server-side storage key; only present once a thumbnail exists. */
  thumbnail?: string;
}

export interface ListVideosParams {
  page?: number;
  limit?: number;
  search?: string;
  status?: VideoStatus;
  /** true = list the caller's own videos (all visibilities); requires auth. */
  mine?: boolean;
}

export interface UploadVideoParams {
  /** The video file. In the browser: a File from an <input type="file">. */
  file: Blob;
  /** Filename hint used for the multipart part (defaults to File.name or "video"). */
  filename?: string;
  /** 1-255 chars; the server rejects an empty title with a 400. */
  title: string;
  description?: string;
  visibility?: VideoVisibility;
  /**
   * Upload progress (0-1). Only fires in environments with XMLHttpRequest
   * (i.e. the browser); silently unused under server-side fetch.
   */
  onProgress?: (fraction: number, loadedBytes: number, totalBytes: number) => void;
  signal?: AbortSignal;
}

// ---------------------------------------------------------------------------
// Social
// ---------------------------------------------------------------------------

export interface Like {
  id: string;
  user_id: string;
  video_id: string;
  is_like: boolean;
  created_at: string;
}

export interface Comment {
  id: string;
  video_id: string;
  user_id: string;
  parent_id?: string | null;
  content: string;
  like_count: number;
  reply_count: number;
  pinned: boolean;
  edited_at?: string | null;
  created_at: string;
  updated_at: string;
  deleted_at?: string | null;
  username?: string;
  avatar_url?: string;
}

export interface SubscriptionEntry {
  user_id: string;
  username: string;
  avatar_url?: string;
  subscriber_count: number;
  notify_uploads: boolean;
  subscribed_at: string;
}

export interface Playlist {
  id: string;
  user_id: string;
  title: string;
  description: string;
  visibility: VideoVisibility;
  video_count: number;
  created_at: string;
  updated_at: string;
}

export interface PlaylistVideo {
  id: string;
  playlist_id: string;
  video_id: string;
  position: number;
  added_at: string;
}

export interface PlaylistItem {
  position: number;
  added_at: string;
  video: Video | null;
}

export interface WatchLaterItem {
  added_at: string;
  video: Video | null;
}

export interface WatchHistory {
  id: string;
  user_id: string;
  video_id: string;
  watched_at: string;
  watch_duration: number;
  completed: boolean;
  last_position: number;
}

export interface Notification {
  id: string;
  user_id: string;
  type: NotificationType;
  title: string;
  message: string;
  action_url?: string | null;
  actor_id?: string | null;
  video_id?: string | null;
  comment_id?: string | null;
  read: boolean;
  created_at: string;
}

export interface CreatePlaylistRequest {
  title: string;
  description?: string;
  visibility?: VideoVisibility;
}

export interface UpdatePlaylistRequest {
  title?: string;
  description?: string;
  visibility?: VideoVisibility;
}

export interface CreateCommentRequest {
  /** 1-10000 chars. */
  content: string;
  /** Reply to another comment on the same video. */
  parent_id?: string;
}

export interface CreateReportRequest {
  report_type: ReportType;
  reason: string;
  description?: string;
  /** At least one of video_id / user_id / comment_id is required. */
  video_id?: string;
  user_id?: string;
  comment_id?: string;
}

export interface ContentReport {
  id: string;
  video_id?: string | null;
  user_id?: string | null;
  comment_id?: string | null;
  reporter_id: string;
  report_type: ReportType;
  reason: string;
  description?: string;
  status: ReportStatus;
  reviewed_by?: string | null;
  reviewed_at?: string | null;
  action?: string | null;
  created_at: string;
  updated_at: string;
}

// ---------------------------------------------------------------------------
// Discovery / search
// ---------------------------------------------------------------------------

export interface VideoSearchItem {
  video_id: string;
  title: string;
  description: string;
  thumbnail_url: string;
  duration: number;
  views: number;
  created_at: string;
  username: string;
  user_id: string;
  user_avatar_url: string;
  user_verified: boolean;
  relevance: number;
  snippet: string;
}

export interface CategoryCount {
  category: string;
  video_count: number;
}

export interface SearchParams {
  /** Required search query. */
  q: string;
  sort?: SearchSort;
  category?: string;
  language?: string;
  /** Comma-separated tag list, e.g. "go,tutorial". */
  tags?: string;
  /** Seconds, >= 0. */
  min_duration?: number;
  /** Seconds, >= 0. */
  max_duration?: number;
  page?: number;
  limit?: number;
}

// ---------------------------------------------------------------------------
// Engagement
// ---------------------------------------------------------------------------

export interface RecordViewRequest {
  quality?: string;
  source?: string;
  /**
   * Required for anonymous (unauthenticated) callers — any stable random id
   * you keep per browser session. Authenticated calls are deduped by user id.
   */
  session_id?: string;
}

export interface RecordViewResult {
  /** true = counted (HTTP 201); false = deduplicated repeat (HTTP 200). */
  counted: boolean;
}

export interface SaveProgressRequest {
  /** Seconds; must be within duration. */
  position: number;
  /** Seconds. */
  duration: number;
  completed: boolean;
}

// ---------------------------------------------------------------------------
// Admin: analytics & monitoring
// ---------------------------------------------------------------------------

export interface DashboardStats {
  total_users: number;
  new_users_today: number;
  new_users_this_week: number;
  active_users_24h: number;
  total_videos: number;
  videos_today: number;
  videos_this_week: number;
  processing_videos: number;
  failed_videos: number;
  total_views: number;
  views_today: number;
  views_this_week: number;
  total_storage_bytes: number;
  storage_used_gb: number;
  queued_jobs: number;
  active_workers: number;
  premium_users: number;
  monthly_revenue: number;
  last_updated: string;
}

export interface CountryStats {
  country: string;
  views: number;
}

export interface VideoAnalytics {
  video_id: string;
  title: string;
  user_id: string;
  username: string;
  total_views: number;
  unique_viewers: number;
  likes: number;
  dislikes: number;
  comments: number;
  shares: number;
  total_watch_time: number;
  avg_watch_time: number;
  avg_watch_percent: number;
  views_by_quality: Record<string, number>;
  avg_buffer_time: number;
  playback_errors: number;
  source_direct: number;
  source_search: number;
  source_embed: number;
  source_social: number;
  top_countries: CountryStats[];
  device_mobile: number;
  device_desktop: number;
  device_tablet: number;
  created_at: string;
  last_viewed: string;
}

export interface RealtimeMetrics {
  active_viewers: number;
  uploads_last_hour: number;
  views_last_hour: number;
  current_cpu: number;
  current_memory: number;
  queued_jobs: number;
  processing_jobs: number;
  timestamp: string;
}

export interface DataPoint {
  timestamp: string;
  value: number;
}

export interface TimeSeriesData {
  label: string;
  datapoints: DataPoint[];
}

export interface QueueStats {
  active: number;
  pending: number;
  scheduled: number;
  retry: number;
  archived: number;
  completed: number;
  aggregating: number;
  processed: number;
  failed: number;
  paused: boolean;
  size: number;
}

export interface WorkerInfo {
  host: string;
  pid: number;
  server_id: string;
  concurrency: number;
  queues: Record<string, number>;
  started: string;
  active_tasks: number;
}

export interface WorkersResponse {
  workers: WorkerInfo[];
  count: number;
}

export interface SystemMetrics {
  cpu_percent: number;
  memory_total: number;
  memory_used: number;
  memory_percent: number;
  disk_total: number;
  disk_used: number;
  disk_percent: number;
  goroutines: number;
  /** Go time.Duration — integer nanoseconds. */
  uptime: number;
  timestamp: string;
}

export interface QueueMetrics {
  pending_jobs: number;
  active_jobs: number;
  failed_jobs: number;
  retry_queue: number;
  archived_jobs: number;
  processed_last: number;
  timestamp: string;
}

export interface DatabaseMetrics {
  active_connections: number;
  idle_connections: number;
  max_connections: number;
  slow_queries: number;
  total_queries: number;
  table_sizes: Record<string, number>;
  timestamp: string;
}

export interface RedisMetrics {
  memory_used: number;
  memory_peak: number;
  total_keys: number;
  hits: number;
  misses: number;
  hit_rate: number;
  connected_clients: number;
  timestamp: string;
}

export interface AllMetrics {
  system: SystemMetrics;
  queue: QueueMetrics;
  database: DatabaseMetrics;
  redis: RedisMetrics;
}

export interface ReviewReportRequest {
  action: ReportReviewAction;
  notes?: string;
}

export interface BanUserRequest {
  reason: string;
  /** Go duration string like "72h". Omit for a permanent ban. */
  duration?: string;
}

// ---------------------------------------------------------------------------
// Simple message payloads
// ---------------------------------------------------------------------------

export interface MessageResponse {
  message: string;
}

export interface PageParams {
  page?: number;
  limit?: number;
}
