# Phase 7: Search, Recommendations & Social Features

## Overview

Phase 7 transforms the video streaming platform into a fully social video community with advanced search, AI-powered recommendations, and comprehensive social interactions. This phase implements the features that drive user engagement, content discovery, and platform growth.

## Completed Components

### 1. Domain Models

#### Search Models (`internal/domain/search.go`)
- **SearchQuery**: Encapsulates search parameters with filters, sorting, and pagination
- **SearchFilters**: Duration, upload date, quality, categories, tags, views range, language
- **SearchResult**: Contains videos, total count, facets, and search metadata
- **VideoSearchItem**: Enhanced video info with relevance score and search snippets
- **SearchFacets**: Aggregated counts for categories, durations, and upload dates
- **DurationFilter**: Short (<4min), Medium (4-20min), Long (>20min)
- **DateFilter**: Today, Week, Month, Year

#### Recommendation Models (`internal/domain/recommendation.go`)
- **RecommendationEngine Interface**: Core methods for personalized recommendations, similar videos, trending content
- **InteractionType**: View, Like, Dislike, Comment, Share, Skip, Complete with weighted scoring
- **Interaction**: Tracks user-video interactions with timestamps
- **UserPreferences**: Favorite categories, tags, watched videos, liked/disliked videos, subscribed creators
- **VideoSimilarity**: Similarity scores between video pairs

#### Social Models (`internal/domain/social.go`)
- **Subscription**: User follows creator with notification preferences
- **Like**: Video likes/dislikes with user and video references
- **Comment**: Hierarchical comments with replies, likes, pinning, soft deletes
- **Playlist**: User-created playlists with visibility controls (public, private, unlisted)
- **PlaylistVideo**: Videos in playlists with position ordering
- **WatchHistory**: Track what users watched with resume positions and completion status
- **WatchLater**: Save videos for later viewing
- **Notification**: Multi-type notifications (new_video, comment, reply, like, subscriber, mention)
- **NotificationType**: Enum for different notification categories

### 2. Database Schema

#### Search Features Migration (`000008_add_search_features`)
- **Full-Text Search**: 
  - `search_vector` column (tsvector) with weighted title (A) and description (B)
  - GIN index for fast full-text queries
  - Auto-update trigger for search vector on INSERT/UPDATE
- **Trigram Search**: 
  - pg_trgm extension for fuzzy matching and autocomplete
  - Trigram index on titles for "type-ahead" suggestions
- **New Video Columns**:
  - `category` VARCHAR(50) - gaming, music, education, entertainment, etc.
  - `tags` TEXT[] - array of searchable tags
  - `language` VARCHAR(10) - content language (default: en)
  - `view_count`, `like_count`, `comment_count`, `share_count` - denormalized counters
- **Indexes**: Category, tags (GIN), language, view/like counts (DESC), created_at (DESC)

#### Social Features Migration (`000009_add_social_features`)
- **subscriptions** table: follower relationships with notification preferences
- **likes** table: video likes/dislikes (unique per user-video pair)
- **comments** table: hierarchical comments with soft deletes, pinning, like counts
- **playlists** table: user playlists with visibility controls
- **playlist_videos** table: videos in playlists with position ordering
- **watch_history** table: viewing history with resume positions and completion tracking
- **watch_later** table: saved videos queue
- **notifications** table: multi-type notifications with read status
- **Automated Triggers**:
  - Update video like/comment counts on INSERT/UPDATE/DELETE
  - Update user subscriber_count on subscription changes
  - Update playlist video_count on additions/removals
  - Auto-update timestamps on comments and playlists
  - Update reply counts for nested comments

### 3. Configuration

Added environment variables in `.env.example`:

**Search Configuration**:
- `ENABLE_AUTOCOMPLETE=true` - Enable search autocomplete
- `AUTOCOMPLETE_MIN_CHARS=2` - Minimum characters before suggesting
- `MAX_SEARCH_RESULTS=50` - Maximum results per page
- `SEARCH_CACHE_TTL=300` - Cache search results for 5 minutes

