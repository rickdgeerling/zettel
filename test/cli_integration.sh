#!/bin/bash
set -e

export PROJECT_DIR=$(mktemp -d)
trap 'rm -rf "$PROJECT_DIR"' EXIT
SCRIPT_DIR=$(dirname "${BASH_SOURCE[0]}")
ZETTEL=$(realpath "$SCRIPT_DIR/../zettel")

cd "$PROJECT_DIR"

echo "=== Testing zettel CLI ==="

# Initialize the store
$ZETTEL init > /dev/null

# init should fail if already initialized
$ZETTEL init 2>&1 && { echo "Should have failed on double init"; exit 1; }

# Verify store-path resolution
path=$($ZETTEL store-path --quiet)
test "$path" = "$PROJECT_DIR/.zettel" || { echo "store-path failed: got '$path', want '$PROJECT_DIR/.zettel'"; exit 1; }

# Write a card via stdin
cat << 'EOF' | $ZETTEL write test-card
---
title: Test Card
created: 2025-01-15
---

This is the body with a [[link]] to another card.
EOF

# Read it back
output=$($ZETTEL read test-card)
echo "$output" | grep -q "Test Card" || { echo "Read failed: title not found"; exit 1; }
echo "$output" | grep -q "This is the body" || { echo "Body mismatch"; exit 1; }

# Search for it
results=$($ZETTEL search "Test")
echo "$results" | grep -q "test-card" || { echo "Search failed"; exit 1; }

# Links (global)
links_output=$($ZETTEL links)
echo "$links_output" | grep -q "test-card" || { echo "Links global failed: test-card not in graph"; exit 1; }
echo "$links_output" | grep -q "1 inbound" || { echo "Inbound count wrong"; exit 1; }

# Archive
$ZETTEL archive test-card
test -f "$PROJECT_DIR/.zettel/archived/test-card.md" || { echo "Archive failed: not in archived dir"; exit 1; }
test ! -f "$PROJECT_DIR/.zettel/cards/test-card.md" || { echo "Active card not removed"; exit 1; }

# Read from archived
output=$($ZETTEL read test-card)
echo "$output" | grep -q "Test Card" || { echo "Read from archived failed"; exit 1; }

# Archive already-archived should fail
$ZETTEL archive test-card && { echo "Should have failed on already-archived"; exit 1; }

echo "=== All CLI integration tests passed ==="
