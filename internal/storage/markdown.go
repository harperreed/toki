// ABOUTME: MarkdownStore provides file-based storage for toki using markdown files and YAML
// ABOUTME: Stores projects in _projects.yaml and todos as individual markdown files per project directory

package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/harperreed/mdstore"
	"gopkg.in/yaml.v3"
)

// MarkdownStore provides file-based storage for toki data using markdown files and YAML.
type MarkdownStore struct {
	dataDir string
}

// Compile-time check that MarkdownStore implements Storage.
var _ Storage = (*MarkdownStore)(nil)

// NewMarkdownStore creates a new markdown-backed store rooted at dataDir.
func NewMarkdownStore(dataDir string) (*MarkdownStore, error) {
	if err := mdstore.EnsureDir(dataDir); err != nil {
		return nil, fmt.Errorf("create data directory: %w", err)
	}
	return &MarkdownStore{dataDir: dataDir}, nil
}

// Close releases resources. For MarkdownStore this is a no-op.
func (s *MarkdownStore) Close() error {
	return nil
}

// projectsFilePath returns the path to the _projects.yaml file.
func (s *MarkdownStore) projectsFilePath() string {
	return filepath.Join(s.dataDir, "_projects.yaml")
}

// projectDirPath returns the directory path for a project.
func (s *MarkdownStore) projectDirPath(projectName string) string {
	return filepath.Join(s.dataDir, mdstore.Slugify(projectName))
}

// todoFileName generates a filename for a todo based on its UUID.
func todoFileName(todoID uuid.UUID) string {
	return fmt.Sprintf("todo-%s.md", todoID.String()[:8])
}

// projectEntry represents a single project in the _projects.yaml file.
type projectEntry struct {
	ID            string `yaml:"id"`
	Name          string `yaml:"name"`
	DirectoryPath string `yaml:"directory_path,omitempty"`
	CreatedAt     string `yaml:"created_at"`
}

// toModel converts a projectEntry to a storage.Project.
func (e *projectEntry) toModel() (*Project, error) {
	id, err := uuid.Parse(e.ID)
	if err != nil {
		return nil, fmt.Errorf("parse project ID %q: %w", e.ID, err)
	}
	createdAt, err := mdstore.ParseTime(e.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("parse project created_at %q: %w", e.CreatedAt, err)
	}
	return &Project{
		ID:            id,
		Name:          e.Name,
		DirectoryPath: e.DirectoryPath,
		CreatedAt:     createdAt,
	}, nil
}

// fromProjectModel converts a storage.Project to a projectEntry.
func fromProjectModel(p *Project) projectEntry {
	return projectEntry{
		ID:            p.ID.String(),
		Name:          p.Name,
		DirectoryPath: p.DirectoryPath,
		CreatedAt:     mdstore.FormatTime(p.CreatedAt.UTC()),
	}
}

// readProjects reads the _projects.yaml file.
func (s *MarkdownStore) readProjects() ([]projectEntry, error) {
	var entries []projectEntry
	if err := mdstore.ReadYAML(s.projectsFilePath(), &entries); err != nil {
		return nil, fmt.Errorf("read projects file: %w", err)
	}
	return entries, nil
}

// writeProjects writes the _projects.yaml file atomically.
func (s *MarkdownStore) writeProjects(entries []projectEntry) error {
	return mdstore.WriteYAML(s.projectsFilePath(), entries)
}

// todoFrontmatter holds the YAML frontmatter of a todo markdown file.
type todoFrontmatter struct {
	ID          string   `yaml:"id"`
	ProjectID   string   `yaml:"project_id"`
	Description string   `yaml:"description"`
	Done        bool     `yaml:"done"`
	Priority    string   `yaml:"priority,omitempty"`
	Tags        []string `yaml:"tags,omitempty"`
	CreatedAt   string   `yaml:"created_at"`
	UpdatedAt   string   `yaml:"updated_at"`
	CompletedAt string   `yaml:"completed_at,omitempty"`
	DueDate     string   `yaml:"due_date,omitempty"`
}

