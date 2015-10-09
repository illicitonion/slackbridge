package bridge

import (
	"sync"

	"github.com/matrix-org/slackbridge/matrix"
	"github.com/matrix-org/slackbridge/slack"
)

func NewUserMap() *UserMap {
	return &UserMap{
		matrixToSlack: make(map[string]*slack.User),
		slackToMatrix: make(map[string]*matrix.User),
	}
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

func (u *UserMap) Link(m *matrix.User, s *slack.User) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.matrixToSlack[m.UserID] = s
	u.slackToMatrix[s.UserID] = m
}

type UserMap struct {
	mu            sync.RWMutex
	matrixToSlack map[string]*slack.User
	slackToMatrix map[string]*matrix.User
}
