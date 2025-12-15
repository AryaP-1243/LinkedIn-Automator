# LinkedIn Automation Tool

## Overview
A Go-based LinkedIn automation proof-of-concept demonstrating advanced browser automation, anti-detection techniques, and clean architecture. Educational purposes only.

## Project Structure
- `cmd/linkedin-automation/main.go` - Main CLI entry point
- `pkg/auth/` - LinkedIn authentication system
- `pkg/browser/` - Rod browser automation wrapper
- `pkg/config/` - YAML/JSON configuration management
- `pkg/connection/` - Connection request handling
- `pkg/messaging/` - Messaging system with templates
- `pkg/search/` - Profile search and targeting
- `pkg/stealth/` - Anti-detection techniques (mouse, timing, fingerprinting, scrolling, typing, scheduling)
- `pkg/storage/` - JSON-based state persistence
- `pkg/logger/` - Structured logging system

## Key Technologies
- Go 1.24
- Rod (browser automation library)
- Chromium (headless browser)

## Configuration
- Environment variables: LINKEDIN_EMAIL, LINKEDIN_PASSWORD
- Config file: config.yaml (YAML/JSON supported)

## Running
```bash
go build -o linkedin-automation ./cmd/linkedin-automation
./linkedin-automation -config=config.yaml -mode=full
```

## Modes
- full: Complete automation cycle
- search: Search profiles only
- connect: Send connection requests only
- message: Send messages only
