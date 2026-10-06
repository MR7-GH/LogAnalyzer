#!/usr/bin/env bash

set -e

mkdir -p cmd/loganalyzer

mkdir -p internal/domain
mkdir -p internal/source
mkdir -p internal/elastic
mkdir -p internal/query
mkdir -p internal/analyze
mkdir -p internal/notify/telegram
mkdir -p internal/result
mkdir -p internal/api/handler
mkdir -p internal/api/routes
mkdir -p internal/config
mkdir -p internal/app

mkdir -p configs

cat > cmd/loganalyzer/main.go <<'EOF'
package main
EOF

cat > internal/domain/query.go <<'EOF'
package domain
EOF

cat > internal/domain/result.go <<'EOF'
package domain
EOF

cat > internal/domain/source.go <<'EOF'
package domain
EOF

cat > internal/domain/comparison.go <<'EOF'
package domain
EOF

cat > internal/domain/notification.go <<'EOF'
package domain
EOF

cat > internal/source/source.go <<'EOF'
package source
EOF

cat > internal/source/registry.go <<'EOF'
package source
EOF

cat > internal/elastic/client.go <<'EOF'
package elastic
EOF

cat > internal/elastic/source.go <<'EOF'
package elastic
EOF

cat > internal/elastic/query.go <<'EOF'
package elastic
EOF

cat > internal/elastic/mapper.go <<'EOF'
package elastic
EOF

cat > internal/query/service.go <<'EOF'
package query
EOF

cat > internal/query/validator.go <<'EOF'
package query
EOF

cat > internal/analyze/service.go <<'EOF'
package analyze
EOF

cat > internal/analyze/matcher.go <<'EOF'
package analyze
EOF

cat > internal/analyze/comparator.go <<'EOF'
package analyze
EOF

cat > internal/notify/sender.go <<'EOF'
package notify
EOF

cat > internal/notify/telegram/client.go <<'EOF'
package telegram
EOF

cat > internal/notify/telegram/sender.go <<'EOF'
package telegram
EOF

cat > internal/notify/telegram/formatter.go <<'EOF'
package telegram
EOF

cat > internal/result/sink.go <<'EOF'
package result
EOF

cat > internal/api/handler/query.go <<'EOF'
package handler
EOF

cat > internal/api/handler/analyze.go <<'EOF'
package handler
EOF

cat > internal/api/handler/health.go <<'EOF'
package handler
EOF

cat > internal/api/routes/routes.go <<'EOF'
package routes
EOF

cat > internal/config/config.go <<'EOF'
package config
EOF

cat > internal/app/app.go <<'EOF'
package app
EOF

touch configs/config.example.yaml
touch README.md
touch .gitignore

echo
echo "LogAnalyzer structure created successfully."
echo

find . \
  -path './.git' -prune -o \
  -type f -print | sort