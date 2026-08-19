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
echo "$links_output" | grep -q "0 inbound, 1 outbound" || { echo "Inbound/outbound counts wrong: $links_output"; exit 1; }
echo "$links_output" | grep -q "ORPHAN" || { echo "test-card should be marked ORPHAN"; exit 1; }

# Broken links ([[link]] points at a card that doesn't exist)
broken_output=$($ZETTEL links --broken)
echo "$broken_output" | grep -q "test-card -> \[\[link\]\] (missing)" || { echo "Broken links failed: $broken_output"; exit 1; }

# Archive
$ZETTEL archive test-card
test -f "$PROJECT_DIR/.zettel/archived/test-card.md" || { echo "Archive failed: not in archived dir"; exit 1; }
test ! -f "$PROJECT_DIR/.zettel/cards/test-card.md" || { echo "Active card not removed"; exit 1; }

# Read from archived
output=$($ZETTEL read test-card)
echo "$output" | grep -q "Test Card" || { echo "Read from archived failed"; exit 1; }

# Archive already-archived is idempotent (no error)
$ZETTEL archive test-card || { echo "Idempotent re-archive should succeed"; exit 1; }
test -f "$PROJECT_DIR/.zettel/archived/test-card.md" || { echo "Archived card disappeared on re-archive"; exit 1; }

echo "=== All CLI integration tests passed ==="