// readTodoFile reads a todo from a markdown file.
func readTodoFile(path string) (*Todo, error) {
	data, err := os.ReadFile(path) // #nosec G304 - path from internal storage directory
	if err != nil {
		return nil, err
	}

	yamlStr, body := mdstore.ParseFrontmatter(string(data))
	if yamlStr == "" {
		return nil, fmt.Errorf("no frontmatter found in %s", path)
	}

	var fm todoFrontmatter
	if err := yaml.Unmarshal([]byte(yamlStr), &fm); err != nil {
		return nil, fmt.Errorf("parse todo frontmatter in %s: %w", path, err)
	}

	return todoFromFrontmatter(&fm, strings.TrimSpace(body))
}

// todoFromFrontmatter converts a todoFrontmatter (plus body notes) into a storage.Todo.
func todoFromFrontmatter(fm *todoFrontmatter, notes string) (*Todo, error) {
	id, err := uuid.Parse(fm.ID)
	if err != nil {
		return nil, fmt.Errorf("parse todo ID %q: %w", fm.ID, err)
	}
	projectID, err := uuid.Parse(fm.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("parse project ID %q: %w", fm.ProjectID, err)
	}
	createdAt, err := mdstore.ParseTime(fm.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("parse created_at %q: %w", fm.CreatedAt, err)
	}
	updatedAt, err := mdstore.ParseTime(fm.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("parse updated_at %q: %w", fm.UpdatedAt, err)
	}

	todo := &Todo{
		ID:          id,
		ProjectID:   projectID,
		Description: fm.Description,
		Done:        fm.Done,
		Priority:    fm.Priority,
		Notes:       notes,
		Tags:        fm.Tags,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}

	if fm.CompletedAt != "" {
		t, err := mdstore.ParseTime(fm.CompletedAt)
		if err != nil {
			return nil, fmt.Errorf("parse completed_at %q: %w", fm.CompletedAt, err)
		}
		todo.CompletedAt = &t
	}
	if fm.DueDate != "" {
		t, err := mdstore.ParseTime(fm.DueDate)
		if err != nil {
			return nil, fmt.Errorf("parse due_date %q: %w", fm.DueDate, err)
		}
		todo.DueDate = &t
	}

	if todo.Tags == nil {
		todo.Tags = []string{}
	}

	return todo, nil
}

// writeTodoFile writes a todo to a markdown file.
func writeTodoFile(path string, todo *Todo) error {
	fm := todoFrontmatter{
		ID:          todo.ID.String(),
		ProjectID:   todo.ProjectID.String(),
		Description: todo.Description,
		Done:        todo.Done,
		Priority:    todo.Priority,
		Tags:        todo.Tags,
		CreatedAt:   mdstore.FormatTime(todo.CreatedAt.UTC()),
		UpdatedAt:   mdstore.FormatTime(todo.UpdatedAt.UTC()),
	}
	if todo.CompletedAt != nil {
		fm.CompletedAt = mdstore.FormatTime(todo.CompletedAt.UTC())
	}
	if todo.DueDate != nil {
		fm.DueDate = mdstore.FormatTime(todo.DueDate.UTC())
	}

	body := ""
	if todo.Notes != "" {
		body = "\n" + todo.Notes + "\n"
	}

	content, err := mdstore.RenderFrontmatter(&fm, body)
	if err != nil {
		return fmt.Errorf("render todo frontmatter: %w", err)
	}

	return mdstore.AtomicWrite(path, []byte(content))
}

// findTodoFile searches all project directories for a todo file with the given ID.
// Returns the file path and the project name.
func (s *MarkdownStore) findTodoFile(todoID uuid.UUID) (string, string, error) {
	projects, err := s.readProjects()
	if err != nil {
		return "", "", err
	}

	targetFile := todoFileName(todoID)
	for _, proj := range projects {
		projDir := s.projectDirPath(proj.Name)
		path := filepath.Join(projDir, targetFile)
		if _, err := os.Stat(path); err == nil {
			return path, proj.Name, nil
		}
	}

	return "", "", fmt.Errorf("todo not found: %s", todoID)
}

