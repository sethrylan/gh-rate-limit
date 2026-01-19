package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/cli/go-gh/v2"
	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/term"
)

// RateLimitResponse represents the GitHub API rate limit response
type RateLimitResponse struct {
	Resources map[string]RateLimit `json:"resources"`
	Rate      RateLimit            `json:"rate"`
}

// RateLimit represents a single rate limit category
type RateLimit struct {
	Limit     int   `json:"limit"`
	Remaining int   `json:"remaining"`
	Reset     int64 `json:"reset"`
	Used      int   `json:"used"`
}

// CategoryLimit combines a category name with its rate limit data
type CategoryLimit struct {
	Name  string
	Limit RateLimit
}

func main() {
	anonymous := flag.Bool("anonymous", false, "Show unauthenticated rate limits instead of authenticated")
	absolute := flag.Bool("absolute", false, "Display reset times as absolute timestamps instead of relative")
	jsonOutput := flag.Bool("json", false, "Output raw GitHub API response as JSON")
	flag.Parse()

	if err := run(*anonymous, *absolute, *jsonOutput); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}

func run(anonymous, absolute, jsonOutput bool) error {
	var responseBody []byte
	var err error

	if anonymous {
		responseBody, err = fetchAnonymous()
	} else {
		responseBody, err = fetchAuthenticated()
	}

	if err != nil {
		return err
	}

	// If JSON output is requested, print raw response and exit
	if jsonOutput {
		fmt.Println(string(responseBody))
		return nil
	}

	// Parse the response
	var rateLimit RateLimitResponse
	if err := json.Unmarshal(responseBody, &rateLimit); err != nil {
		return fmt.Errorf("failed to parse rate limit response: %w", err)
	}

	// Display the rate limits
	return displayRateLimits(rateLimit, absolute)
}

func fetchAuthenticated() ([]byte, error) {
	// Check if authenticated by trying to get auth status
	_, _, err := gh.Exec("auth", "status")
	if err != nil {
		return nil, errors.New("not authenticated\nRun `gh auth login` to authenticate, or use `gh rate-limit --anonymous` to check unauthenticated rate limits")
	}

	client, err := api.DefaultRESTClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	var response json.RawMessage
	err = client.Get("rate_limit", &response)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch rate limits: %w", err)
	}

	return response, nil
}

func fetchAnonymous() ([]byte, error) {
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/rate_limit", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch rate limits: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	return body, nil
}

func displayRateLimits(rateLimit RateLimitResponse, absolute bool) error {
	terminal := term.FromEnv()
	isTTY := terminal.IsTerminalOutput()
	termWidth, _, _ := terminal.Size()
	if termWidth == 0 {
		termWidth = 80 // Default width
	}

	// Collect all categories
	categories := make([]CategoryLimit, 0, len(rateLimit.Resources))
	for name, limit := range rateLimit.Resources {
		categories = append(categories, CategoryLimit{Name: name, Limit: limit})
	}

	// Sort by depletion level (lowest percentage remaining first)
	sort.Slice(categories, func(i, j int) bool {
		pctI := percentageRemaining(categories[i].Limit)
		pctJ := percentageRemaining(categories[j].Limit)
		return pctI < pctJ
	})

	// Calculate column widths
	maxNameLen := 0
	maxCountLen := 0
	for _, cat := range categories {
		if len(cat.Name) > maxNameLen {
			maxNameLen = len(cat.Name)
		}
		countStr := fmt.Sprintf("%d/%d", cat.Limit.Remaining, cat.Limit.Limit)
		if len(countStr) > maxCountLen {
			maxCountLen = len(countStr)
		}
	}

	// Calculate progress bar width
	// Format: name  count  [progressbar]  reset_time
	// We need space for: name + 2 spaces + count + 2 spaces + progressbar + 2 spaces + reset time (~20 chars)
	resetTimeWidth := 25
	fixedWidth := maxNameLen + 2 + maxCountLen + 2 + 2 + resetTimeWidth
	progressBarWidth := termWidth - fixedWidth
	progressBarWidth = max(progressBarWidth, 10)
	progressBarWidth = min(progressBarWidth, 40)

	// Display each category
	for _, cat := range categories {
		displayCategory(cat, maxNameLen, maxCountLen, progressBarWidth, absolute, isTTY)
	}

	return nil
}

