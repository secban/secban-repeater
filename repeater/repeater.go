package repeater

import (
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"gitlab.com/secban/secban-agent/ipc"
)

const (
	maxResponseBodyRead = 1024
	repeaterTag         = "[repeater]"
)

var ErrAgentResponse = errors.New("agent error response")

type Client struct {
	client   *http.Client
	rulesMap map[string]string
	agentURL string
	hostname string
}

type Repeater struct {
	clients []*Client
}

func getHostname() string {
	hostname, hostErr := os.Hostname()
	if hostErr != nil {
		return "unknown"
	}

	return hostname
}

func isRepeated(msg ipc.Message) bool {
	for _, entry := range msg.Offender.Entries {
		if strings.Contains(strings.ToLower(entry.Message), repeaterTag) {
			return true
		}
	}

	return false
}

func NewClient(agentURL string, timeout time.Duration, rulesMap map[string]string) *Client {
	cleanURL := strings.TrimRight(agentURL, "/")

	httpClient := &http.Client{
		Timeout: timeout,
	}

	copiedRulesMap := make(map[string]string, len(rulesMap))
	maps.Copy(copiedRulesMap, rulesMap)

	return &Client{
		agentURL: cleanURL,
		client:   httpClient,
		hostname: getHostname(),
		rulesMap: copiedRulesMap,
	}
}

func (c *Client) Ban(ctx context.Context, msg ipc.Message) error {
	if isRepeated(msg) {
		log.Debug().
			Str("agent", c.agentURL).
			Str("ip", msg.IP).
			Str("rule", msg.Rule).
			Msg("skipping repeated ban to prevent loop")

		return nil
	}

	fullURL := c.buildBanURL(msg)

	log.Debug().
		Str("agent", c.agentURL).
		Str("ip", msg.IP).
		Str("rule", msg.Rule).
		Str("url", fullURL).
		Msg("repeating ban to agent")

	req, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, nil)
	if reqErr != nil {
		return fmt.Errorf("failed to create ban request for %s: %w", c.agentURL, reqErr)
	}

	resp, doErr := c.client.Do(req)
	if doErr != nil {
		return fmt.Errorf("failed to send ban request to %s: %w", c.agentURL, doErr)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodyRead))

		return fmt.Errorf("%w (%s returned status %d: %s)", ErrAgentResponse, c.agentURL, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return nil
}

func (c *Client) resolveRule(localRule string) string {
	if mappedRule, exists := c.rulesMap[localRule]; exists && mappedRule != "" {
		return mappedRule
	}

	return localRule
}

func (c *Client) buildBanURL(msg ipc.Message) string {
	targetRule := c.resolveRule(msg.Rule)

	banPath := fmt.Sprintf(
		"/ban/%s/%s",
		url.PathEscape(msg.IP),
		url.PathEscape(targetRule),
	)
	fullURL := c.agentURL + banPath

	messageText := "[" + msg.Rule + "]"
	if len(msg.Offender.Entries) > 0 {
		lastEntry := msg.Offender.Entries[len(msg.Offender.Entries)-1]
		if lastEntry.Message != "" {
			messageText = lastEntry.Message
		}
	}

	fullMessage := fmt.Sprintf("%s %s %s", repeaterTag, c.hostname, messageText)

	urlParams := url.Values{}
	urlParams.Set("message", fullMessage)

	return fullURL + "?" + urlParams.Encode()
}

func New(agentURLs []string, timeout time.Duration, rulesMap map[string]string) *Repeater {
	clients := make([]*Client, 0, len(agentURLs))
	for _, agentURL := range agentURLs {
		clients = append(clients, NewClient(agentURL, timeout, rulesMap))
	}

	return &Repeater{
		clients: clients,
	}
}

func (r *Repeater) Repeat(ctx context.Context, msg ipc.Message) []error {
	if isRepeated(msg) {
		log.Debug().
			Str("ip", msg.IP).
			Str("rule", msg.Rule).
			Msg("skipping repeated ban to prevent loop")

		return nil
	}

	var errs []error

	for _, client := range r.clients {
		banErr := client.Ban(ctx, msg)
		if banErr != nil {
			log.Error().
				Str("agent", client.agentURL).
				Str("ip", msg.IP).
				Str("rule", msg.Rule).
				Err(banErr).
				Msg("failed to repeat ban to agent")

			errs = append(errs, banErr)
		} else {
			log.Info().
				Str("agent", client.agentURL).
				Str("ip", msg.IP).
				Str("rule", msg.Rule).
				Msg("successfully repeated ban to agent")
		}
	}

	return errs
}
