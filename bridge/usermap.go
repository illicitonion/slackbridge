package bridge

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/matrix-org/slackbridge/matrix"
	"github.com/matrix-org/slackbridge/slack"
)

func NewUserMap(db *sql.DB, httpClient http.Client, rooms *RoomMap, echoSuppresser *matrix.EchoSuppresser) (*UserMap, error) {
	m := &UserMap{
		matrixToSlack:  make(map[string]*slack.User),
		slackToMatrix:  make(map[string]*matrix.User),
		db:             db,
		echoSuppresser: echoSuppresser,
	}

	rows, err := db.Query("SELECT id, slack_user_id, slack_access_token, matrix_user_id, matrix_access_token, matrix_homeserver FROM users ORDER BY id ASC")
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var id int32
		var slackID, matrixID string
		var slackToken, matrixToken, matrixHomeserver sql.NullString
		if err := rows.Scan(&id, &slackID, &slackToken, &matrixID, &matrixToken, &matrixHomeserver); err != nil {
			return nil, err
		}
		if !matrixToken.Valid {
			log.Printf("Skipping user matrix:%q = slack:%q because no matrix token", matrixID, slackID)
			continue
		}
		if !matrixHomeserver.Valid {
			log.Printf("Skipping user matrix:%q = slack:%q because no matrix homeserver", matrixID, slackID)
			continue
		}
		if !slackToken.Valid {
			log.Printf("Skipping user matrix:%q = slack:%q because no slack token", matrixID, slackID)
			continue
		}
		matrixClient := matrix.NewClient(matrixToken.String, httpClient, matrixHomeserver.String, m.echoSuppresser)
		matrixUser := matrix.NewUser(matrixID, matrixClient)

		slackClient := slack.NewClient(slackToken.String, httpClient, rooms.ShouldNotify)
		slackUser := &slack.User{slackID, slackClient}

		if err := m.Link(matrixUser, slackUser); err != nil {
			return nil, err
		}
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	return m, nil
}

func (u *UserMap) MatrixForSlack(slackUser string) *matrix.User {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return u.slackToMatrix[slackUser]
}

func (u *UserMap) SlackForMatrix(matrixUser string) *slack.User {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return u.matrixToSlack[matrixUser]
}

func (u *UserMap) Link(m *matrix.User, s *slack.User) error {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.matrixToSlack[m.UserID] = s
	u.slackToMatrix[s.UserID] = m

	var count int32
	dbRow := u.db.QueryRow(`SELECT COUNT(id) FROM users WHERE slack_user_id == $1 AND matrix_user_id == $2`, s.UserID, m.UserID)
	scanErr := dbRow.Scan(&count)
	if scanErr == sql.ErrNoRows {
		log.Printf("Count query returned 0 rows - don't know what to do so skipping")
		return fmt.Errorf("count returned 0 rows")
	} else if scanErr != nil {
		return fmt.Errorf("error reading from db: %v", scanErr)
	}
	if count == 0 {
		if _, err := u.db.Exec(`INSERT INTO users (slack_user_id, slack_access_token, matrix_user_id, matrix_access_token, matrix_homeserver) VALUES ($1, $2, $3, $4, $5)`, s.UserID, s.Client.AccessToken(), m.UserID, m.Client.AccessToken(), m.Client.Homeserver()); err != nil {
			return fmt.Errorf("error writing to db: %v", err)
		}
	}
	return nil
}

type UserMap struct {
	mu             sync.RWMutex
	matrixToSlack  map[string]*slack.User
	slackToMatrix  map[string]*matrix.User
	echoSuppresser *matrix.EchoSuppresser
	db             *sql.DB
}
