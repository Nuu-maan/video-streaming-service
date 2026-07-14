package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Nuu-maan/video-streaming-service/internal/domain"
	"github.com/Nuu-maan/video-streaming-service/internal/service"
)

// onlyPublicReady is the single visibility predicate every discovery query in
// this file includes. Search, suggestions, trending, related and feeds must
// never surface a private, unlisted, or still-processing video; keeping the
// predicate in one constant means that property cannot be lost in one branch
// while surviving in another.
const onlyPublicReady = `v.status = 'ready' AND v.visibility = 'public'`

// searchItemColumns matches scanSearchItem field-for-field. Every query in this
// file aliases videos as v and users as u. The COALESCEs cover legacy rows:
// videos uploaded before authentication have a NULL user_id, and description,
// thumbnail and avatar are all nullable.
const searchItemColumns = `
	v.id, v.title, COALESCE(v.description, ''), COALESCE(v.thumbnail_path, ''),
	v.duration, v.view_count, v.created_at,
	COALESCE(u.id, '00000000-0000-0000-0000-000000000000'::uuid),
	COALESCE(u.username, ''), COALESCE(u.avatar_url, ''), COALESCE(u.email_verified, false)`

const searchFrom = ` FROM videos v LEFT JOIN users u ON u.id = v.user_id`

// SearchRepository serves search, discovery and recommendation reads. It is
// read-only by design: writes to the columns it queries belong to the upload
// and analytics paths.
type SearchRepository struct {
	pool *pgxpool.Pool
}

var _ service.SearchRepository = (*SearchRepository)(nil)

func NewSearchRepository(pool *pgxpool.Pool) *SearchRepository {
	return &SearchRepository{pool: pool}
}

// scanSearchItem reads one row in searchItemColumns order plus a trailing
// relevance column, which every query provides (ts_rank_cd for full-text
// search, a literal 0 elsewhere) so a single scanner serves them all.
func scanSearchItem(row scanner) (*domain.VideoSearchItem, error) {
	var item domain.VideoSearchItem
	var thumbnailKey string
	err := row.Scan(
		&item.VideoID,
		&item.Title,
		&item.Description,
		&thumbnailKey,
		&item.Duration,
		&item.Views,
		&item.CreatedAt,
		&item.UserID,
		&item.Username,
		&item.UserAvatarURL,
		&item.UserVerified,
		&item.Relevance,
	)
	if err != nil {
		return nil, err
	}

	// The column holds a storage key, which is not fetchable by a client. Search
	// hands back the same URL the full video record does, so a result list and a
	// video detail page never disagree about where the poster image lives.
	if thumbnailKey != "" {
		item.ThumbnailURL = domain.VideoThumbnailURL(item.VideoID)
	}

	item.Snippet = snippet(item.Description)
	return &item, nil
}

// snippet is a plain prefix of the description. ts_headline would give
// query-aware fragments but re-parses every document on every request, which
// is a poor trade for a result list that already carries the full description.
func snippet(description string) string {
	const maxRunes = 200
	runes := []rune(description)
	if len(runes) <= maxRunes {
		return description
	}
	return string(runes[:maxRunes]) + "…"
}