// findProjectForTodo finds the project entry that contains a todo by scanning projects.
func (s *MarkdownStore) findProjectName(projectID uuid.UUID) (string, error) {
	projects, err := s.readProjects()
	if err != nil {
		return "", err
	}
	for _, proj := range projects {
		if proj.ID == projectID.String() {
			return proj.Name, nil
		}
	}
	return "", fmt.Errorf("project not found: %s", projectID)
}

// --- Project operations ---

// CreateProject creates a new project.
func (s *MarkdownStore) CreateProject(project *Project) error {
	return mdstore.WithLock(s.dataDir, func() error {
		entries, err := s.readProjects()
		if err != nil {
			return err
		}

		// Check for duplicate name
		for _, e := range entries {
			if e.Name == project.Name {
				return fmt.Errorf("project already exists: %s", project.Name)
			}
		}

		entries = append(entries, fromProjectModel(project))
		if err := s.writeProjects(entries); err != nil {
			return err
		}

		// Create project directory
		return mdstore.EnsureDir(s.projectDirPath(project.Name))
	})
}

// GetProject retrieves a project by ID.
func (s *MarkdownStore) GetProject(id uuid.UUID) (*Project, error) {
	entries, err := s.readProjects()
	if err != nil {
		return nil, err
	}

	for _, e := range entries {
		if e.ID == id.String() {
			return e.toModel()
		}
	}

	return nil, fmt.Errorf("project not found: %s", id)
}

// GetProjectByName retrieves a project by name.
func (s *MarkdownStore) GetProjectByName(name string) (*Project, error) {
	entries, err := s.readProjects()
	if err != nil {
		return nil, err
	}

	for _, e := range entries {
		if e.Name == name {
			return e.toModel()
		}
	}

	return nil, fmt.Errorf("project not found: %s", name)
}

// GetProjectByPath retrieves a project by directory path.
func (s *MarkdownStore) GetProjectByPath(path string) (*Project, error) {
	entries, err := s.readProjects()
	if err != nil {
		return nil, err
	}

	for _, e := range entries {
		if e.DirectoryPath == path {
			return e.toModel()
		}
	}

	return nil, fmt.Errorf("project not found for path: %s", path)
}

// ListProjects returns all projects sorted by name.
func (s *MarkdownStore) ListProjects() ([]*Project, error) {
	entries, err := s.readProjects()
	if err != nil {
		return nil, err
	}

	var projects []*Project
	for _, e := range entries {
		p, err := e.toModel()
		if err != nil {
			// Skip malformed entries
			continue
		}
		projects = append(projects, p)
	}

	sort.Slice(projects, func(i, j int) bool {
		return projects[i].Name < projects[j].Name
	})

	return projects, nil
}

// UpdateProject updates an existing project.
func (s *MarkdownStore) UpdateProject(project *Project) error {
	return mdstore.WithLock(s.dataDir, func() error {
		entries, err := s.readProjects()
		if err != nil {
			return err
		}

		found := false
		var oldName string
		for i, e := range entries {
			if e.ID == project.ID.String() {
				oldName = e.Name
				entries[i] = fromProjectModel(project)
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("project not found: %s", project.ID)
		}

		if err := s.writeProjects(entries); err != nil {
			return err
		}

		// Handle project directory rename if name changed
		if oldName != project.Name {
			oldDir := s.projectDirPath(oldName)
			newDir := s.projectDirPath(project.Name)
			if _, err := os.Stat(oldDir); err == nil {
				if err := os.Rename(oldDir, newDir); err != nil {
					return fmt.Errorf("rename project directory: %w", err)
				}
			} else {
				// Create the new directory if old one didn't exist
				return mdstore.EnsureDir(newDir)
			}
		}

		return nil
	})
}

// DeleteProject removes a project by ID, along with all its todo files.
func (s *MarkdownStore) DeleteProject(id uuid.UUID) error {
	return mdstore.WithLock(s.dataDir, func() error {
		entries, err := s.readProjects()
		if err != nil {
			return err
		}

		found := false
		var projectName string
		var remaining []projectEntry
		for _, e := range entries {
			if e.ID == id.String() {
				found = true
				projectName = e.Name
			} else {
				remaining = append(remaining, e)
			}
		}

		if !found {
			return fmt.Errorf("project not found: %s", id)
		}

		if err := s.writeProjects(remaining); err != nil {
			return err
		}

		// Remove project directory and all contents
		projDir := s.projectDirPath(projectName)
		if err := os.RemoveAll(projDir); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove project directory: %w", err)
		}

		return nil
	})
}

