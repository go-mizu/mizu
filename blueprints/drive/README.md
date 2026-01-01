# Drive

A full-featured cloud file storage application built with Go and the Mizu framework. Drive replicates 100% of the functionality found in Google Drive, Box, and Dropbox.

## Features

### File Management
- **Upload**: Single file and chunked/resumable uploads for large files
- **Download**: Direct download with range request support
- **Preview**: Thumbnails for images and documents
- **Versioning**: Full version history with restore capability
- **Organization**: Star, tag, and search files

### Folder Management
- Nested folder hierarchy
- Move, copy, rename operations
- Color-coded folders
- Folder tree navigation

### Sharing & Permissions
- Share with specific users (viewer/commenter/editor)
- Public share links with:
  - Password protection
  - Expiration dates
  - Download limits
- Permission inheritance for folders

### Collaboration
- Comments on files
- Activity log
- File locking
- Real-time notifications

### Storage
- Per-user quota (default 15 GB)
- Usage statistics and breakdown
- Automatic trash cleanup (30 days)

## Quick Start

```bash
# Initialize database
make init

# Seed demo data
make seed

# Start server
make serve
```

The server will start at http://localhost:8080

**Demo credentials:**
- Username: `demo` / Password: `demo1234`
- Username: `admin` / Password: `admin1234`

## CLI Commands

```bash
# Start server
drive serve --addr :8080 --data ~/.drive

# Initialize database
drive init --data ~/.drive

# Seed demo data
drive seed --data ~/.drive
```

## Configuration

| Environment Variable | Default | Description |
|---------------------|---------|-------------|
| ADDR | :8080 | HTTP listen address |
| DATA_DIR | ~/.drive | Data directory |
| DEV | false | Development mode |

## API Reference

### Authentication

```bash
# Register
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"john","email":"john@example.com","password":"secret123"}'

# Login
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"john","password":"secret123"}' \
  -c cookies.txt

# Get current user
curl http://localhost:8080/api/v1/auth/me -b cookies.txt
```

### Files

```bash
# Upload file
curl -X POST http://localhost:8080/api/v1/files \
  -F "file=@document.pdf" \
  -F "folder_id=FOLDER_ID" \
  -b cookies.txt

# Download file
curl http://localhost:8080/api/v1/files/FILE_ID/download \
  -b cookies.txt -o downloaded.pdf

# Update file
curl -X PATCH http://localhost:8080/api/v1/files/FILE_ID \
  -H "Content-Type: application/json" \
  -d '{"name":"new-name.pdf"}' \
  -b cookies.txt

# Delete file (trash)
curl -X DELETE http://localhost:8080/api/v1/files/FILE_ID \
  -b cookies.txt
```

### Chunked Upload

```bash
# Create upload session
curl -X POST http://localhost:8080/api/v1/uploads \
  -H "Content-Type: application/json" \
  -d '{"filename":"large.zip","size":1073741824}' \
  -b cookies.txt

# Upload chunks (parallel)
curl -X PUT http://localhost:8080/api/v1/uploads/UPLOAD_ID/chunk/0 \
  --data-binary @chunk_0 -b cookies.txt

# Complete upload
curl -X POST http://localhost:8080/api/v1/uploads/UPLOAD_ID/complete \
  -b cookies.txt
```

### Folders

```bash
# Create folder
curl -X POST http://localhost:8080/api/v1/folders \
  -H "Content-Type: application/json" \
  -d '{"name":"Documents","parent_id":"PARENT_ID"}' \
  -b cookies.txt

# Get folder contents
curl http://localhost:8080/api/v1/folders/FOLDER_ID/contents \
  -b cookies.txt

# Move folder
curl -X POST http://localhost:8080/api/v1/folders/FOLDER_ID/move \
  -H "Content-Type: application/json" \
  -d '{"parent_id":"NEW_PARENT_ID"}' \
  -b cookies.txt
```

### Sharing

```bash
# Share with user
curl -X POST http://localhost:8080/api/v1/shares \
  -H "Content-Type: application/json" \
  -d '{"item_id":"FILE_ID","item_type":"file","shared_with":"USER_ID","permission":"editor"}' \
  -b cookies.txt

# Create share link
curl -X POST http://localhost:8080/api/v1/share-links \
  -H "Content-Type: application/json" \
  -d '{"item_id":"FILE_ID","item_type":"file","permission":"viewer","password":"secret"}' \
  -b cookies.txt

# Access share link
curl http://localhost:8080/s/TOKEN
```

## Project Structure

```
drive/
├── cmd/drive/          # CLI entry point
├── cli/                # CLI commands
├── app/web/            # HTTP server
│   └── handler/        # Request handlers
├── feature/            # Business logic
│   ├── accounts/       # User accounts
│   ├── files/          # File management
│   ├── folders/        # Folder management
│   └── shares/         # Sharing & permissions
├── store/duckdb/       # Database layer
├── storage/local/      # File storage
├── pkg/                # Utilities
│   ├── ulid/           # ID generation
│   ├── password/       # Password hashing
│   ├── crypto/         # Tokens
│   ├── hash/           # Checksums
│   └── mime/           # MIME detection
└── assets/             # Static files & templates
```

## Data Models

### File
```go
type File struct {
    ID             string
    OwnerID        string
    FolderID       string
    Name           string
    Path           string
    Size           int64
    MimeType       string
    ChecksumSHA256 string
    Starred        bool
    Trashed        bool
    Locked         bool
    VersionCount   int
    CreatedAt      time.Time
    UpdatedAt      time.Time
}
```

### Folder
```go
type Folder struct {
    ID        string
    OwnerID   string
    ParentID  string
    Name      string
    Path      string
    Depth     int
    Color     string
    IsRoot    bool
    Starred   bool
    Trashed   bool
    CreatedAt time.Time
}
```

### Share
```go
type Share struct {
    ID         string
    ItemID     string
    ItemType   string
    OwnerID    string
    SharedWith string
    Permission string  // viewer, commenter, editor
    CreatedAt  time.Time
}
```

### ShareLink
```go
type ShareLink struct {
    ID            string
    ItemID        string
    ItemType      string
    Token         string
    Permission    string
    HasPassword   bool
    ExpiresAt     *time.Time
    DownloadLimit *int
    DownloadCount int
    AllowDownload bool
    Disabled      bool
}
```

## Database

Uses DuckDB for embedded storage. The database file is stored at `{data_dir}/drive.duckdb`.

## File Storage

Files are stored in the local filesystem:

```
{data_dir}/
├── drive.duckdb          # Database
├── files/                # File storage
│   └── {owner_id}/
│       └── {file_id}/
│           ├── current   # Current version
│           └── versions/
│               ├── 1
│               └── 2
├── thumbnails/           # Thumbnail cache
└── temp/uploads/         # Chunked upload staging
```

## Development

```bash
# Run tests
make test

# Build binary
make build

# Clean up
make clean
```

## License

MIT
