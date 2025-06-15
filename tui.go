package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2" // Import tcell
	"github.com/rivo/tview"      // Import tview
	"tailscale.com/ipn/ipnstate" // Import for ipnstate.Status
	"tailscale.com/tsnet"        // Import for tsnet.Server
)

// tuiActionResult is defined in main.go

// startTUI initializes and runs the interactive text-based user interface.
//
// The 'sshUser' parameter provided to this function is the username that will be
// used for all SSH connection attempts made from this TUI session.
// This username is determined at the launch of the application from:
// 1. The value passed to the '-l' command-line flag (e.g., -l myuser).
// 2. The 'user@' part of the target argument if specified (e.g., user@example-host)
//    when not using TUI mode (though this pre-parsed user is passed into TUI mode).
// 3. The current operating system's username if neither of the above is provided.
//
// It is not dynamically fetched per host within the TUI. The same sshUser is
// passed to connectToHostFromTUI and performSCPTransfer.
//
// Other parameters:
// - app: The tview application instance.
// - srv: The tsnet.Server for Tailscale connectivity.
// - appCtx: The main application context for cancellation.
// - logger: For logging messages.
// - initialStatus: The initial Tailscale network status.
// - sshKeyPath: Path to the SSH private key.
// - insecureHostKey: If true, host key checking is disabled.
// - verbose: If true, verbose logging is enabled.
//
// It returns a tuiActionResult indicating the user's choice and any error encountered.
func startTUI(app *tview.Application, srv *tsnet.Server, appCtx context.Context, logger *log.Logger, initialStatus *ipnstate.Status,
	sshUser string, sshKeyPath string, insecureHostKey bool, verbose bool) (tuiActionResult, error) {
	logger.Printf("TUI started with SSH user: %s (from -l flag, user@host arg, or OS user)", sshUser)

	result := tuiActionResult{} // This will store what the user selects

	// Pages manage different views within the TUI (e.g., host list, action choice, SCP form)
	pages := tview.NewPages()

	// List view for displaying Tailscale peers
	hostList := tview.NewList().ShowSecondaryText(true)
	hostList.SetBorder(true).SetTitle("Select a Host (ts-ssh) - Press (q) or Ctrl+C to Quit")

	// TextView for informational messages below the host list
	originalInfoText := "Use arrow keys to navigate, Enter to select. Press (q) or Ctrl+C to quit."
	infoBox := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true). // Enable dynamic colors for infoBox
		SetText(originalInfoText)

	// Function to populate the host list from Tailscale status
	populateHostList := func(currentStatus *ipnstate.Status) {
		hostList.Clear() // Clear previous items

		if currentStatus == nil || len(currentStatus.Peer) == 0 {
			hostList.AddItem("No peers found", "Please check your Tailscale network or wait for connection.", 0, nil)
			return
		}

		// Iterate over peers and add them to the list
		for _, peer := range currentStatus.Peer { // Renamed peerStatus to peer for clarity in capture
			var baseDisplayName, connectTarget string
			statusPrefix := ""
			styledDisplayName := ""

			// Determine the best display name and connection target
			if peer.DNSName != "" {
				baseDisplayName = strings.TrimSuffix(peer.DNSName, ".")
				connectTarget = baseDisplayName // Prefer DNS name for connection
			} else if peer.HostName != "" {
				baseDisplayName = peer.HostName
				connectTarget = baseDisplayName // Fallback to OS hostname
			} else if len(peer.TailscaleIPs) > 0 {
				baseDisplayName = peer.TailscaleIPs[0].String()
				connectTarget = peer.TailscaleIPs[0].String() // Fallback to IP
			} else {
				baseDisplayName = "Unknown Peer (ID: " + string(peer.ID) + ")"
				connectTarget = "unknown-peer-" + string(peer.ID) // Should not be selectable
			}

			// Secondary text for more details (IP, OS, online status)
			var secondaryText string
			if len(peer.TailscaleIPs) > 0 {
				secondaryText = peer.TailscaleIPs[0].String()
			} else {
				secondaryText = "No IP"
			}
			secondaryText += " | OS: " + peer.OS
			
			if peer.Online {
				statusPrefix = "[green][ONLINE][white] "
				secondaryText += " | Status: [green]Online[white]"
			} else {
				statusPrefix = "[red][OFFLINE][white] "
				secondaryText += " | Status: [red]Offline[white]"
			}
			styledDisplayName = statusPrefix + baseDisplayName


			// Store connectTarget for when an item is selected
			capturedPeer := peer                 // Capture current peer for the closure
			capturedDisplayName := baseDisplayName // Capture base display name for messages
			capturedConnectTarget := connectTarget

			if strings.HasPrefix(connectTarget, "unknown-peer-") {
				hostList.AddItem(styledDisplayName, secondaryText, 0, nil) // Not selectable
			} else {
				hostList.AddItem(styledDisplayName, secondaryText, 0, func() {
					if !capturedPeer.Online {
						logger.Printf("Host %s is offline, action prevented.", capturedDisplayName)
						infoBox.SetText(fmt.Sprintf("[red]Host %s is offline. Cannot perform actions.[white]", capturedDisplayName))
						// Reset infoBox text after a delay
						go func() {
							time.Sleep(3 * time.Second)
							app.QueueUpdateDraw(func() {
								// Check if the message is still the offline message before resetting
								if infoBox.GetText(false) == fmt.Sprintf("[red]Host %s is offline. Cannot perform actions.[white]", capturedDisplayName) {
									infoBox.SetText(originalInfoText)
								}
							})
						}()
						return
					}
					// If host is online, reset infoBox to original in case it was showing offline message
					if infoBox.GetText(false) != originalInfoText {
						infoBox.SetText(originalInfoText)
					}
					logger.Printf("Selected host: %s, connect target: %s", capturedDisplayName, capturedConnectTarget)
					result.selectedHostTarget = capturedConnectTarget
					pages.ShowPage("actionChoice") // Move to action choice page
				})
			}
		}
	}
	populateHostList(initialStatus) // Initial population
	hostList.SetChangedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		// When selection changes, reset infoBox to original text if it's not already
		// This handles cases where user moves away from an offline host without clicking
		currentText := strings.TrimSpace(infoBox.GetText(false))
		expectedOriginalInfoText := strings.TrimSpace(originalInfoText)
		if !strings.HasPrefix(currentText, "[red]Host") && currentText != expectedOriginalInfoText { // if it's some other message
			// This condition might be too broad. Let's be more specific or just reset if it's an error.
		} else if strings.HasPrefix(currentText, "[red]Host") { // If it's an offline message
			infoBox.SetText(originalInfoText)
		}
		// If a specific item is focused and it's an offline host, we could show a message,
		// but AddItem's func already handles clicks. This is for visual feedback on selection change.
	})

	// List view for choosing actions (SSH or SCP)
	actionList := tview.NewList().
		AddItem("Interactive SSH", "Connect via an interactive SSH shell", 's', func() {
			result.action = "ssh"
			app.Stop() // Stop the TUI application, main will proceed with SSH
		}).
		AddItem("SCP File Transfer", "Upload or download files via SCP", 'c', func() {
			result.action = "scp"
			// Before showing scpForm, ensure scpDetails are for the current host
			// This is implicitly handled as result.selectedHostTarget is already set.
			pages.ShowPage("scpForm") // Move to SCP parameters form
		}).
		AddItem("Back to Host List", "Choose a different host", 'b', func() {
			pages.HidePage("actionChoice")
			pages.ShowPage("hostSelection") // Go back to host list
		}).
		SetWrapAround(true) // Allow selection to wrap around
	actionList.SetBorder(true).SetTitle("Choose Action for: " + result.selectedHostTarget)
	// Update title when host changes - this needs to be dynamic if we go back and select another host.
	// For now, it's set once. If we implement dynamic title update, it would be in the hostList selection func.
	// This is now handled by pages.SetChangedFunc

	// Form for SCP parameters
	scpForm := tview.NewForm().
		AddInputField("Local Path:", "", 40, nil, func(text string) { result.scpLocalPath = text }).
		AddInputField("Remote Path:", "", 40, nil, func(text string) { result.scpRemotePath = text }).
		AddDropDown("Direction:", []string{"Upload (Local to Remote)", "Download (Remote to Local)"}, 0, func(option string, optionIndex int) {
			result.scpIsUpload = (optionIndex == 0)
		}).
		AddButton("Start SCP", func() {
			if result.scpLocalPath == "" || result.scpRemotePath == "" {
				// TODO: Show an error message in the TUI instead of just logging
				logger.Println("SCP Form: Local or Remote path is empty. User needs to be notified in TUI.")
				// How to show error in tview form? Maybe add a TextView to the form page for errors.
				// For now, we stop and the main function will handle empty paths if performSCPTransfer checks.
			}
			app.Stop() // Stop TUI, main will proceed with SCP
		}).
		AddButton("Cancel", func() {
			result.action = "" // Clear action
			pages.HidePage("scpForm")
			pages.ShowPage("actionChoice") // Go back to action choice
		})
	scpForm.SetBorder(true).SetTitle("SCP Parameters for: " + result.selectedHostTarget)
	// Similar to actionList, title might need dynamic update if host changes.

	// Add pages to the page manager
	pages.AddPage("hostSelection", tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(hostList, 0, 1, true). // hostList takes up most space, has focus
		AddItem(infoBox, 1, 0, false), // infoBox is small, no focus
		true, true) // Resizable, visible
	pages.AddPage("actionChoice", actionList, true, false) // Initially not visible
	pages.AddPage("scpForm", scpForm, true, false)         // Initially not visible

	// When actionChoice page is shown, update its title
	pages.SetChangedFunc(func() {
		name, item := pages.GetFrontPage()
		if name == "actionChoice" {
			if acl, ok := item.(*tview.List); ok {
				acl.SetTitle("Choose Action for: " + result.selectedHostTarget)
			}
		}
		if name == "scpForm" {
			if scpf, ok := item.(*tview.Form); ok {
				scpf.SetTitle("SCP Parameters for: " + result.selectedHostTarget)
			}
		}
	})

	// Global input capture for quitting and navigation
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlC {
			result.action = "" // Indicate exit
			app.Stop()
			return nil
		}
		currentPage, _ := pages.GetFrontPage()
		// If 'q' is pressed and we are on the host selection page, and hostList has focus
		if event.Rune() == 'q' && currentPage == "hostSelection" && app.GetFocus() == hostList {
			result.action = "" // Indicate exit
			app.Stop()
			return nil
		}
		// If 'q' is pressed and we are on actionChoice or scpForm, go back or quit
		if event.Rune() == 'q' {
			if currentPage == "actionChoice" {
				pages.HidePage("actionChoice").ShowPage("hostSelection")
				infoBox.SetText("Use arrow keys to navigate, Enter to select. Press (q) or Ctrl+C to quit.") // Reset infoBox
				return nil // Absorb 'q'
			} else if currentPage == "scpForm" {
				// scpForm's cancel button already handles this, but this provides a keyboard shortcut
				result.action = "" 
				pages.HidePage("scpForm").ShowPage("actionChoice")
				return nil // Absorb 'q'
			}
		}
		return event // Return event for default processing
	})

	// Goroutine to handle context cancellation (e.g., signal from main)
	go func() {
		<-appCtx.Done()
		logger.Println("TUI: Application context cancelled, stopping tview app.")
		result.action = "" // Ensure action is cleared on context cancel
		app.Stop()         // Stop the tview application
	}()

	logger.Println("Starting TUI application with Pages...")
	if err := app.SetRoot(pages, true).EnableMouse(true).Run(); err != nil {
		logger.Printf("Error running TUI: %v", err)
		return result, fmt.Errorf("error running TUI: %w", err)
	}

	// After app.Stop() is called, this point is reached.
	logger.Println("TUI application stopped.")
	return result, nil
}
