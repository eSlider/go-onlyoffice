# go-onlyoffice

Go client library for the [OnlyOffice](https://www.onlyoffice.com/) Project Management API.

## Features

- Token-based authentication with automatic renewal
- Project CRUD operations (create, read, update, delete)
- Task management (create, update, list, filter)
- Milestone management
- User listing
- Query parameter serialization via struct tags

## Installation

```bash
go get github.com/eslider/go-onlyoffice
```

## Usage

```go
package main

import (
    "fmt"
    "log"

    onlyoffice "github.com/eslider/go-onlyoffice"
)

func main() {
    client := onlyoffice.NewClient(onlyoffice.Credentials{
        Url:      "https://your-onlyoffice.example.com",
        User:     "admin@example.com",
        Password: "your-password",
    })

    // List projects
    projects, err := client.GetProjects()
    if err != nil {
        log.Fatal(err)
    }
    for _, p := range projects {
        fmt.Printf("Project: %s (ID: %d)\n", *p.Title, *p.ID)
    }
}
```

Or load credentials from environment variables:

```go
client := onlyoffice.NewClient(onlyoffice.GetEnvironmentCredentials())
```

## Environment Variables

| Variable | Description |
|---|---|
| `ONLYOFFICE_URL` | OnlyOffice instance URL |
| `ONLYOFFICE_USER` | Login email or username |
| `ONLYOFFICE_PASS` | Password |

## API Reference

### Client

- `NewClient(credentials)` - Create a new client
- `GetEnvironmentCredentials()` - Load credentials from env vars
- `Auth(credentials)` - Authenticate and get token
- `Query(request, result)` - Execute an API request

### Projects

- `GetProjects()` - List all projects
- `CreateProject(req)` - Create a new project
- `UpdateProject(req)` - Update a project
- `DeleteProject(id)` - Delete a project
- `GetProjectMilestones(project)` - Get milestones for a project

### Tasks

- `GetTasks(req)` - List tasks with filtering
- `CreateProjectTask(req)` - Create a new task
- `UpdateProjectTask(req)` - Update a task

### Users

- `GetUsers()` - List all users

## License

[MIT](LICENSE)
