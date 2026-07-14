import type { components } from "@/types/api";

/**
 * The backend's own schemas, re-exported under the names the app uses. Nothing
 * outside this file should reach into the generated `components["schemas"]`
 * shape: it is 4,000 lines of machine output, and every import of it is a place
 * that breaks when the generator changes its nesting.
 *
 * Regenerate with: npm run api:types
 */
type Schemas = components["schemas"];

export type Video = Schemas["Video"];
export type VideoStatus = Schemas["VideoStatus"];
export type VideoVisibility = Schemas["VideoVisibility"];
export type User = Schemas["User"];
export type Role = Schemas["Role"];
export type TokenPair = Schemas["TokenPair"];
export type VideoStatusReport = Schemas["VideoStatusReport"];
export type VideoAnalytics = Schemas["VideoAnalytics"];
export type Like = Schemas["Like"];
export type Comment = Schemas["Comment"];
export type Playlist = Schemas["Playlist"];
export type PlaylistItem = Schemas["PlaylistItem"];
export type WatchLaterItem = Schemas["WatchLaterItem"];
export type Notification = Schemas["Notification"];
export type SubscriptionEntry = Schemas["SubscriptionEntry"];
export type WatchHistory = Schemas["WatchHistory"];
export type VideoSearchItem = Schemas["VideoSearchItem"];
export type CategoryCount = Schemas["CategoryCount"];
export type ContentReport = Schemas["ContentReport"];
export type DashboardStats = Schemas["DashboardStats"];
export type PaginationMeta = Schemas["PaginationMeta"];

/**
 * A page of results. The API answers every list endpoint with the items under
 * `data` and the counts under `pagination`; the client unwraps that envelope, so
 * callers see this instead.
 */
export interface Page<T> {
  items: T[];
  pagination: PaginationMeta;
}

/** Query parameters shared by every paginated endpoint. */
export interface PageParams {
  page?: number;
  limit?: number;
}