// --- Todo operations ---

// CreateTodo creates a new todo with optional tags.
func (s *MarkdownStore) CreateTodo(todo *Todo) error {
	projectName, err := s.findProjectName(todo.ProjectID)
	if err != nil {
		return err
	}

	projDir := s.projectDirPath(projectName)
	if err := mdstore.EnsureDir(projDir); err != nil {
		return fmt.Errorf("ensure project directory: %w", err)
	}

	path := filepath.Join(projDir, todoFileName(todo.ID))
	return writeTodoFile(path, todo)
}

// GetTodo retrieves a todo by ID.
func (s *MarkdownStore) GetTodo(id uuid.UUID) (*Todo, error) {
	path, projectName, err := s.findTodoFile(id)
	if err != nil {
		return nil, err
	}

	todo, err := readTodoFile(path)
	if err != nil {
		return nil, err
	}

	todo.ProjectName = projectName
	return todo, nil
}

// GetTodoByPrefix retrieves a todo by ID prefix.
func (s *MarkdownStore) GetTodoByPrefix(prefix string) (*Todo, error) {
	projects, err := s.readProjects()
	if err != nil {
		return nil, err
	}

	var matches []*Todo
	for _, proj := range projects {
		projDir := s.projectDirPath(proj.Name)
		entries, err := os.ReadDir(projDir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("read project directory: %w", err)
		}

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}
			fp := filepath.Join(projDir, entry.Name())
			todo, err := readTodoFile(fp)
			if err != nil {
				continue
			}
			if strings.HasPrefix(todo.ID.String(), prefix) {
				todo.ProjectName = proj.Name
				matches = append(matches, todo)
			}
		}
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("no todo found with prefix: %s", prefix)
	}
	if len(matches) > 1 {
		ids := make([]string, 0, len(matches))
		for _, m := range matches {
			ids = append(ids, m.ID.String()[:8])
		}
		return nil, fmt.Errorf("ambiguous prefix '%s', matches: %s", prefix, strings.Join(ids, ", "))
	}

	return matches[0], nil
}

// ListTodos returns todos matching the given filter.
//
//nolint:funlen,nestif,gocognit // Query logic is clearer when kept together
func (s *MarkdownStore) ListTodos(filter *TodoFilter) ([]*Todo, error) {
	projects, err := s.readProjects()
	if err != nil {
		return nil, err
	}

	// Build a project ID -> name map
	projectNameMap := make(map[string]string)
	for _, p := range projects {
		projectNameMap[p.ID] = p.Name
	}

	var todos []*Todo
	now := time.Now().UTC()

	for _, proj := range projects {
		// Filter by project
		if filter != nil && filter.ProjectID != nil {
			if proj.ID != filter.ProjectID.String() {
				continue
			}
		}

		projDir := s.projectDirPath(proj.Name)
		entries, err := os.ReadDir(projDir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("read project directory: %w", err)
		}

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}

			fp := filepath.Join(projDir, entry.Name())
			todo, err := readTodoFile(fp)
			if err != nil {
				continue
			}
			todo.ProjectName = proj.Name

			// Apply filters
			if filter != nil {
				if filter.Done != nil && todo.Done != *filter.Done {
					continue
				}
				if filter.Priority != nil && todo.Priority != *filter.Priority {
					continue
				}
				if filter.Tag != nil {
					hasTag := false
					for _, t := range todo.Tags {
						if t == *filter.Tag {
							hasTag = true
							break
						}
					}
					if !hasTag {
						continue
					}
				}
				if filter.Overdue != nil && *filter.Overdue {
					if todo.Done || todo.DueDate == nil || !todo.DueDate.Before(now) {
						continue
					}
				}
			}

			todos = append(todos, todo)
		}
	}

	// Sort by created_at DESC (matching SQLite behavior)
	sort.Slice(todos, func(i, j int) bool {
		return todos[i].CreatedAt.After(todos[j].CreatedAt)
	})

	return todos, nil
}

