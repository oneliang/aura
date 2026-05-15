package tui

import (
	"strings"
)

// renderOverlay renders a popup as a centered overlay on top of the base screen content.
func renderOverlay(base string, popupContent string, termWidth, termHeight int, styles UIStyles) string {
	popupLines := strings.Split(popupContent, "\n")
	popupH := len(popupLines)
	popupW := 0
	for _, line := range popupLines {
		if w := len(line); w > popupW {
			popupW = w
		}
	}
	// Ensure minimum dimensions
	if popupW < OverlayMinWidth {
		popupW = OverlayMinWidth
	}
	if popupH < OverlayMinHeight {
		popupH = OverlayMinHeight
	}

	// Calculate centered position
	top := (termHeight - popupH) / 2
	if top < OverlayPadding {
		top = OverlayPadding
	}

	// Split base into lines
	baseLines := strings.Split(base, "\n")

	// Render border line
	borderLine := styles.Separator.Render(strings.Repeat(OverlayBorderChar, popupW))

	// Build the bordered frame
	allLines := append([]string{borderLine}, popupLines...)
	allLines = append(allLines, borderLine)

	// Overlay onto base lines
	frameStart := top
	for i, overlayLine := range allLines {
		targetLine := frameStart + i
		if targetLine >= 0 && targetLine < len(baseLines) {
			baseLines[targetLine] = overlayLine
		}
	}

	return strings.Join(baseLines, "\n")
}

// renderOverlayAt renders a popup at a specific line position on the base screen.
// topLine is the 0-indexed line number where the popup should start.
func renderOverlayAt(base, content string, topLine, width int, styles UIStyles) string {
	popupLines := strings.Split(content, "\n")

	// Split base into lines and overlay
	baseLines := strings.Split(base, "\n")
	for i, overlayLine := range popupLines {
		targetLine := topLine + i
		if targetLine >= 0 && targetLine < len(baseLines) {
			baseLines[targetLine] = overlayLine
		}
	}

	return strings.Join(baseLines, "\n")
}
