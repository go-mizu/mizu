package git

import "time"

// ObjectType represents the type of a git object
type ObjectType string

const (
	ObjectBlob   ObjectType = "blob"
	ObjectTree   ObjectType = "tree"
	ObjectCommit ObjectType = "commit"
	ObjectTag    ObjectType = "tag"
)

// FileMode represents git file modes
type FileMode uint32

const (
	ModeFile       FileMode = 0100644 // Regular file
	ModeExecutable FileMode = 0100755 // Executable
	ModeSymlink    FileMode = 0120000 // Symbolic link
	ModeSubmodule  FileMode = 0160000 // Submodule (gitlink)
	ModeDir        FileMode = 0040000 // Directory
)

// String returns the octal string representation of the file mode
func (m FileMode) String() string {
	switch m {
	case ModeFile:
		return "100644"
	case ModeExecutable:
		return "100755"
	case ModeSymlink:
		return "120000"
	case ModeSubmodule:
		return "160000"
	case ModeDir:
		return "040000"
	default:
		return "100644"
	}
}

// ParseFileMode parses a file mode string
func ParseFileMode(s string) FileMode {
	switch s {
	case "100644":
		return ModeFile
	case "100755":
		return ModeExecutable
	case "120000":
		return ModeSymlink
	case "160000":
		return ModeSubmodule
	case "040000":
		return ModeDir
	default:
		return ModeFile
	}
}

// Blob represents a git blob object
type Blob struct {
	SHA     string
	Size    int64
	Content []byte
}

// Commit represents a git commit object
type Commit struct {
	SHA       string
	TreeSHA   string
	Parents   []string
	Author    Signature
	Committer Signature
	Message   string
}

// Signature represents author/committer info
type Signature struct {
	Name  string
	Email string
	When  time.Time
}

// Tree represents a git tree object
type Tree struct {
	SHA       string
	Entries   []TreeEntry
	Truncated bool
}

// TreeEntry represents an entry in a tree
type TreeEntry struct {
	Name string
	Mode FileMode
	Type ObjectType
	SHA  string
	Size int64 // Only for blobs
}

// Tag represents an annotated tag object
type Tag struct {
	SHA        string
	Name       string
	TargetSHA  string
	TargetType ObjectType
	Message    string
	Tagger     Signature
}

// TagRef represents a lightweight tag reference
type TagRef struct {
	Name      string
	CommitSHA string
}

// Ref represents a git reference
type Ref struct {
	Name       string     // Full ref name (refs/heads/main)
	SHA        string     // Object SHA
	ObjectType ObjectType // Type of referenced object
}

// CreateCommitOpts contains options for creating a commit
type CreateCommitOpts struct {
	Message   string
	TreeSHA   string
	Parents   []string
	Author    Signature
	Committer Signature
}

// CreateTreeOpts contains options for creating a tree
type CreateTreeOpts struct {
	BaseSHA string // Optional base tree to modify
	Entries []TreeEntryInput
}

// TreeEntryInput represents input for a tree entry
type TreeEntryInput struct {
	Path    string
	Mode    FileMode
	Type    ObjectType
	SHA     string // Existing object SHA
	Content []byte // Or content to create new blob
}

// CreateTagOpts contains options for creating a tag
type CreateTagOpts struct {
	Name       string
	TargetSHA  string
	TargetType ObjectType
	Message    string
	Tagger     Signature
}

// CloneOptions contains options for cloning
type CloneOptions struct {
	Bare   bool
	Depth  int
	Branch string
}