**Recommendation Configuration**:
- `RECOMMENDATION_CACHE_TTL=3600` - Cache recommendations for 1 hour
- `TRENDING_UPDATE_INTERVAL=600` - Recalculate trending every 10 minutes
- `PERSONALIZED_FEED_SIZE=20` - Number of videos in personalized feed
- `RECOMMENDATION_SIMILARITY_THRESHOLD=0.3` - Minimum similarity score

**Social Features Configuration**:
- `MAX_COMMENT_LENGTH=10000` - Maximum comment characters
- `ENABLE_COMMENT_REPLIES=true` - Allow nested comment replies
- `MAX_PLAYLIST_VIDEOS=500` - Maximum videos per playlist
- `WATCH_HISTORY_RETENTION_DAYS=365` - How long to keep watch history
- `MAX_SUBSCRIPTIONS_PER_USER=1000` - Subscription limit per user

**Notification Configuration**:
- `ENABLE_EMAIL_NOTIFICATIONS=false` - Send email notifications
- `ENABLE_PUSH_NOTIFICATIONS=false` - Send push notifications
- `BATCH_NOTIFICATIONS=true` - Batch similar notifications
- `NOTIFICATION_BATCH_WINDOW=3600` - Batch window in seconds (1 hour)
- `NOTIFICATION_CACHE_TTL=60` - Cache notification data

## Architecture Overview

### Search System
```
User Query → API Handler → Search Service → Redis Cache
                                ↓
                          PostgreSQL Full-Text Search
                                ↓
                          Results + Facets → Cache → Response
```

**Key Features**:
- PostgreSQL native full-text search with tsvector
- Weighted search (title has higher weight than description)
- Advanced filters: duration, date, category, tags, quality, views
- Faceted search results for UI filtering
- Redis caching of search results (5-minute TTL)
- Autocomplete with trigram similarity matching
- Popular/trending search tracking in Redis sorted sets

### Recommendation Engine
```
User Interactions → Redis → Recommendation Engine
                              ↓
                    ┌─────────┴─────────┐
           Collaborative      Content-Based
             Filtering          Filtering
                    └─────────┬─────────┘
                              ↓
                        Trending Algorithm
                              ↓
                        Personalized Feed
                              ↓
                          Cache → API
```

**Recommendation Algorithms**:

1. **Collaborative Filtering** (User-Based)
   - "Users who watched X also watched Y"
   - Calculate Jaccard similarity between users
   - Recommend videos from similar users

2. **Content-Based Filtering**
   - Match by category (40% weight)
   - Match by tags (30% weight)
   - Same creator (20% weight)
   - Similar duration (10% weight)

3. **Trending Algorithm**
   - Combines recency with engagement
   - Decay factor: `exp(-ageHours / 168)` (half-life 1 week)
   - Engagement: views + (likes × 10) + (comments × 5) + (shares × 20)
   - Velocity: views per hour bonus
   - Stored in Redis sorted set, updated every 10 minutes

4. **Personalized Feed**
   - Subscription videos (40% weight)
   - Collaborative recommendations (30% weight)
   - Trending in user's interests (20% weight)
   - General trending (10% weight)
   - Deduplicated and filtered (remove already watched)

### Social Interaction Flow

**Subscribe**:
```
User clicks Subscribe → API creates subscription → 
Trigger updates creator's subscriber_count → 
Send notification to creator →
Add to recommendation signals
```

**Like Video**:
```
User clicks Like → Insert/Update likes table →
Trigger updates video.like_count →
Record interaction for recommendations →
Cache invalidation
```

**Comment on Video**:
```
User posts comment → Validate content →
Insert comment → Trigger updates video.comment_count →
Send notification to video owner →
If reply, notify parent commenter
```

**Add to Playlist**:
```
User saves to playlist → Check ownership →
Get max position, increment → Insert playlist_video →
Trigger updates playlist.video_count →
Update playlist.updated_at
```

## Database Schema Details

