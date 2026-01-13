#!/bin/bash
# Load Testing Scripts for Video Streaming Platform
# Requires: vegeta (go install github.com/tsenart/vegeta@latest)

set -e

BASE_URL="${BASE_URL:-http://localhost:8080}"
DURATION="${DURATION:-30s}"
RATE="${RATE:-100}"

echo "Load Testing Video Streaming Platform"
echo "======================================"
echo "Base URL: $BASE_URL"
echo "Duration: $DURATION"
echo "Rate: $RATE requests/second"
echo ""

# Health check test
echo "1. Health Check Endpoint"
echo "GET $BASE_URL/health" | vegeta attack -duration=$DURATION -rate=$RATE | vegeta report
echo ""

# Video listing test
echo "2. Video Listing (Paginated)"
echo "GET $BASE_URL/api/v1/videos?page=1&limit=20" | vegeta attack -duration=$DURATION -rate=$RATE | vegeta report
echo ""

# Single video fetch
echo "3. Single Video Fetch"
cat << EOF | vegeta attack -duration=$DURATION -rate=$RATE | vegeta report
GET $BASE_URL/api/v1/videos/test-video-id
EOF
echo ""

# Search endpoint
echo "4. Search Endpoint"
cat << EOF | vegeta attack -duration=$DURATION -rate=50 | vegeta report
GET $BASE_URL/search?q=test&page=1&limit=20
EOF
echo ""

# HLS Master Playlist
echo "5. HLS Master Playlist"
cat << EOF | vegeta attack -duration=$DURATION -rate=200 | vegeta report
GET $BASE_URL/api/videos/test-video-id/hls/master.m3u8
EOF
echo ""

echo "Load testing complete!"
