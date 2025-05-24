package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time" // Added import for time

	"github.com/gdamore/tcell/v2" 
	"github.com/rivo/tview"      
	"tailscale.com/ipn/ipnstate" 
	"tailscale.com/tsnet"        
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

	result := tuiActionResult{} 

	pages := tview.NewPages()

	hostList := tview.NewList().ShowSecondaryText(true)
	hostList.SetBorder(true).SetTitle("Select a Host (ts-ssh) - Press (q) or Ctrl+C to Quit")

	infoBox := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true). 
		SetText("Use arrow keys to navigate, Enter to select. Press (q) or Ctrl+C to quit.")

	populateHostList := func(currentStatus *ipnstate.Status) {
		hostList.Clear() 
		originalInfoText := "Use arrow keys to navigate, Enter to select. Press (q) or Ctrl+C to quit." // Defined originalInfoText

		if currentStatus == nil || len(currentStatus.Peer) == 0 {
			hostList.AddItem("No peers found", "Please check your Tailscale network or wait for connection.", 0, nil)
			return
		}

		for _, peer := range currentStatus.Peer { 
			var baseDisplayName, connectTarget string
			statusPrefix := ""
			styledDisplayName := ""

			if peer.DNSName != "" {
				baseDisplayName = strings.TrimSuffix(peer.DNSName, ".")
				connectTarget = baseDisplayName 
			} else if peer.HostName != "" {
				baseDisplayName = peer.HostName
				connectTarget = baseDisplayName 
			} else if len(peer.TailscaleIPs) > 0 {
				baseDisplayName = peer.TailscaleIPs[0].String()
				connectTarget = peer.TailscaleIPs[0].String() 
			} else {
				baseDisplayName = "Unknown Peer (ID: " + string(peer.ID) + ")"
				connectTarget = "unknown-peer-" + string(peer.ID) 
			}

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

			capturedPeer := peer                 
			capturedDisplayName := baseDisplayName 
			capturedConnectTarget := connectTarget

			if strings.HasPrefix(connectTarget, "unknown-peer-") {
				hostList.AddItem(styledDisplayName, secondaryText, 0, nil) 
			} else {
				hostList.AddItem(styledDisplayName, secondaryText, 0, func() {
					if !capturedPeer.Online {
						logger.Printf("Host %s is offline, action prevented.", capturedDisplayName)
						infoBox.SetText(fmt.Sprintf("[red]Host %s is offline. Cannot perform actions.[white]", capturedDisplayName))
						go func() {
							time.Sleep(3 * time.Second)
							app.QueueUpdateDraw(func() {
								if infoBox.GetText(false) == fmt.Sprintf("[red]Host %s is offline. Cannot perform actions.[white]", capturedDisplayName) {
									infoBox.SetText(originalInfoText)
								}
							})
						}()
						return
					}
					if infoBox.GetText(false) != originalInfoText {
						infoBox.SetText(originalInfoText)
					}
					logger.Printf("Selected host: %s, connect target: %s", capturedDisplayName, capturedConnectTarget)
					result.selectedHostTarget = capturedConnectTarget
					pages.ShowPage("actionChoice") 
				})
			}
		}
	}
	populateHostList(initialStatus) 
	hostList.SetChangedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		currentText := strings.TrimSpace(infoBox.GetText(false))
		// Define originalInfoText here as well, or pass it down if populateHostList is not recalled on selection change.
		// For simplicity, defining it again. A better approach might be to make originalInfoText a package-level var or pass to SetChangedFunc.
		originalInfoTextForChangedFunc := "Use arrow keys to navigate, Enter to select. Press (q) or Ctrl+C to quit."
		// expectedOriginalInfoText := strings.TrimSpace(originalInfoTextForChangedFunc) // This variable is not used
		
		if strings.HasPrefix(currentText, "[red]Host") { 
			infoBox.SetText(originalInfoTextForChangedFunc)
		}
	})

	actionList := tview.NewList().
		AddItem("Interactive SSH", "Connect via an interactive SSH shell", 's', func() {
			result.action = "ssh"
			app.Stop() 
		}).
		AddItem("SCP File Transfer", "Upload or download files via SCP", 'c', func() {
			result.action = "scp"
			pages.ShowPage("scpForm") 
		}).
		AddItem("Back to Host List", "Choose a different host", 'b', func() {
			pages.HidePage("actionChoice")
			pages.ShowPage("hostSelection") 
		}).
		SetWrapAround(true) 
	actionList.SetBorder(true).SetTitle("Choose Action for: " + result.selectedHostTarget)

	scpForm := tview.NewForm().
		AddInputField("Local Path:", "", 40, nil, func(text string) { result.scpLocalPath = text }).
		AddInputField("Remote Path:", "", 40, nil, func(text string) { result.scpRemotePath = text }).
		AddDropDown("Direction:", []string{"Upload (Local to Remote)", "Download (Remote to Local)"}, 0, func(option string, optionIndex int) {
			result.scpIsUpload = (optionIndex == 0)
		}).
		AddButton("Start SCP", func() {
			if result.scpLocalPath == "" || result.scpRemotePath == "" {
				logger.Println("SCP Form: Local or Remote path is empty. User needs to be notified in TUI.")
			}
			app.Stop() 
		}).
		AddButton("Cancel", func() {
			result.action = "" 
			pages.HidePage("scpForm")
			pages.ShowPage("actionChoice") 
		})
	scpForm.SetBorder(true).SetTitle("SCP Parameters for: " + result.selectedHostTarget)

	pages.AddPage("hostSelection", tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(hostList, 0, 1, true). 
		AddItem(infoBox, 1, 0, false), 
		true, true) 
	pages.AddPage("actionChoice", actionList, true, false) 
	pages.AddPage("scpForm", scpForm, true, false)         

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

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlC {
			result.action = "" 
			app.Stop()
			return nil
		}
		currentPage, _ := pages.GetFrontPage()
		if event.Rune() == 'q' && currentPage == "hostSelection" && app.GetFocus() == hostList {
			result.action = "" 
			app.Stop()
			return nil
		}
		if event.Rune() == 'q' {
			if currentPage == "actionChoice" {
				pages.HidePage("actionChoice").ShowPage("hostSelection")
				infoBox.SetText("Use arrow keys to navigate, Enter to select. Press (q) or Ctrl+C to quit.") 
				return nil 
			} else if currentPage == "scpForm" {
				result.action = "" 
				pages.HidePage("scpForm").ShowPage("actionChoice")
				return nil 
			}
		}
		return event 
	})

	go func() {
		<-appCtx.Done()
		logger.Println("TUI: Application context cancelled, stopping tview app.")
		result.action = "" 
		app.Stop()         
	}()

	logger.Println("Starting TUI application with Pages...")
	if err := app.SetRoot(pages, true).EnableMouse(true).Run(); err != nil {
		logger.Printf("Error running TUI: %v", err)
		return result, fmt.Errorf("error running TUI: %w", err)
	}

	logger.Println("TUI application stopped.")
	return result, nil
}