### Key Relationships
- `subscriptions`: subscriber_id → users, creator_id → users
- `likes`: user_id → users, video_id → videos
- `comments`: user_id → users, video_id → videos, parent_id → comments (self-reference)
- `playlists`: user_id → users
- `playlist_videos`: playlist_id → playlists, video_id → videos
- `watch_history`: user_id → users, video_id → videos
- `watch_later`: user_id → users, video_id → videos
- `notifications`: user_id → users, actor_id → users, video_id → videos, comment_id → comments

### Indexes for Performance
- **Search**: GIN indexes on search_vector, tags; B-tree on category, language
- **Sorting**: DESC indexes on view_count, like_count, created_at
- **Social Lookups**: Composite indexes on (user_id, video_id), (video_id, created_at)
- **Notifications**: Index on (user_id, read, created_at) for fast unread queries

### Automated Counts
All denormalized counts are maintained by PostgreSQL triggers:
- `videos.like_count` - Updated on likes INSERT/UPDATE/DELETE
- `videos.comment_count` - Updated on comments INSERT/UPDATE/DELETE (respects soft deletes)
- `videos.view_count` - Updated on watch_history INSERT
- `users.subscriber_count` - Updated on subscriptions INSERT/DELETE
- `playlists.video_count` - Updated on playlist_videos INSERT/DELETE
- `comments.reply_count` - Updated on child comments INSERT/UPDATE/DELETE

## Implementation Roadmap

### Completed ✅
- [x] Domain models for search, recommendations, social features
- [x] Database migrations with comprehensive schema
- [x] Environment configuration
- [x] Architecture documentation

### Next Steps (In Order)

1. **Search Repository** (`internal/repository/postgres/search_repo.go`)
   - Implement `Search(ctx, query)` with full-text search SQL
   - Implement `GetSearchFacets(ctx, query)` for filter aggregations
   - Implement `AutocompleteSearch(ctx, prefix, limit)` with trigram similarity
   - Add helper methods for building dynamic WHERE clauses

2. **Social Repositories** (`internal/repository/postgres/`)
   - `subscription_repo.go`: CRUD operations for subscriptions
   - `like_repo.go`: CRUD operations for likes
   - `comment_repo.go`: CRUD with nested replies support
   - `playlist_repo.go`: Playlist and playlist_videos operations
   - `watch_history_repo.go`: History tracking with upsert
   - `watch_later_repo.go`: Simple CRUD operations
   - `notification_repo.go`: Create, fetch, mark read operations

3. **Search Service** (`internal/service/search_service.go`)
   - Redis caching layer with hash-based cache keys
   - Popular search tracking with Redis sorted sets (ZINCRBY)
   - Trending searches (ZADD with timestamp scores, ZREVRANGE)
   - Autocomplete caching (1-hour TTL)

4. **Recommendation Service** (`internal/service/recommendation_service.go`)
   - Implement collaborative filtering algorithm
   - Implement content-based similarity calculation
   - Implement trending score calculation
   - Implement personalized feed generation
   - Store user interactions in Redis sets
   - Background job to precompute recommendations

5. **Social Services** (`internal/service/`)
   - `subscription_service.go`: Subscribe/unsubscribe with validation
   - `like_service.go`: Like/dislike/remove with recommendation tracking
   - `comment_service.go`: Create/update/delete with profanity filtering
   - `playlist_service.go`: Full playlist management with reordering
   - `watch_history_service.go`: Track views, update view counts, resume positions
   - `watch_later_service.go`: Add/remove from queue
   - `notification_service.go`: Create, batch, send, mark read

6. **HTTP Handlers** (`internal/handler/`)
   - `search_handler.go`: /search, /api/search/autocomplete, /api/search/trending
   - `recommendation_handler.go`: /feed, /trending, /api/videos/:id/related
   - `social_handler.go`: All social API endpoints (subscribe, like, comment, playlist)

7. **HTMX Templates** (`web/templates/`)
   - Search page with filters
   - Video player with social interactions (like, subscribe, comment)
   - Comment section with replies
   - Personalized feed page
   - Trending page
   - Playlist management UI

