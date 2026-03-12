#!/bin/bash

# Start server:
# go run main.go --port 7700 &
# sleep 2

# Create index
echo "Creating user index..."
curl -X POST http://localhost:7700/indexes \
    -H 'Content-Type: application/json' \
    -d '{"name": "users"}'
echo

# Add documents
echo "Adding user documents..."
curl -X POST http://localhost:7700/indexes/users/documents \
    -H 'Content-Type: application/json' \
    -d '[
  {"id": "1", "name": "Alice Smith", "role": "admin"},
  {"id": "2", "name": "Bob Jones", "role": "editor"},
  {"id": "3", "name": "Charlie Brown", "role": "viewer"}
]'
echo

# Search
echo "Searching for 'admin'..."
curl "http://localhost:7700/indexes/users/search?q=admin"
echo

# Search exact name
echo "Searching for 'Alice'..."
curl "http://localhost:7700/indexes/users/search?q=Alice"
echo
