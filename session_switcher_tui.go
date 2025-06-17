package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// sessionSwitcherResult represents the result from the session switching TUI
type sessionSwitcherResult struct {
	action    string // "exit", "switch", "close_session"
	sessionID string // ID of the session to switch to or close
}

// startSessionSwitcherTUI starts the session switching interface
func startSessionSwitcherTUI(app *tview.Application, sessionManager *SessionManager, appCtx context.Context, logger *log.Logger) error {
	logger.Println("Starting session switcher TUI...")

	// Main layout containers
	pages := tview.NewPages()
	
	// Session tabs display (shows all sessions in tab format)
	sessionTabs := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	sessionTabs.SetBorder(true).SetTitle("Active Sessions (Tab/Shift+Tab to switch, 'q' to exit)")

	// Session details panel (shows info about current session)
	sessionDetails := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft).
		SetScrollable(true)
	sessionDetails.SetBorder(true).SetTitle("Session Details")

	// Terminal output panel (shows last output from active session)
	terminalOutput := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true)
	terminalOutput.SetBorder(true).SetTitle("Terminal Output (Active Session)")
	
	// Set up auto-scroll for terminal output
	terminalOutput.SetChangedFunc(func() {
		app.QueueUpdateDraw(func() {
			terminalOutput.ScrollToEnd()
		})
	})

	// Status bar
	statusBar := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	statusBar.SetBorder(false)

	// Input field for sending commands
	commandInput := tview.NewInputField().
		SetLabel("Command: ").
		SetFieldWidth(0)
	commandInput.SetBorder(true).SetTitle("Send Command to Active Session")
	
	// Set up command input handler
	commandInput.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			command := strings.TrimSpace(commandInput.GetText())
			if command != "" {
				// Send command to active session
				activeSession, err := sessionManager.GetActiveSession()
				if err != nil {
					statusBar.SetText(fmt.Sprintf("[red]Error: %v[white]", err))
					return
				}
				
				// Send the command
				_, err = activeSession.Write([]byte(command + "\n"))
				if err != nil {
					statusBar.SetText(fmt.Sprintf("[red]Failed to send command: %v[white]", err))
				} else {
					statusBar.SetText(fmt.Sprintf("[green]Sent: %s[white]", command))
					// Clear the input
					commandInput.SetText("")
				}
			}
		}
	})

	// Function to update session tabs display
	updateSessionTabs := func() {
		sessions := sessionManager.ListSessions()
		activeSessionID := ""
		if activeSession, err := sessionManager.GetActiveSession(); err == nil {
			activeSessionID = activeSession.ID
		}

		var tabsText strings.Builder
		for i, session := range sessions {
			if session.GetStatus() == SessionStatusClosed {
				continue // Skip closed sessions
			}
			
			// Format tab with session info
			statusColor := "white"
			switch session.GetStatus() {
			case SessionStatusActive:
				statusColor = "green"
			case SessionStatusIdle:
				statusColor = "yellow"
			case SessionStatusError:
				statusColor = "red"
			case SessionStatusConnecting:
				statusColor = "blue"
			}

			tabPrefix := "  "
			tabSuffix := "  "
			if session.ID == activeSessionID {
				tabPrefix = "[black:white] "
				tabSuffix = " [white:black]"
			}

			tabsText.WriteString(fmt.Sprintf("%s[%s]%d: %s%s", 
				tabPrefix, statusColor, i+1, session.GetDisplayName(), tabSuffix))
			
			if i < len(sessions)-1 {
				tabsText.WriteString("  |  ")
			}
		}

		sessionTabs.SetText(tabsText.String())
	}

	// Function to update session details
	updateSessionDetails := func() {
		activeSession, err := sessionManager.GetActiveSession()
		if err != nil {
			sessionDetails.SetText(fmt.Sprintf("[red]No active session: %v[white]", err))
			return
		}

		details := fmt.Sprintf(
			"[yellow]Session ID:[white] %s\n"+
			"[yellow]Host:[white] %s\n"+
			"[yellow]User:[white] %s\n"+
			"[yellow]Status:[white] %s\n"+
			"[yellow]Created:[white] %s\n"+
			"[yellow]Last Active:[white] %s\n"+
			"[yellow]Idle Time:[white] %s\n",
			activeSession.ID,
			activeSession.HostTarget,
			activeSession.SSHUser,
			activeSession.GetStatusDisplay(),
			activeSession.CreatedAt.Format("15:04:05"),
			activeSession.LastActive.Format("15:04:05"),
			activeSession.GetIdleTime().Truncate(time.Second),
		)

		if errorMsg := activeSession.GetError(); errorMsg != "" {
			details += fmt.Sprintf("[yellow]Error:[white] [red]%s[white]\n", errorMsg)
		}

		sessionDetails.SetText(details)
	}

	// Function to update status bar
	updateStatusBar := func() {
		stats := sessionManager.GetSessionStats()
		statusText := fmt.Sprintf(
			"Sessions: %d total, %d active, %d idle, %d error | "+
			"Controls: Tab/Shift+Tab (switch), Ctrl+C (close session), 'q' (exit)",
			stats["total"], stats["active"], stats["idle"], stats["error"])
		statusBar.SetText(statusText)
	}

	// Function to refresh all displays
	refreshDisplay := func() {
		updateSessionTabs()
		updateSessionDetails()
		updateStatusBar()
	}

	// Layout the interface
	// Top: Session tabs
	// Middle: Split between session details (left) and terminal output (right)
	// Bottom: Command input and status bar
	
	middlePanel := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(sessionDetails, 0, 1, false).
		AddItem(terminalOutput, 0, 2, false)

	bottomPanel := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(commandInput, 3, 0, true).
		AddItem(statusBar, 1, 0, false)

	mainLayout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(sessionTabs, 3, 0, false).
		AddItem(middlePanel, 0, 1, false).
		AddItem(bottomPanel, 4, 0, false)

	pages.AddPage("sessionSwitcher", mainLayout, true, true)

	// Set up keyboard handling
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyTab:
			// Switch to next session
			if err := sessionManager.SwitchToNextSession(); err != nil {
				statusBar.SetText(fmt.Sprintf("[red]Error switching session: %v[white]", err))
			} else {
				refreshDisplay()
			}
			return nil
			
		case tcell.KeyBacktab: // Shift+Tab
			// Switch to previous session
			if err := sessionManager.SwitchToPrevSession(); err != nil {
				statusBar.SetText(fmt.Sprintf("[red]Error switching session: %v[white]", err))
			} else {
				refreshDisplay()
			}
			return nil
			
		case tcell.KeyCtrlC:
			// Close current session
			activeSession, err := sessionManager.GetActiveSession()
			if err != nil {
				statusBar.SetText(fmt.Sprintf("[red]No active session to close: %v[white]", err))
				return nil
			}
			
			sessionID := activeSession.ID
			displayName := activeSession.GetDisplayName()
			
			if err := sessionManager.CloseSession(sessionID); err != nil {
				statusBar.SetText(fmt.Sprintf("[red]Error closing session: %v[white]", err))
			} else {
				statusBar.SetText(fmt.Sprintf("[green]Closed session: %s[white]", displayName))
				
				// If no more sessions, exit
				if sessionManager.GetSessionCount() == 0 {
					app.Stop()
					return nil
				}
				
				refreshDisplay()
			}
			return nil
		}

		// Handle 'q' key to quit
		if event.Rune() == 'q' {
			app.Stop()
			return nil
		}

		return event
	})

	// Initial display refresh
	refreshDisplay()

	// Set focus to command input initially
	app.SetFocus(commandInput)

	// Periodic refresh to update session status
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		
		for {
			select {
			case <-ticker.C:
				app.QueueUpdateDraw(func() {
					refreshDisplay()
				})
			case <-appCtx.Done():
				return
			}
		}
	}()

	// Run the TUI
	logger.Println("Session switcher TUI ready")
	if err := app.SetRoot(pages, true).EnableMouse(true).Run(); err != nil {
		return fmt.Errorf("error running session switcher TUI: %w", err)
	}

	return nil
}