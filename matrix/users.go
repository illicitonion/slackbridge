package matrix

import (
	"sync"
)

func NewUsers() *Users {
	return &Users{
		users: make(map[string]*User),
	}
}

type Users struct {
	Mu    sync.Mutex
	users map[string]*User
}

func (u *Users) Get_Locked(userID string) *User {
	return u.users[userID]
}

func (u *Users) Save_Locked(user *User) {
	u.users[user.UserID] = user
}
