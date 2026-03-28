package session

import (
	"crypto/rand"
	"encoding/hex"
)

const sessionTokenBytes = 24

type Store struct {
	tokens map[string]string
}

func NewStore() *Store {
	return &Store{
		tokens: make(map[string]string),
	}
}

func (s *Store) Create(userAddr string) (string, error) {
	token, err := newToken()
	if err != nil {
		return "", err
	}
	s.tokens[token] = userAddr
	return token, nil
}

func (s *Store) Lookup(token string) (string, bool) {
	userAddr, ok := s.tokens[token]
	return userAddr, ok
}

func (s *Store) Delete(token string) {
	delete(s.tokens, token)
}

func newToken() (string, error) {
	buffer := make([]byte, sessionTokenBytes)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer), nil
}
