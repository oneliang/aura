# Habit Module

User operation habit learning system for Aura. Tracks user behavior patterns (tool usage, command sequences, output preferences) and builds personalized habits per user.

## Architecture

Layer 1 base module — depends only on `shared` and `storage`.

### Package Structure

```
modules/habit/
├── pkg/
│   ├── model/            # Data models
│   │   ├── habit.go      # Habit, Pattern, Frequency
│   │   ├── action.go     # Action record
│   │   └── preference.go # Preference record
│   ├── storage/          # JSONL storage layer (per-user isolated)
│   │   └── storage.go
│   ├── tracker/          # Action recording interface
│   │   └── tracker.go
│   ├── analyzer/         # Pattern analysis
│   │   ├── analyzer.go   # Mode recognition
│   │   └── rules.go      # Analysis rules
│   └── manager/          # Unified manager
│       ├── manager.go    # External interface
│       └── integration.go # Profile integration
├── go.mod
└── README.md
```

### Data Models

#### Habit

A learned behavior pattern:

```go
type Habit struct {
    ID         string    // Unique identifier
    UserID     string    // User isolation key
    Name       string    // e.g., "Prefers using glob tool"
    Category   string    // tool_usage / command / style / preference / workflow
    Pattern    Pattern   // Trigger pattern (keywords, tools, context)
    Frequency  Frequency // Count, trend, last active
    Confidence float64   // 0.0 - 1.0
    LastSeen   time.Time
}
```

#### Action

A raw operation record:

```go
type Action struct {
    ID          string
    UserID      string
    SessionID   string
    Timestamp   time.Time
    Input       string       // User input
    ToolsUsed   []string     // Tools invoked
    OutputStyle string       // concise/verbose
    Duration    time.Duration
    Feedback    string       // User feedback (e.g., "too long")
}
```

#### Preference

An explicit or implicit user preference:

```go
type Preference struct {
    ID        string
    UserID    string
    Category  string // tool/style/format
    Name      string // e.g., "output_style"
    Value     string // e.g., "concise"
    Source    string // explicit/implicit
    UpdatedAt time.Time
}
```

## Storage

Per-user isolated JSONL files under `~/.aura/users/{user_id}/habits/`:

```
~/.aura/users/
├── user_alice/
│   └── habits/
│       ├── habits.jsonl      # Habits (JSON array)
│       ├── actions.jsonl     # Raw action log (JSONL)
│       └── preferences.jsonl # Preferences (JSONL)
└── user_bob/
    └── habits/
        └── ...
```

## Usage

### CLI Commands

```bash
# List all habits for current user
aura habit list

# Show habit details
aura habit show <habit-id>

# Delete a habit
aura habit delete <habit-id>

# Re-analyze actions and refresh habits
aura habit refresh

# List preferences
aura preference list
```

### Programmatic

```go
// Create manager
mgr, err := manager.New(manager.DefaultConfig())

// Record an action
err := mgr.RecordAction(ctx, userID, &model.Action{
    UserID:    userID,
    SessionID: sessionID,
    Input:     input,
    ToolsUsed: []string{"grep", "file_read"},
})

// Get habits
habits, err := mgr.GetHabits(ctx, userID)

// Get preferences
prefs, err := mgr.GetPreferences(ctx, userID)

// Refresh habits from actions
habits, err := mgr.RefreshHabits(ctx, userID)
```

### Integration Points

#### Runtime Integration

The habit module is integrated into `core/pkg/runtime/runtime.go`:

1. **Initialization**: `AgentRuntime.Initialize()` creates `habitManager` when `userID != ""`
2. **Recording**: After each agent interaction completes, tool usage is asynchronously recorded as an action
3. **Non-invasive**: Uses existing event stream (`EventTypeToolStart`) to capture tool names — no Engine modifications needed

#### Backward Compatibility

- When `userID == ""` (legacy single-user mode), all habit operations are silently skipped
- Existing users are unaffected — habits only activate in multi-user mode

## Configuration

```go
type Config struct {
    MinOccurrences int           // Minimum occurrences to detect a habit (default: 3)
    ConfThreshold  float64       // Confidence threshold (default: 0.3)
    MaxActionAge   time.Duration // Max age of actions to analyze (default: 30 days)
    AnalysisLimit  int           // Max actions to analyze (default: 500)
}
```

## Analysis Rules

### Tool Usage Habits
- Counts frequency of each tool used by a user
- Creates a habit for tools used >= `MinOccurrences` times
- Confidence = usage_count / total_actions

### Output Style Preferences
- Analyzes `OutputStyle` field in action records
- Detects concise vs verbose preferences
- Creates `style` category habits

### Workflow Patterns
- Identifies common tool sequence patterns (consecutive pairs)
- e.g., `glob -> file_read` as a common search-then-read workflow
- Confidence based on pair co-occurrence frequency

## Testing

```bash
# Run all habit module tests
go test -v ./modules/habit/...

# Run with coverage
go test -cover ./modules/habit/pkg/...

# Run specific test
go test -run TestUserIsolation ./modules/habit/pkg/storage/...
```

### Test Coverage

| Package | Tests | Coverage |
|---------|-------|----------|
| `model` | 6 | 100% |
| `storage` | 12 | ~95% |
| `analyzer` | 10 | ~90% |
| `manager` | 11 | ~90% |
| **Total** | **39** | **~92%** |

## Multi-User Isolation

Key isolation mechanisms:

1. **Storage**: Each user's data in separate `~/.aura/users/{user_id}/habits/` directory
2. **Access**: All methods require `userID` parameter — no cross-user reads possible
3. **Recording**: Actions include `UserID` field — cannot be attributed to wrong user
4. **Legacy Mode**: Empty `userID` skips all operations — no mixed data

### Verification

Run the user isolation test:

```bash
go test -run TestUserIsolation ./modules/habit/pkg/storage/...
```

This verifies that User A's actions are completely invisible to User B and vice versa.
