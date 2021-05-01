package pirsch

import (
	// ClickHouse is an essential part of Pirsch.
	_ "github.com/ClickHouse/clickhouse-go"

	"database/sql"
	"github.com/jmoiron/sqlx"
	"log"
	"os"
	"time"
)

// Client is a ClickHouse database client.
type Client struct {
	sqlx.DB
	logger *log.Logger
}

// NewClient returns a new client for given database connection string.
// The logger is optional.
func NewClient(connection string, logger *log.Logger) (*Client, error) {
	c, err := sqlx.Open("clickhouse", connection)

	if err != nil {
		return nil, err
	}

	if err := c.Ping(); err != nil {
		return nil, err
	}

	if logger == nil {
		logger = log.New(os.Stdout, "[pirsch] ", log.LstdFlags)
	}

	return &Client{
		*c,
		logger,
	}, nil
}

// SaveHits implements the Store interface.
func (client *Client) SaveHits(hits []Hit) error {
	tx, err := client.Beginx()

	if err != nil {
		return err
	}

	query, err := tx.Prepare(`INSERT INTO "hit" (client_id, fingerprint, time, session, previous_time_on_page_seconds,
		user_agent, path, url, language, country_code, referrer, referrer_name, referrer_icon, os, os_version,
		browser, browser_version, desktop, mobile, screen_width, screen_height, screen_class) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`)

	if err != nil {
		return err
	}

	for _, hit := range hits {
		_, err := query.Exec(hit.ClientID,
			hit.Fingerprint,
			hit.Time,
			hit.Session,
			hit.PreviousTimeOnPageSeconds,
			hit.UserAgent,
			hit.Path,
			hit.URL,
			hit.Language,
			hit.CountryCode,
			hit.Referrer,
			hit.ReferrerName,
			hit.ReferrerIcon,
			hit.OS,
			hit.OSVersion,
			hit.Browser,
			hit.BrowserVersion,
			client.boolean(hit.Desktop),
			client.boolean(hit.Mobile),
			hit.ScreenWidth,
			hit.ScreenHeight,
			hit.ScreenClass)

		if err != nil {
			if e := tx.Rollback(); e != nil {
				client.logger.Printf("error rolling back transaction to save hits: %s", err)
			}

			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

// Session implements the Store interface.
func (client *Client) Session(clientID int64, fingerprint string, maxAge time.Time) (string, time.Time, time.Time, error) {
	query := `SELECT path, time, session FROM hit WHERE client_id = ? AND fingerprint = ? AND time > ? LIMIT 1`
	data := struct {
		Path    string
		Time    time.Time
		Session time.Time
	}{}

	if err := client.DB.Get(&data, query, clientID, fingerprint, maxAge); err != nil && err != sql.ErrNoRows {
		client.logger.Printf("error reading session timestamp: %s", err)
		return "", time.Time{}, time.Time{}, err
	}

	return data.Path, data.Time, data.Session, nil
}

// Count implements the Store interface.
func (client *Client) Count(query string, args ...interface{}) (int, error) {
	count := 0

	if err := client.DB.Get(&count, query, args...); err != nil {
		client.logger.Printf("error counting results: %s", err)
		return 0, err
	}

	return count, nil
}

// Get implements the Store interface.
func (client *Client) Get(query string, args ...interface{}) (*Stats, error) {
	stats := new(Stats)

	if err := client.DB.Get(stats, query, args...); err != nil {
		client.logger.Printf("error getting result: %s", err)
		return nil, err
	}

	return stats, nil
}

// Select implements the Store interface.
func (client *Client) Select(query string, args ...interface{}) ([]Stats, error) {
	var stats []Stats

	if err := client.DB.Select(&stats, query, args...); err != nil {
		client.logger.Printf("error selecting results: %s", err)
		return nil, err
	}

	return stats, nil
}

func (client *Client) boolean(b bool) int8 {
	if b {
		return 1
	}

	return 0
}