// UpdateTodo updates an existing todo.
func (s *MarkdownStore) UpdateTodo(todo *Todo) error {
	todo.UpdatedAt = time.Now().UTC()

	path, projectName, err := s.findTodoFile(todo.ID)
	if err != nil {
		return err
	}

	// Read existing to preserve tags if not set on update
	existing, err := readTodoFile(path)
	if err != nil {
		return err
	}
	if todo.Tags == nil {
		todo.Tags = existing.Tags
	}
	todo.ProjectName = projectName

	return writeTodoFile(path, todo)
}

// DeleteTodo removes a todo by ID.
func (s *MarkdownStore) DeleteTodo(id uuid.UUID) error {
	path, _, err := s.findTodoFile(id)
	if err != nil {
		return err
	}

	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to delete todo file: %w", err)
	}
	return nil
}

// MarkTodoDone sets the done status of a todo.
func (s *MarkdownStore) MarkTodoDone(id uuid.UUID, done bool) error {
	path, projectName, err := s.findTodoFile(id)
	if err != nil {
		return err
	}

	todo, err := readTodoFile(path)
	if err != nil {
		return err
	}

	todo.Done = done
	todo.UpdatedAt = time.Now().UTC()
	if done {
		now := time.Now().UTC()
		todo.CompletedAt = &now
	} else {
		todo.CompletedAt = nil
	}
	todo.ProjectName = projectName

	return writeTodoFile(path, todo)
}

// --- Tag operations ---

// GetOrCreateTag retrieves a tag by name, creating it if it doesn't exist.
// For the markdown backend, tags are stored inline in todo frontmatter.
// This method returns a Tag with a synthetic ID derived from the name.
func (s *MarkdownStore) GetOrCreateTag(name string) (*Tag, error) {
	// Check if any todo already has this tag
	tags, err := s.ListTags()
	if err != nil {
		return nil, err
	}

	for _, t := range tags {
		if t.Name == name {
			return t, nil
		}
	}

	// Tag doesn't exist yet - return a synthetic entry
	// The tag will be persisted when it's added to a todo
	return &Tag{
		ID:        syntheticTagID(name),
		Name:      name,
		CreatedAt: time.Now().UTC(),
	}, nil
}

// syntheticTagID generates a stable int64 ID from a tag name.
func syntheticTagID(name string) int64 {
	var h int64
	for _, c := range name {
		h = h*31 + int64(c)
	}
	if h < 0 {
		h = -h
	}
	return h
}

// ListTags returns all unique tags found across all todos, sorted by name.
func (s *MarkdownStore) ListTags() ([]*Tag, error) {
	tagSet := make(map[string]bool)

	projects, err := s.readProjects()
	if err != nil {
		return nil, err
	}

	for _, proj := range projects {
		projDir := s.projectDirPath(proj.Name)
		entries, err := os.ReadDir(projDir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("read project directory: %w", err)
		}

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}
			fp := filepath.Join(projDir, entry.Name())
			todo, err := readTodoFile(fp)
			if err != nil {
				continue
			}
			for _, tag := range todo.Tags {
				tagSet[tag] = true
			}
		}
	}

	var tags []*Tag
	for name := range tagSet {
		tags = append(tags, &Tag{
			ID:        syntheticTagID(name),
			Name:      name,
			CreatedAt: time.Now().UTC(),
		})
	}

	sort.Slice(tags, func(i, j int) bool {
		return tags[i].Name < tags[j].Name
	})

	return tags, nil
}

