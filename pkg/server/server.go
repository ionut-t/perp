package server

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"time"

	"github.com/google/uuid"
)

type Server struct {
	ID                     uuid.UUID `json:"id"`
	Name                   string    `json:"name"`
	Address                string    `json:"address"`
	Port                   int       `json:"port"`
	Database               string    `json:"database"`
	Username               string    `json:"username"`
	Password               string    `json:"password"`
	CreatedAt              time.Time `json:"createdAt"`
	UpdatedAt              time.Time `json:"updatedAt"`
	ShareDatabaseSchemaLLM bool      `json:"shareDatabaseSchemaLLM"`
	TimingEnabled          bool      `json:"timingEnabled"`
}

type CreateServer struct {
	Name                   string
	Address                string
	Port                   string
	Username               string
	Password               string
	Database               string
	ShareDatabaseSchemaLLM bool
}

// New creates a new server instance and saves it to the storage file.
func New(server CreateServer, storage string) (*Server, error) {
	port, err := strconv.Atoi(server.Port)

	if err != nil {
		return nil, fmt.Errorf("invalid port '%s': %w", server.Port, err)
	}

	newServer := &Server{
		ID:                     uuid.New(),
		Name:                   server.Name,
		Address:                server.Address,
		Port:                   port,
		Username:               server.Username,
		Password:               server.Password,
		Database:               server.Database,
		ShareDatabaseSchemaLLM: server.ShareDatabaseSchemaLLM,
		CreatedAt:              time.Now().In(time.UTC),
		UpdatedAt:              time.Now().In(time.UTC),
	}

	if err := save(newServer, storage); err != nil {
		return nil, fmt.Errorf("failed to save server: %w", err)
	}

	return newServer, nil
}

// Load retrieves all servers from the storage file.
func Load(storage string) ([]Server, error) {
	path := filepath.Join(storage, "servers.json")

	var servers []Server
	if _, err := os.Stat(path); err == nil {
		data, err := os.ReadFile(path)

		if err != nil {
			return nil, fmt.Errorf("failed to read server file: %w", err)
		}

		if err := json.Unmarshal(data, &servers); err != nil {
			return nil, fmt.Errorf("failed to unmarshal server file: %w", err)
		}
	}

	return servers, nil
}

func save(server *Server, storage string) error {
	path := filepath.Join(storage, "servers.json")

	var servers []Server
	if _, err := os.Stat(path); err == nil {
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read server file: %w", err)
		}

		if err := json.Unmarshal(data, &servers); err != nil {
			return fmt.Errorf("failed to unmarshal server file: %w", err)
		}
	}

	for _, existingServer := range servers {
		if existingServer.Name == server.Name && existingServer.ID != server.ID {
			return fmt.Errorf("server with name '%s' already exists", server.Name)
		}
	}

	servers = slices.DeleteFunc(servers, func(srv Server) bool {
		return srv.ID == server.ID
	})

	servers = append(servers, *server)
	slices.SortStableFunc(servers, func(a, b Server) int {
		return -1 * a.CreatedAt.Compare(b.CreatedAt)
	})

	data, err := json.MarshalIndent(servers, "", "  ")

	if err != nil {
		return fmt.Errorf("failed to marshal server data: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write server file: %w", err)
	}

	return nil
}

// Update modifies an existing server's details.
func (s *Server) Update(server CreateServer, storage string) error {
	port, err := strconv.Atoi(server.Port)

	if err != nil {
		return fmt.Errorf("invalid port '%s': %w", server.Port, err)
	}

	s.Name = server.Name
	s.Address = server.Address
	s.Port = port
	s.Username = server.Username
	s.Password = server.Password
	s.Database = server.Database
	s.ShareDatabaseSchemaLLM = server.ShareDatabaseSchemaLLM
	s.UpdatedAt = time.Now().In(time.UTC)

	if err := save(s, storage); err != nil {
		return fmt.Errorf("failed to update server: %w", err)
	}

	return nil
}

func (s *Server) EnableDatabaseSchemaLLM(enabled bool, storage string) error {
	if s.ShareDatabaseSchemaLLM == enabled {
		return nil
	}

	s.ShareDatabaseSchemaLLM = enabled
	s.UpdatedAt = time.Now().In(time.UTC)

	if err := save(s, storage); err != nil {
		return fmt.Errorf("failed to update server: %w", err)
	}

	return nil
}

func (s *Server) ToggleTiming(storage string) error {
	s.TimingEnabled = !s.TimingEnabled
	s.UpdatedAt = time.Now().In(time.UTC)

	if err := save(s, storage); err != nil {
		return fmt.Errorf("failed to update server: %w", err)
	}

	return nil
}

// Delete removes a server by its ID and returns the updated list of servers.
func Delete(id uuid.UUID, storage string) ([]Server, error) {
	path := filepath.Join(storage, "servers.json")

	var servers []Server
	if _, err := os.Stat(path); err == nil {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read server file: %w", err)
		}

		if err := json.Unmarshal(data, &servers); err != nil {
			return nil, fmt.Errorf("failed to unmarshal server file: %w", err)
		}
	}

	servers = slices.DeleteFunc(servers, func(srv Server) bool {
		return srv.ID == id
	})

	data, err := json.MarshalIndent(servers, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal server data: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return nil, fmt.Errorf("failed to write server file: %w", err)
	}

	return servers, nil
}

// ConnectionString returns the PostgreSQL connection string for the server.
func (s *Server) ConnectionString() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s",
		s.Username, s.Password, s.Address, s.Port, s.Database)
}