8. **Real-Time Features**
   - Server-Sent Events (SSE) for notifications
   - Live view count updates
   - Real-time comment updates

## Testing Strategy

### Manual Testing
1. Run migrations: `make migrate-up`
2. Create test users with subscriptions
3. Upload videos with categories and tags
4. Test search with various filters
5. Test social interactions (like, comment, subscribe)
6. Verify all triggers update counts correctly
7. Check Redis caching behavior

### Integration Tests
- Search functionality with various query combinations
- Recommendation accuracy and caching
- Social interaction workflows
- Notification creation and batching
- Playlist ordering and management

### Performance Monitoring
- Search query execution time (<100ms target)
- Recommendation generation time (<500ms target)
- Redis cache hit rate (>80% target)
- Database query performance with EXPLAIN ANALYZE

## Deployment Checklist

- [ ] Run database migrations in production
- [ ] Configure Redis for caching and recommendations
- [ ] Set up background job for trending video updates
- [ ] Set up background job for recommendation precomputation
- [ ] Configure notification delivery (email/push if enabled)
- [ ] Monitor PostgreSQL full-text search performance
- [ ] Set up alerts for cache miss rates
- [ ] Monitor denormalized count accuracy

## API Endpoints (Future Implementation)

### Search
- `GET /search?q=query&category=gaming&duration=short&sort=views`
- `GET /api/search/autocomplete?q=prefix`
- `GET /api/search/trending`

### Recommendations
- `GET /feed` - Personalized home feed
- `GET /trending?category=gaming`
- `GET /api/videos/:id/related` - Similar videos

### Social
- `POST /api/users/:id/subscribe`
- `POST /api/users/:id/unsubscribe`
- `GET /api/users/:id/subscribers`
- `GET /api/users/:id/subscriptions`
- `POST /api/videos/:id/like`
- `POST /api/videos/:id/dislike`
- `DELETE /api/videos/:id/like`
- `POST /api/videos/:id/comments`
- `GET /api/videos/:id/comments?sort=top`
- `PUT /api/comments/:id`
- `DELETE /api/comments/:id`
- `POST /api/playlists`
- `POST /api/playlists/:id/videos`
- `DELETE /api/playlists/:id/videos/:videoId`
- `POST /api/videos/:id/watch`
- `GET /api/history`
- `POST /api/videos/:id/watch-later`
- `GET /api/watch-later`
- `GET /api/notifications`
- `PUT /api/notifications/:id/read`

## Performance Optimizations

### Database
- GIN indexes for full-text and array searches
- Materialized views for trending videos (optional)
- Partitioning for watch_history and notifications (future)
- Query result caching with Redis

### Caching Strategy
- Search results: 5-minute TTL
- Recommendations: 1-hour TTL (regenerate on interactions)
- Trending videos: 10-minute refresh cycle
- Autocomplete suggestions: 1-hour TTL
- User preferences: 30-minute TTL

### Redis Data Structures
- **Sets**: `user_interactions:{userID}` → Set of video IDs
- **Sets**: `video_watchers:{videoID}` → Set of user IDs
- **Sorted Sets**: `trending_videos` → score:videoID pairs
- **Sorted Sets**: `popular_searches` → count:query pairs
- **Strings/Hashes**: Cache keys for search results and recommendations

## Future Enhancements

- [ ] Machine learning models for better recommendations
- [ ] Elasticsearch for advanced search features
- [ ] Real-time collaboration (live comments during playback)
- [ ] Video tags auto-generation with AI
- [ ] Spam and toxicity detection in comments
- [ ] A/B testing for recommendation algorithms
- [ ] Analytics dashboard for content creators
- [ ] Recommendation explanation ("Why this video?")

## Conclusion

Phase 7 provides the foundation for a fully social video platform. The modular architecture allows for incremental implementation while the database schema and domain models are production-ready. The next steps focus on implementing the service and handler layers to bring these features to life.
