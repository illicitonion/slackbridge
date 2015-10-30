package bridge

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"sync"

	"github.com/matrix-org/slackbridge/matrix"
	"github.com/matrix-org/slackbridge/slack"
)

func NewRoomMap(db *sql.DB) (*RoomMap, error) {
	m := &RoomMap{
		matrixToSlack: make(map[string]string),
		slackToMatrix: make(map[string]*matrix.Room),
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

	rows, err := db.Query("SELECT id, slack_channel_id, matrix_room_id FROM rooms ORDER BY id ASC")
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var id int32
		var slack string
		var matrixID string
		if err := rows.Scan(&id, &slack, &matrixID); err != nil {
			return nil, err
		}
		matrixRoom := matrix.NewRoom(matrixID)
		if err := m.Link(matrixRoom, slack); err != nil {
			return nil, err
		}
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	return m, nil
}

func (m *RoomMap) MatrixRoom(matrixRoomID string) *matrix.Room {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.slackToMatrix[m.matrixToSlack[matrixRoomID]]
}

func (m *RoomMap) MatrixForSlack(slack string) *matrix.Room {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.slackToMatrix[slack]
}

func (m *RoomMap) SlackForMatrix(matrix string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.matrixToSlack[matrix]
}

func (m *RoomMap) Link(matrix *matrix.Room, slack string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.matrixToSlack[matrix.ID] = slack
	m.slackToMatrix[slack] = matrix
	row, ok := m.rows[matrix.ID]
	if !ok {
		row = &entry{
			MatrixRoom: matrix,
		}
		m.rows[matrix.ID] = row
	}
	var lastMatrixStreamToken sql.NullString
	dbRow := m.db.QueryRow(`SELECT id, slack_channel_id, matrix_room_id, last_slack_timestamp, last_matrix_stream_token FROM rooms WHERE slack_channel_id == $1 AND matrix_room_id == $2`, slack, matrix.ID)
	scanErr := dbRow.Scan(&row.id, &row.SlackChannelID, &row.MatrixRoom.ID, &row.LastSlackTimestampS, &lastMatrixStreamToken)
	if scanErr == sql.ErrNoRows {
		log.Printf("Writing matrix room %q to table", matrix)
		if _, err := m.db.Exec(`INSERT INTO rooms (slack_channel_id, matrix_room_id) VALUES ($1, $2)`, slack, matrix.ID); err != nil {
			return fmt.Errorf("error writing to db: %v", err)
		}
	} else if scanErr != nil {
		return fmt.Errorf("error reading from db: %v", scanErr)
	} else {
		if lastMatrixStreamToken.Valid {
			row.MatrixRoom.LastStreamToken = lastMatrixStreamToken.String
		}
		log.Printf("Loaded row: %v", row)
	}
	return nil
}

func (m *RoomMap) ShouldNotify(message *slack.Message) bool {
	log.Printf("Got call to shouldNotify for: %v", message)
	matrix := m.MatrixForSlack(message.Channel)
	if matrix == nil {
		return false
	}
	m.mu.RLock()
	row, ok := m.rows[matrix.ID]
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

	if _, err := m.db.Exec(`UPDATE rooms SET last_slack_timestamp = $1 WHERE slack_channel_id == $2 AND matrix_room_id == $3`, message.TS, message.Channel, matrix.ID); err != nil {
		log.Printf("Error updating DB for new Slack message: %v", err)
	}
	row.LastSlackTimestampS = sql.NullString{message.TS, true}
	return true
}

type RoomMap struct {
	mu            sync.RWMutex
	matrixToSlack map[string]string
	slackToMatrix map[string]*matrix.Room
	db            *sql.DB

	// matrix room ID -> mutex
	rows map[string]*entry
}

type entry struct {
	mu                  sync.RWMutex
	id                  int32
	SlackChannelID      string
	LastSlackTimestampS sql.NullString
	MatrixRoom          *matrix.Room
}

func (e *entry) LastSlackTimestamp() float64 {
	if !e.LastSlackTimestampS.Valid {
		return 0
	}
	// We write this data, so if we have a parse error, the value is nil, so will parse to 0
	f, _ := strconv.ParseFloat(e.LastSlackTimestampS.String, 64)
	return f
}