// DeleteTag removes a tag from all todos.
func (s *MarkdownStore) DeleteTag(name string) error {
	projects, err := s.readProjects()
	if err != nil {
		return err
	}

	found := false
	for _, proj := range projects {
		removed, err := s.removeTagFromProject(proj.Name, name)
		if err != nil {
			return err
		}
		if removed {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("tag not found: %s", name)
	}
	return nil
}

// removeTagFromProject removes a tag from all todos in a project directory.
// Returns true if the tag was found in at least one todo.
func (s *MarkdownStore) removeTagFromProject(projectName, tagName string) (bool, error) {
	projDir := s.projectDirPath(projectName)
	entries, err := os.ReadDir(projDir)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("read project directory: %w", err)
	}

	found := false
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		fp := filepath.Join(projDir, entry.Name())
		todo, readErr := readTodoFile(fp)
		if readErr != nil {
			continue
		}

		var remaining []string
		hasTag := false
		for _, t := range todo.Tags {
			if t == tagName {
				hasTag = true
			} else {
				remaining = append(remaining, t)
			}
		}

		if hasTag {
			found = true
			todo.Tags = remaining
			if writeErr := writeTodoFile(fp, todo); writeErr != nil {
				return false, fmt.Errorf("update todo file after tag removal: %w", writeErr)
			}
		}
	}
	return found, nil
}

// AddTagToTodo adds a tag to a todo.
func (s *MarkdownStore) AddTagToTodo(todoID uuid.UUID, tagName string) error {
	path, _, err := s.findTodoFile(todoID)
	if err != nil {
		return err
	}

	todo, err := readTodoFile(path)
	if err != nil {
		return err
	}

	// Check if already has this tag
	for _, t := range todo.Tags {
		if t == tagName {
			return nil // Already tagged, idempotent
		}
	}

	todo.Tags = append(todo.Tags, tagName)
	sort.Strings(todo.Tags)

	return writeTodoFile(path, todo)
}

// RemoveTagFromTodo removes a tag from a todo.
func (s *MarkdownStore) RemoveTagFromTodo(todoID uuid.UUID, tagName string) error {
	path, _, err := s.findTodoFile(todoID)
	if err != nil {
		return err
	}

	todo, err := readTodoFile(path)
	if err != nil {
		return err
	}

	var remaining []string
	for _, t := range todo.Tags {
		if t != tagName {
			remaining = append(remaining, t)
		}
	}
	todo.Tags = remaining

	return writeTodoFile(path, todo)
}

// GetTagsForTodo returns all tag names for a todo.
func (s *MarkdownStore) GetTagsForTodo(todoID uuid.UUID) ([]string, error) {
	path, _, err := s.findTodoFile(todoID)
	if err != nil {
		return nil, err
	}

	todo, err := readTodoFile(path)
	if err != nil {
		return nil, err
	}

	tags := make([]string, len(todo.Tags))
	copy(tags, todo.Tags)
	sort.Strings(tags)

	return tags, nil
}

// --- Database maintenance ---

// Vacuum is a no-op for the markdown backend.
func (s *MarkdownStore) Vacuum() error {
	return nil
}

// IntegrityCheck validates that project directories and todo files are consistent.
func (s *MarkdownStore) IntegrityCheck() (bool, error) {
	projects, err := s.readProjects()
	if err != nil {
		return false, fmt.Errorf("read projects: %w", err)
	}

	for _, proj := range projects {
		projDir := s.projectDirPath(proj.Name)
		if _, err := os.Stat(projDir); os.IsNotExist(err) {
			return false, nil
		}

		entries, err := os.ReadDir(projDir)
		if err != nil {
			return false, fmt.Errorf("read project directory %q: %w", proj.Name, err)
		}

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}
			fp := filepath.Join(projDir, entry.Name())
			if !isTodoFileValid(fp) {
				return false, nil
			}
		}
	}

	return true, nil
}

// isTodoFileValid checks whether a todo file can be parsed without error.
func isTodoFileValid(path string) bool {
	_, err := readTodoFile(path)
	return err == nil
}
