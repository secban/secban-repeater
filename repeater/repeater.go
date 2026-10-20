package repeater

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"gitlab.com/secban/secban-agent/ipc"
)

const maxResponseBodyRead = 1024

var ErrAgentResponse = errors.New("agent error response")

type Client struct {
	client   *http.Client
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

func NewClient(agentURL string, timeout time.Duration) *Client {
	cleanURL := strings.TrimRight(agentURL, "/")

	httpClient := &http.Client{
		Timeout: timeout,
	}

	return &Client{
		agentURL: cleanURL,
		client:   httpClient,
		hostname: getHostname(),
	}
}

func (c *Client) Ban(ctx context.Context, msg ipc.Message) error {
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

func (c *Client) buildBanURL(msg ipc.Message) string {
	banPath := fmt.Sprintf(
		"/ban/%s/%s",
		url.PathEscape(msg.IP),
		url.PathEscape(msg.Rule),
	)
	fullURL := c.agentURL + banPath

	messageText := "repeated from " + msg.Rule
	if len(msg.Offender.Entries) > 0 {
		lastEntry := msg.Offender.Entries[len(msg.Offender.Entries)-1]
		if lastEntry.Message != "" {
			messageText = lastEntry.Message
		}
	}

	fullMessage := fmt.Sprintf("repeater %s %s", c.hostname, messageText)

	urlParams := url.Values{}
	urlParams.Set("message", fullMessage)

	return fullURL + "?" + urlParams.Encode()
}

func New(agentURLs []string, timeout time.Duration) *Repeater {
	clients := make([]*Client, 0, len(agentURLs))
	for _, agentURL := range agentURLs {
		clients = append(clients, NewClient(agentURL, timeout))
	}

	return &Repeater{
		clients: clients,
	}
}

func (r *Repeater) Repeat(ctx context.Context, msg ipc.Message) []error {
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
