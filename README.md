# LinkedIn Automation Tool

A comprehensive Go-based LinkedIn automation tool demonstrating advanced browser automation, anti-detection techniques, and clean architecture. This is an **educational proof-of-concept** for technical evaluation purposes only.

## Critical Disclaimer

**Educational Purpose Only**: This project is designed exclusively for technical evaluation and educational purposes. It demonstrates automation concepts and anti-detection techniques in a controlled environment.

**Terms of Service Violation**: Automating LinkedIn directly violates their Terms of Service. Using such tools on live accounts may result in permanent account bans, legal action, or other consequences.

**Do Not Use in Production**: This tool must never be deployed in production environments or used for actual LinkedIn automation. It exists solely to demonstrate technical capabilities and automation engineering skills.

## Demonstration Videos

Below are the proof-of-concept recordings demonstrating the bot's advanced human-like interaction and "Sonic" speed optimizations.

### 1. Stability & Robustness Pass
Demonstrates absolute error handling, "calm" login typing (100% accuracy), and zero-flicker visible cursor.

https://github.com/AryaP-1243/LinkedIn-Automator/raw/main/docs/assets/demo_v1_stability.mov

### 2. "Sonic" Speed Optimization
Demonstrates turbo-charged profile indexing and ultra-fast selection logic (instant transition from search to click).

https://github.com/AryaP-1243/LinkedIn-Automator/raw/main/docs/assets/demo_v2_sonic_speed.mov

*(Alternatively, refer to the `walkthrough.md` for highlighted screenshots and execution steps.)*

## Features

### Core Functional Requirements

- **Authentication System**
  - Login using credentials from environment variables
  - Detect and handle login failures gracefully
  - Identify security checkpoints (2FA, captcha)
  - Persist session cookies for seamless reuse

- **Search & Targeting**
  - Search users by job title, company, location, keywords
  - Parse and collect profile URLs efficiently
  - Handle pagination across search results
  - Implement duplicate profile detection

- **Connection Requests**
  - Navigate to user profiles programmatically
  - Click Connect button with precise targeting
  - Send personalized notes within character limits
  - Track sent requests and enforce daily limits

- **Messaging System**
  - Detect newly accepted connections
  - Send follow-up messages automatically
  - Support templates with dynamic variables
  - Maintain comprehensive message tracking

### Anti-Bot Detection Strategy

1. **Human-like Mouse Movement**
   - Bézier curves with variable speed
   - Natural overshoot and micro-corrections
   - Avoid straight-line trajectories that indicate bot behavior

2. **Randomized Timing Patterns**
   - Realistic, randomized delays between actions
   - Vary think time, scroll speed, and interaction intervals
   - Mimic human cognitive processing

3. **Browser Fingerprint Masking**
   - Modify user agent strings
   - Adjust viewport dimensions
   - Disable automation flags (navigator.webdriver)
   - Randomize browser properties to avoid detection

### Additional Stealth Techniques

- **Random Scrolling Behavior**: Variable scroll speeds with natural acceleration/deceleration
- **Realistic Typing Simulation**: Varied keystroke intervals with occasional typos and corrections
- **Mouse Hovering & Movement**: Random hover events and natural cursor wandering
- **Activity Scheduling**: Operate only during business hours with realistic break patterns
- **Rate Limiting & Throttling**: Connection request quotas with cooldown periods

## Project Structure

