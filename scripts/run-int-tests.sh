set -e

SCRIPT_DIR="$(dirname "$0")"
ENV_DIR="$(cd "$SCRIPT_DIR/../internal/tests/env" && pwd)"

echo "🚀 Starting integration tests..."
cd "$ENV_DIR" && docker-compose up --build --abort-on-container-exit --exit-code-from integration_test
echo "🎉 Integration tests completed successfully!"
