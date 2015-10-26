package common

import "sync"

func NewEchoSuppresser() *EchoSuppresser {
	return &EchoSuppresser{
		sentEvents: make(map[string]bool),
	}
}

type EchoSuppresser struct {
	sentEvents map[string]bool // eventID -> didSend
	mu         sync.RWMutex
	wg         sync.WaitGroup
}

func (s *EchoSuppresser) StartSending() {
	s.wg.Add(1)
}

func (s *EchoSuppresser) DoneSending() {
	s.wg.Add(-1)
}

func (s *EchoSuppresser) Wait() {
	s.wg.Wait()
}

func (s *EchoSuppresser) Sent(eventID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sentEvents[eventID] = true
}

func (s *EchoSuppresser) WasSent(eventID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sentEvents[eventID]
}