```
linkedin-automation/
├── cmd/
│   └── linkedin-automation/
│       └── main.go              # Main CLI entry point
├── pkg/
│   ├── auth/
│   │   └── auth.go              # Authentication system
│   ├── browser/
│   │   └── browser.go           # Browser automation wrapper
│   ├── config/
│   │   └── config.go            # Configuration management
│   ├── connection/
│   │   └── connection.go        # Connection request handling
│   ├── logger/
│   │   └── logger.go            # Structured logging
│   ├── messaging/
│   │   └── messaging.go         # Messaging system
│   ├── search/
│   │   └── search.go            # Search & targeting
│   ├── stealth/
│   │   ├── fingerprint.go       # Browser fingerprint masking
│   │   ├── mouse.go             # Human-like mouse movement
│   │   ├── scheduler.go         # Activity scheduling
│   │   ├── scrolling.go         # Random scrolling behavior
│   │   ├── timing.go            # Randomized timing patterns
│   │   └── typing.go            # Realistic typing simulation
│   └── storage/
│       └── storage.go           # State persistence (JSON)
├── data/                        # Runtime data directory
├── config.example.yaml          # Example configuration
├── .env.example                 # Environment variables template
├── go.mod                       # Go module definition
└── README.md                    # This file
```

## Installation

### Prerequisites

- Go 1.21 or later
- Chromium browser (automatically managed by Rod)

### Setup

1. Clone the repository:
   ```bash
   git clone https://github.com/your-username/linkedin-automation.git
   cd linkedin-automation
   ```

2. Install dependencies:
   ```bash
   go mod download
   ```

3. Copy and configure environment variables:
   ```bash
   cp .env.example .env
   # Edit .env with your credentials
   ```

4. Copy and customize the config file:
   ```bash
   cp config.example.yaml config.yaml
   # Edit config.yaml with your settings
   ```

5. Build the application:
   ```bash
   go build -o linkedin-automation ./cmd/linkedin-automation
   ```

## Configuration

### Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `LINKEDIN_EMAIL` | Your LinkedIn email | Yes |
| `LINKEDIN_PASSWORD` | Your LinkedIn password | Yes |
| `CONFIG_PATH` | Path to config file | No |
| `BROWSER_HEADLESS` | Run browser headless | No (default: true) |
| `LOG_LEVEL` | Logging level | No (default: info) |

### Configuration File

See `config.example.yaml` for a complete configuration reference with all available options.

## Usage

### Command Line Options

```bash
# Run with default settings (full automation mode)
./linkedin-automation

# Run with custom config file
./linkedin-automation -config=./my-config.yaml

# Run in specific mode
./linkedin-automation -mode=search    # Search only
./linkedin-automation -mode=connect   # Connection requests only
./linkedin-automation -mode=message   # Messaging only

# Dry run (no actual actions)
./linkedin-automation -dry-run
```

### Operation Modes

- **full**: Complete automation cycle (search → connect → message)
- **search**: Only search for profiles and save to storage
- **connect**: Only send connection requests to saved profiles
- **message**: Only send follow-up messages to accepted connections

## Code Quality Standards

### Modular Architecture

The codebase is organized into logical packages:
- `auth`: Authentication handling
- `search`: Profile search and targeting
- `connection`: Connection request management
- `messaging`: Message composition and delivery
- `stealth`: Anti-detection techniques
- `config`: Configuration management

### Robust Error Handling

- Comprehensive error detection
- Graceful degradation on failures
- Retry mechanisms with exponential backoff
- Detailed error logging

### Structured Logging

- Leveled logging (debug, info, warn, error)
- Contextual information with each log entry
- Configurable output formats (text/JSON)
- File and console output support

### Configuration Management

- YAML/JSON config file support
- Environment variable overrides
- Validation of config values
- Sensible defaults for all settings

### State Persistence

- JSON storage for connections and messages
- Session cookie persistence
- Enable resumption after interruptions
- Track all automation activities

## Evaluation Criteria

| Criterion | Weight | Description |
|-----------|--------|-------------|
| Anti-Detection Quality | 35% | Sophistication of stealth techniques |
| Automation Correctness | 30% | Accuracy and reliability of features |
| Code Architecture | 25% | Modularity and Go best practices |
| Practical Implementation | 10% | Real-world applicability |

## License

This project is for educational purposes only. Do not use for actual LinkedIn automation.

## Contributing

This is an educational project. Contributions that improve code quality, add documentation, or enhance the learning experience are welcome.
