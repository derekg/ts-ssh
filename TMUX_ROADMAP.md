# Tmux-like Multi-Host Session Management Roadmap

## Overview
Add tmux-like functionality to ts-ssh allowing users to:
- Connect to multiple hosts simultaneously
- Switch between sessions with key combinations
- Split screen horizontally/vertically
- Manage multiple sessions in tabs

## Phase 1: Foundation (Current Phase)
**Goal**: Basic multi-session architecture

### 1.1 Session Management Core
- [ ] Create `Session` struct to represent individual host connections
- [ ] Create `SessionManager` to handle multiple sessions
- [ ] Design session lifecycle (create, pause, resume, destroy)
- [ ] Add session storage and retrieval

### 1.2 Basic Multi-Host Connection
- [ ] Extend TUI to support "Connect to Multiple Hosts" option
- [ ] Allow selecting multiple hosts from peer list
- [ ] Create concurrent SSH connections
- [ ] Store sessions in manager

### 1.3 Simple Session Switching
- [ ] Implement basic tab-based view using tview
- [ ] Add keyboard shortcuts for switching (Ctrl+N/P for next/prev)
- [ ] Display session status (connected, disconnected, active)
- [ ] Basic session identification (host name, connection status)

## Phase 2: Enhanced Display (Future)
**Goal**: Improved visual session management

### 2.1 Tabbed Interface
- [ ] Enhanced tab display with host names and status indicators
- [ ] Color coding for session states (active=green, idle=yellow, error=red)
- [ ] Tab close functionality
- [ ] Tab reordering

### 2.2 Session Information Panel
- [ ] Show current session details (host, uptime, last activity)
- [ ] Display connection statistics
- [ ] Session history/logs

## Phase 3: Screen Splitting (Future)
**Goal**: Multi-pane display like tmux

### 3.1 Horizontal Split
- [ ] Split current view horizontally
- [ ] Each pane shows different session
- [ ] Pane focus switching with arrow keys
- [ ] Pane resizing

### 3.2 Vertical Split
- [ ] Split current view vertically
- [ ] Support mixed horizontal/vertical layouts
- [ ] Pane management (create, destroy, move)

### 3.3 Complex Layouts
- [ ] Save/restore layout configurations
- [ ] Preset layouts (2x2 grid, sidebar, etc.)
- [ ] Dynamic pane resizing

## Phase 4: Advanced Features (Future)
**Goal**: Advanced session management

### 4.1 Session Persistence
- [ ] Save active sessions on exit
- [ ] Restore sessions on restart
- [ ] Session configuration files

### 4.2 Session Sharing
- [ ] Export session configurations
- [ ] Quick connect to common host groups
- [ ] Session templates

### 4.3 Advanced Controls
- [ ] Broadcast commands to multiple sessions
- [ ] Session groups/workspaces
- [ ] Session monitoring and alerting

## Technical Architecture

### Core Components
```go
type Session struct {
    ID          string
    HostTarget  string
    SSHClient   *ssh.Client
    Status      SessionStatus
    LastActive  time.Time
    PTY         *ssh.Session
    // ... other fields
}

type SessionManager struct {
    sessions    map[string]*Session
    activeID    string
    mu          sync.RWMutex
}

type MultiSessionTUI struct {
    manager     *SessionManager
    currentView ViewMode // tabs, split-h, split-v
    app         *tview.Application
    // ... other fields
}
```

### Key Considerations
- **Concurrency**: Each session runs in its own goroutine
- **Terminal Handling**: Manage raw mode per session
- **Resource Management**: Proper cleanup of connections
- **Error Handling**: Session failure recovery
- **Performance**: Efficient terminal rendering with multiple sessions

## Implementation Strategy
1. **Start Small**: Begin with basic session storage and tab switching
2. **Incremental**: Add one feature at a time, test thoroughly
3. **User Feedback**: Get input on UX before adding complexity
4. **Backwards Compatibility**: Maintain existing single-session functionality

## Success Criteria for Phase 1
- [ ] Can connect to multiple hosts from TUI
- [ ] Can switch between active sessions with keyboard shortcuts
- [ ] Sessions maintain state when not active
- [ ] Clean session termination and resource cleanup
- [ ] No interference with existing single-session mode