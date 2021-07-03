set -e

# APIs

mockgen -package app_service_mock \
-destination internal/genmocks/types.go \
-source internal/types.go