func collectSearchItems(rows pgx.Rows) ([]*domain.VideoSearchItem, error) {
	defer rows.Close()

	items := []*domain.VideoSearchItem{}
	for rows.Next() {
		item, err := scanSearchItem(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning search item: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating search items: %w", err)
	}
	return items, nil
}

// buildSearchWhere renders a SearchQuery into a WHERE clause and its bound
// arguments. The query text is always $1 so the rank expression in Search can
// reference it without threading a placeholder index around. Every value is
// bound as a placeholder; nothing user-supplied is interpolated into the SQL.
func buildSearchWhere(q *domain.SearchQuery) (string, []any) {
	args := []any{q.Query}
	conditions := []string{
		onlyPublicReady,
		// websearch_to_tsquery rather than plainto_tsquery: it supports quoted
		// phrases, OR and -exclusion, and unlike to_tsquery it never raises on
		// malformed input, so arbitrary user text is safe to pass through.
		"v.search_vector @@ websearch_to_tsquery('english', $1)",
	}

	add := func(condition string, value any) {
		args = append(args, value)
		conditions = append(conditions, fmt.Sprintf(condition, len(args)))
	}

	f := q.Filters
	if len(f.Categories) > 0 {
		add("v.category = ANY($%d)", f.Categories)
	}
	if len(f.Tags) > 0 {
		add("v.tags && $%d", f.Tags)
	}
	if f.Language != nil {
		add("v.language = $%d", *f.Language)
	}
	if len(f.Quality) > 0 {
		add("v.available_qualities && $%d", f.Quality)
	}
	if f.Duration != nil {
		min, max := f.Duration.ToSeconds()
		add("v.duration >= $%d", min)
		add("v.duration <= $%d", max)
	}
	if f.MinDuration != nil {
		add("v.duration >= $%d", *f.MinDuration)
	}
	if f.MaxDuration != nil {
		add("v.duration <= $%d", *f.MaxDuration)
	}
	if f.UploadDate != nil {
		add("v.created_at >= $%d", f.UploadDate.ToTime())
	}
	if f.MinViews != nil {
		add("v.view_count >= $%d", *f.MinViews)
	}
	if f.MaxViews != nil {
		add("v.view_count <= $%d", *f.MaxViews)
	}
	if f.OnlyVerified {
		conditions = append(conditions, "u.email_verified")
	}

	return " WHERE " + strings.Join(conditions, " AND "), args
}

func searchOrder(sortBy string) string {
	switch sortBy {
	case "newest":
		return "v.created_at DESC"
	case "views":
		return "v.view_count DESC, v.created_at DESC"
	case "likes":
		return "v.like_count DESC, v.created_at DESC"
	default:
		// created_at is the tiebreak so equally-ranked results page stably
		// instead of shuffling between requests.
		return "relevance DESC, v.created_at DESC"
	}
}

// Search runs a full-text query and returns one page of results plus the total
// match count. The count query shares the exact WHERE clause, so the total can
// never disagree with what paging through the results would yield.
func (r *SearchRepository) Search(ctx context.Context, q *domain.SearchQuery) ([]*domain.VideoSearchItem, int64, error) {
	where, args := buildSearchWhere(q)

	var total int64
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*)`+searchFrom+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting search results: %w", err)
	}
	if total == 0 {
		return []*domain.VideoSearchItem{}, 0, nil
	}

	query := fmt.Sprintf(
		`SELECT %s, ts_rank_cd(v.search_vector, websearch_to_tsquery('english', $1)) AS relevance%s%s ORDER BY %s LIMIT $%d OFFSET $%d`,
		searchItemColumns, searchFrom, where, searchOrder(q.SortBy), len(args)+1, len(args)+2,
	)
	args = append(args, q.Limit, (q.Page-1)*q.Limit)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("searching videos: %w", err)
	}
	items, err := collectSearchItems(rows)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// likeEscaper neutralises LIKE metacharacters in user input so a query for
// "100%" matches that literal text instead of acting as a wildcard.
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// Suggest returns up to limit titles for autocomplete. The ILIKE containment
// match and the similarity() ordering both run off the trigram GIN index on
// title; a bare similarity threshold was rejected because short prefixes like
// "ca" score near zero against long titles and returned nothing.
func (r *SearchRepository) Suggest(ctx context.Context, prefix string, limit int) ([]string, error) {
	query := `
	SELECT v.title
	FROM videos v
	WHERE ` + onlyPublicReady + ` AND v.title ILIKE $1
	ORDER BY similarity(v.title, $2) DESC, v.view_count DESC
	LIMIT $3`

	rows, err := r.pool.Query(ctx, query, "%"+likeEscaper.Replace(prefix)+"%", prefix, limit)
	if err != nil {
		return nil, fmt.Errorf("suggesting titles: %w", err)
	}
	defer rows.Close()

	titles := []string{}
	for rows.Next() {
		var title string
		if err := rows.Scan(&title); err != nil {
			return nil, fmt.Errorf("scanning suggestion: %w", err)
		}
		titles = append(titles, title)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating suggestions: %w", err)
	}
	return titles, nil
}

// Trending ranks public, ready videos by engagement inside the window. The
// score weights are the interaction weights from domain.InteractionType.Weight
// (view 1, like 5, comment 3, share 10), so search's notion of engagement
// cannot drift from the recommender's. Likes, comments and shares are lifetime
// counters — there is no per-window table for them — while views are counted
// from video_views inside the window; when the window holds no views the score
// degrades to lifetime engagement with lifetime view_count as the tiebreak,
// which is the graceful fallback for a quiet instance.
func (r *SearchRepository) Trending(ctx context.Context, windowHours, limit int, exclude []uuid.UUID) ([]*domain.VideoSearchItem, error) {
	if exclude == nil {
		exclude = []uuid.UUID{}
	}

	query := `
	SELECT ` + searchItemColumns + `, 0::float8 AS relevance
	FROM videos v
	LEFT JOIN users u ON u.id = v.user_id
	LEFT JOIN (
		SELECT video_id, COUNT(*) AS recent_views
		FROM video_views
		WHERE created_at > now() - make_interval(hours => $1)
		GROUP BY video_id
	) recent ON recent.video_id = v.id
	WHERE ` + onlyPublicReady + ` AND NOT (v.id = ANY($2))
	ORDER BY
		(COALESCE(recent.recent_views, 0) * $3 + v.like_count * $4 + v.comment_count * $5 + v.share_count * $6) DESC,
		v.view_count DESC,
		v.created_at DESC
	LIMIT $7`

	rows, err := r.pool.Query(ctx, query,
		windowHours, exclude,
		domain.InteractionView.Weight(),
		domain.InteractionLike.Weight(),
		domain.InteractionComment.Weight(),
		domain.InteractionShare.Weight(),
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("listing trending videos: %w", err)
	}
	return collectSearchItems(rows)
}

// Related returns content-based recommendations: videos sharing tags with the
// source (ranked by how many tags overlap), then same-category videos. This is
// content similarity over metadata we actually store — it is not collaborative
// filtering and does not pretend to be. A source with no tags and no category
// matches nothing here; the service layer tops the list up from trending.
func (r *SearchRepository) Related(ctx context.Context, videoID uuid.UUID, limit int) ([]*domain.VideoSearchItem, error) {
	var tags []string
	var category *string
	err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(tags, '{}'), category FROM videos WHERE id = $1`, videoID,
	).Scan(&tags, &category)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrVideoNotFound
		}
		return nil, fmt.Errorf("looking up source video %s: %w", videoID, err)
	}

	sourceCategory := ""
	if category != nil {
		sourceCategory = *category
	}

	// (v.category = $3) IS TRUE rather than a bare comparison: category is
	// nullable, and under DESC ordering Postgres puts NULLs first, which would
	// rank uncategorised videos above genuine category matches.
	query := `
	SELECT ` + searchItemColumns + `, 0::float8 AS relevance
	FROM videos v
	LEFT JOIN users u ON u.id = v.user_id
	WHERE ` + onlyPublicReady + `
	  AND v.id <> $1
	  AND (v.tags && $2 OR ($3 <> '' AND v.category = $3))
	ORDER BY
		cardinality(ARRAY(SELECT unnest(COALESCE(v.tags, '{}')) INTERSECT SELECT unnest($2::text[]))) DESC,
		((v.category = $3) IS TRUE) DESC,
		v.view_count DESC,
		v.created_at DESC
	LIMIT $4`

	rows, err := r.pool.Query(ctx, query, videoID, tags, sourceCategory, limit)
	if err != nil {
		return nil, fmt.Errorf("listing related videos for %s: %w", videoID, err)
	}
	return collectSearchItems(rows)
}

