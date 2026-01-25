package alerts

import (
	"fmt"
	"os/exec"
	"strings"
)

// AlertEngine handles sending macOS notifications
type AlertEngine struct {
	sound string
}

// New creates a new AlertEngine
func New(sound string) *AlertEngine {
	return &AlertEngine{
		sound: sound,
	}
}

// SendNotification sends a macOS notification using osascript
func (a *AlertEngine) SendNotification(title, message, sound string) error {
	if sound == "" {
		sound = a.sound
	}
	if sound == "" {
		sound = "default"
	}

	// Escape quotes in message and title
	escapedTitle := strings.ReplaceAll(title, `"`, `\"`)
	escapedMessage := strings.ReplaceAll(message, `"`, `\"`)

	script := fmt.Sprintf(
		`display notification "%s" with title "%s" sound name "%s"`,
		escapedMessage,
		escapedTitle,
		sound,
	)

	cmd := exec.Command("osascript", "-e", script)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("sending notification: %w", err)
	}

	return nil
}

// CheckThresholdAlert checks if usage exceeds any threshold and sends alerts
func (a *AlertEngine) CheckThresholdAlert(usagePct float64, thresholds []float64, billingCycle string, alertAlreadySent func(string, float64, string) (bool, error), recordAlert func(string, float64, string) error) error {
	for _, threshold := range thresholds {
		if usagePct >= threshold {
			sent, err := alertAlreadySent("threshold", threshold, billingCycle)
			if err != nil {
				return fmt.Errorf("checking alert history: %w", err)
			}

			if !sent {
				message := fmt.Sprintf("Usage at %.1f%% (%v%% threshold)", usagePct, threshold)
				if err := a.SendNotification("Cursor Usage Alert", message, ""); err != nil {
					return err
				}

				if err := recordAlert("threshold", threshold, billingCycle); err != nil {
					return fmt.Errorf("recording alert: %w", err)
				}
			}
		}
	}
	return nil
}

// CheckOnDemandSwitch checks if billing switched from Included to On-Demand
func (a *AlertEngine) CheckOnDemandSwitch(currentIsOnDemand, previousIsOnDemand bool, onDemandSpendCents int) error {
	if currentIsOnDemand && !previousIsOnDemand {
		message := fmt.Sprintf(
			"Billing switched from Included to On-Demand. Current spend: $%.2f",
			float64(onDemandSpendCents)/100,
		)
		return a.SendNotification("CRITICAL: Cursor On-Demand Active", message, "Basso")
	}
	return nil
}
