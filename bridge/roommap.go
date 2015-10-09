package bridge

import "sync"

func NewRoomMap() *RoomMap {
	return &RoomMap{
		matrixToSlack: make(map[string]string),
		slackToMatrix: make(map[string]string),
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

func (m *RoomMap) Link(matrix, slack string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.matrixToSlack[matrix] = slack
	m.slackToMatrix[slack] = matrix
}

type RoomMap struct {
	mu            sync.RWMutex
	matrixToSlack map[string]string
	slackToMatrix map[string]string
}