// Feed lists videos from creators the subscriber follows, newest first. It
// queries the subscriptions table directly rather than going through the
// subscriptions repository so this package stays free of a dependency on it.
// Deliberately public-only: unlisted means "reachable by link, not listed
// anywhere", and a subscription feed is a listing, so unlisted uploads from
// followed creators are excluded.
func (r *SearchRepository) Feed(ctx context.Context, subscriberID uuid.UUID, limit, offset int) ([]*domain.VideoSearchItem, int64, error) {
	from := `
	FROM videos v
	JOIN subscriptions s ON s.creator_id = v.user_id AND s.subscriber_id = $1
	JOIN users u ON u.id = v.user_id
	WHERE ` + onlyPublicReady

	var total int64
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*)`+from, subscriberID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting feed videos: %w", err)
	}
	if total == 0 {
		return []*domain.VideoSearchItem{}, 0, nil
	}

	query := `SELECT ` + searchItemColumns + `, 0::float8 AS relevance` + from + `
	ORDER BY v.created_at DESC
	LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, query, subscriberID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("listing feed videos: %w", err)
	}
	items, err := collectSearchItems(rows)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// Categories returns the distinct categories carried by publicly listable
// videos with their counts, most-populated first.
func (r *SearchRepository) Categories(ctx context.Context) ([]*domain.CategoryCount, error) {
	query := `
	SELECT v.category, COUNT(*)
	FROM videos v
	WHERE ` + onlyPublicReady + ` AND v.category IS NOT NULL
	GROUP BY v.category
	ORDER BY COUNT(*) DESC, v.category ASC`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("listing categories: %w", err)
	}
	defer rows.Close()

	categories := []*domain.CategoryCount{}
	for rows.Next() {
		var cc domain.CategoryCount
		if err := rows.Scan(&cc.Category, &cc.VideoCount); err != nil {
			return nil, fmt.Errorf("scanning category: %w", err)
		}
		categories = append(categories, &cc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating categories: %w", err)
	}
	return categories, nil
}
