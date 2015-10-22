package matrix

import (
	"log"
	"sync"
)

func NewUser(userID string, client Client) *User {
	return &User{
		UserID: userID,
		Client: client,
		rooms:  make(map[string]bool),
	}
}

type User struct {
	UserID string
	Client Client

	mu    sync.Mutex
	rooms map[string]bool
}

func (u *User) Rooms(update bool) map[string]bool {
	if update {
		u.updateRooms()
	}
	return u.rooms
}

func (u *User) JoinRoom(roomID string) error {
	err := u.Client.JoinRoom(roomID)
	if err == nil {
		u.rooms[roomID] = true
	}
	return err
}

func (u *User) updateRooms() {
	u.mu.Lock()
	defer u.mu.Unlock()
	rooms, err := u.Client.ListRooms()
	if err != nil {
		log.Printf("Error updating room list: %v", err)
		return
	}
	u.rooms = rooms
}
