#!/bin/bash
set -e

# GitHub CLI helper script to manage manual Windows test builds
# Usage: ./scripts/test-windows-manual.sh [command] [options]

REPO="azophy/sshifu"
DEFAULT_BRANCH="fix/windows-npm-install-test"
DEFAULT_VERSION="0.7.8-test.1"

show_help() {
    cat << EOF
GitHub CLI Helper for Manual Windows Testing

Usage: $0 <command> [options]

Commands:
  trigger [version] [branch]   Trigger a new manual test build
  status <run-id>              Get status of a workflow run
  logs <run-id>                Stream logs from a workflow run
  list                         List recent test workflow runs
  cleanup [pattern]            Delete test releases matching pattern
  release <version>            Create a draft release from artifacts

Examples:
  $0 trigger 0.7.8-test.1 fix/windows-npm-install-test
  $0 status 1234567890
  $0 logs 1234567890
  $0 list
  $0 cleanup "test"
  $0 release 0.7.8-test.1
EOF
}

trigger_workflow() {
    local version="${1:-$DEFAULT_VERSION}"
    local branch="${2:-$DEFAULT_BRANCH}"
    
    echo "Triggering manual Windows test build..."
    echo "  Branch: $branch"
    echo "  Version: $version"
    
    gh workflow run test-windows-manual.yml \
        --repo "$REPO" \
        --field ref="$branch" \
        --field version="$version" \
        --field npm_tag="test" \
        --field publish_npm="true" \
        --field create_release="true" \
        --field run_tests="true"
    
    echo ""
    echo "Workflow triggered! Check status with:"
    echo "  gh run watch --repo $REPO"
    echo "  $0 status <run-id>"
}

list_runs() {
    echo "Recent manual test workflow runs:"
    gh run list --repo "$REPO" --workflow test-windows-manual.yml --limit 10
}

get_status() {
    local run_id="$1"
    if [ -z "$run_id" ]; then
        echo "Error: run-id required"
        exit 1
    fi
    gh run view "$run_id" --repo "$REPO"
}

stream_logs() {
    local run_id="$1"
    if [ -z "$run_id" ]; then
        echo "Error: run-id required"
        exit 1
    fi
    gh run watch "$run_id" --repo "$REPO" --log
}

cleanup_releases() {
    local pattern="${1:-test}"
    echo "Finding test releases matching '$pattern'..."
    
    gh release list --repo "$REPO" | grep -i "$pattern" | while read -r tag; do
        tag_name=$(echo "$tag" | awk '{print $1}')
        echo "Deleting release $tag_name..."
        gh release delete "$tag_name" --repo "$REPO" --yes --cleanup-tag
    done
    
    echo "Cleanup complete!"
}

# Main command handler
case "${1:-help}" in
    trigger)
        trigger_workflow "$2" "$3"
        ;;
    status)
        get_status "$2"
        ;;
    logs)
        stream_logs "$2"
        ;;
    list)
        list_runs
        ;;
    cleanup)
        cleanup_releases "$2"
        ;;
    help|--help|-h)
        show_help
        ;;
    *)
        echo "Unknown command: $1"
        show_help
        exit 1
        ;;
esac
