#!/bin/bash
set -e

# Load environment variables from .env
export $(grep -v '^#' .env | xargs)

# Run the application with the provided arguments
./bin/telegramsender "$@"