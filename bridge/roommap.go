package bridge

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"sync"

	"github.com/matrix-org/slackbridge/slack"
)

func NewRoomMap(db *sql.DB) *RoomMap {
	return &RoomMap{
		matrixToSlack: make(map[string]string),
		slackToMatrix: make(map[string]string),
		rows:          make(map[string]*entry),

		/*
			CREATE TABLE IF NOT EXISTS rooms(
			id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			slack_channel_id TEXT,
			matrix_room_id TEXT,
			last_slack_timestamp TEXT,
			last_matrix_stream_token TEXT)
		*/
		db: db,
	}
}

func (m *RoomMap) MatrixForSlack(slack string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.slackToMatrix[slack]
}

func (m *RoomMap) SlackForMatrix(matrix string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.matrixToSlack[matrix]
}

func (m *RoomMap) Link(matrix, slack string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.matrixToSlack[matrix] = slack
	m.slackToMatrix[slack] = matrix
	row, ok := m.rows[matrix]
	if !ok {
		row = &entry{}
		m.rows[matrix] = row
	}
	dbRow := m.db.QueryRow(`SELECT id, slack_channel_id, matrix_room_id, last_slack_timestamp, last_matrix_stream_token FROM rooms WHERE slack_channel_id == $1 AND matrix_room_id == $2`, slack, matrix)
	scanErr := dbRow.Scan(&row.id, &row.SlackChannelID, &row.MatrixRoomID, &row.LastSlackTimestampS, &row.LastMatrixStreamToken)
	if scanErr == sql.ErrNoRows {
		log.Printf("Writing matrix room %q to table", matrix)
		if _, err := m.db.Exec(`INSERT INTO rooms (slack_channel_id, matrix_room_id) VALUES ($1, $2)`, slack, matrix); err != nil {
			return fmt.Errorf("error writing to db: %v", err)
		}
	} else if scanErr != nil {
		return fmt.Errorf("error reading from db: %v", scanErr)
	} else {
		log.Printf("Loaded row: %v", row)
	}
	return nil
}

func (m *RoomMap) ShouldNotify(message *slack.Message) bool {
	log.Printf("Got call to shouldNotify for: %v", message)
	matrix := m.MatrixForSlack(message.Channel)
	m.mu.RLock()
	row, ok := m.rows[matrix]
	m.mu.RUnlock()
	if !ok {
		return false
	}

	row.mu.Lock()
	defer row.mu.Unlock()
	if message.Timestamp() <= row.LastSlackTimestamp() {
		// We hope that Slack gives us messages in order.
		return false
	}

	if _, err := m.db.Exec(`UPDATE rooms SET last_slack_timestamp = $1 WHERE slack_channel_id == $2 AND matrix_room_id == $3`, message.TS, message.Channel, matrix); err != nil {
		log.Printf("Error updating DB for new Slack message: %v", err)
	}
	row.LastSlackTimestampS = sql.NullString{message.TS, true}
	return true
}

type RoomMap struct {
	mu            sync.RWMutex
	matrixToSlack map[string]string
	slackToMatrix map[string]string
	db            *sql.DB

	// matrix room ID -> mutex
	rows map[string]*entry
}

type entry struct {
	mu                    sync.RWMutex
	id                    int32
	SlackChannelID        string
	MatrixRoomID          string
	LastSlackTimestampS   sql.NullString
	LastMatrixStreamToken sql.NullString
}

func (e *entry) LastSlackTimestamp() float64 {
	if !e.LastSlackTimestampS.Valid {
		return 0
	}
	// We write this data, so if we have a parse error, the value is nil, so will parse to 0
	f, _ := strconv.ParseFloat(e.LastSlackTimestampS.String, 64)
	return f
}
