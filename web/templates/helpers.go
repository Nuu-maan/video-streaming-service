package templates

import (
	"fmt"
	"time"

	"github.com/Nuu-maan/video-streaming-service/internal/domain"
)

// This file holds the shared view helpers for the templates package. It is a
// plain .go file (not a .templ) so `templ generate` never regenerates it away
// and the helpers stay declared exactly once in the package.

// formatDuration renders a duration given in whole seconds. Callers should skip
// rendering entirely when the duration is 0 ("unknown").
func formatDuration(seconds int) string {
	if seconds < 60 {
		return fmt.Sprintf("%d sec", seconds)
	}
	minutes := seconds / 60
	remainingSeconds := seconds % 60
	if minutes < 60 {
		return fmt.Sprintf("%d:%02d", minutes, remainingSeconds)
	}
	hours := minutes / 60
	remainingMinutes := minutes % 60
	return fmt.Sprintf("%d:%02d:%02d", hours, remainingMinutes, remainingSeconds)
}

// formatFileSize renders a byte count using binary units.
func formatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// formatTimeAgo renders a coarse relative time such as "3 hours ago".
func formatTimeAgo(t time.Time) string {
	diff := time.Since(t)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		mins := int(diff.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	default:
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}
}

// statusBadgeClass maps a video status onto its badge styling.
func statusBadgeClass(status domain.VideoStatus) string {
	switch status {
	case domain.VideoStatusReady:
		return "inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-600/30 text-green-200 border border-green-500/30"
	case domain.VideoStatusProcessing:
		return "inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-yellow-600/30 text-yellow-200 border border-yellow-500/30"
	case domain.VideoStatusFailed:
		return "inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-red-600/30 text-red-200 border border-red-500/30"
	default:
		return "inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-600/30 text-gray-200 border border-gray-500/30"
	}
}
