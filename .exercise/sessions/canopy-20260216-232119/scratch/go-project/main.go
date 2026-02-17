package main

import (
	"fmt"
	"strings"
)

// User represents a user in the system
type User struct {
	ID    int
	Name  string
	Email string
}

// UserService provides user operations
type UserService interface {
	GetUser(id int) (*User, error)
	ListUsers() ([]*User, error)
	CreateUser(name, email string) (*User, error)
}

type userServiceImpl struct {
	users map[int]*User
	nextID int
}

func NewUserService() UserService {
	return &userServiceImpl{
		users:  make(map[int]*User),
		nextID: 1,
	}
}

func (s *userServiceImpl) GetUser(id int) (*User, error) {
	u, ok := s.users[id]
	if !ok {
		return nil, fmt.Errorf("user %d not found", id)
	}
	return u, nil
}

func (s *userServiceImpl) ListUsers() ([]*User, error) {
	result := make([]*User, 0, len(s.users))
	for _, u := range s.users {
		result = append(result, u)
	}
	return result, nil
}

func (s *userServiceImpl) CreateUser(name, email string) (*User, error) {
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("name is required")
	}
	u := &User{
		ID:    s.nextID,
		Name:  name,
		Email: email,
	}
	s.users[s.nextID] = u
	s.nextID++
	return u, nil
}

func main() {
	svc := NewUserService()
	u, _ := svc.CreateUser("Alice", "alice@example.com")
	fmt.Printf("Created user: %s\n", u.Name)

	found, _ := svc.GetUser(u.ID)
	fmt.Printf("Found user: %s (%s)\n", found.Name, found.Email)
}