func percentageRemaining(limit RateLimit) float64 {
	if limit.Limit == 0 {
		return 100.0 // Avoid division by zero; treat 0/0 as fully available
	}
	return float64(limit.Remaining) / float64(limit.Limit) * 100.0
}

func percentageUsed(limit RateLimit) float64 {
	if limit.Limit == 0 {
		return 0.0 // Avoid division by zero; treat 0/0 as nothing used
	}
	return float64(limit.Used) / float64(limit.Limit) * 100.0
}

func displayCategory(cat CategoryLimit, nameWidth, countWidth, barWidth int, absolute, isTTY bool) {
	pctRemaining := percentageRemaining(cat.Limit)
	pctUsed := percentageUsed(cat.Limit)
	// Exhausted means remaining is 0 but there was a limit to begin with
	isExhausted := cat.Limit.Remaining == 0 && cat.Limit.Limit > 0

	// Format the count
	countStr := fmt.Sprintf("%d/%d", cat.Limit.Remaining, cat.Limit.Limit)

	// Format reset time
	resetTime := formatResetTime(cat.Limit.Reset, absolute)

	// Build the progress bar
	progressBar := buildProgressBar(pctUsed, barWidth, isTTY, pctRemaining)

	// Apply coloring if TTY
	var line string
	if isTTY {
		nameStr := cat.Name
		countDisplay := countStr
		resetDisplay := resetTime
		if isExhausted {
			// Bold red for exhausted limits
			nameStr = boldRed(cat.Name)
			countDisplay = boldRed(countStr)
			resetDisplay = boldRed(resetTime)
		}
		// Manual padding to handle ANSI escape codes
		namePadding := strings.Repeat(" ", nameWidth-len(cat.Name))
		countPadding := strings.Repeat(" ", countWidth-len(countStr))
		line = fmt.Sprintf("%s%s  %s%s  %s  %s", nameStr, namePadding, countPadding, countDisplay, progressBar, resetDisplay)
	} else {
		// Plain text for non-TTY
		line = fmt.Sprintf("%-*s  %*s  %s  %s", nameWidth, cat.Name, countWidth, countStr, progressBar, resetTime)
	}

	fmt.Println(line)
}

func buildProgressBar(pctUsed float64, width int, isTTY bool, pctRemaining float64) string {
	if !isTTY {
		// Simple ASCII progress bar for non-TTY
		filled := min(int(pctUsed/100.0*float64(width)), width)
		return "[" + strings.Repeat("#", filled) + strings.Repeat("-", width-filled) + "]"
	}

	// Unicode block progress bar for TTY
	// Filled portion represents percentage used
	filledCount := min(int(pctUsed/100.0*float64(width)), width)
	emptyCount := width - filledCount

	// Build the bar with color based on remaining percentage
	color := getColor(pctRemaining)
	filled := strings.Repeat("█", filledCount)
	empty := strings.Repeat("░", emptyCount)

	return colorize(filled, color) + empty
}

func getColor(pctRemaining float64) int {
	// 256-color codes for gradient
	// Green (> 50%): 46 (bright green)
	// Yellow/Orange (20-50%): gradient from 226 (yellow) to 208 (orange)
	// Red (< 20%): gradient from 208 to 196 (red)

	switch {
	case pctRemaining > 50:
		// Green
		return 46
	case pctRemaining > 20:
		// Yellow to Orange gradient (50% -> 20%)
		// Map 50-20 to 226-208
		ratio := (pctRemaining - 20) / 30.0 // 0 to 1
		return 208 + int(ratio*18)          // 208 to 226
	default:
		// Orange to Red gradient (20% -> 0%)
		// Map 20-0 to 208-196
		ratio := pctRemaining / 20.0 // 0 to 1
		return 196 + int(ratio*12)   // 196 to 208
	}
}

func colorize(text string, color int) string {
	return fmt.Sprintf("\033[38;5;%dm%s\033[0m", color, text)
}

func boldRed(text string) string {
	return fmt.Sprintf("\033[1;31m%s\033[0m", text)
}

func formatResetTime(resetUnix int64, absolute bool) string {
	resetTime := time.Unix(resetUnix, 0)

	if absolute {
		return "resets at " + resetTime.Format("3:04:05 PM")
	}

	// Relative time
	duration := time.Until(resetTime)
	if duration < 0 {
		return "resets now"
	}

	return "resets in " + formatDuration(duration)
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}
