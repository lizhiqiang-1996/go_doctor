package types

type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

type Category string

const (
	CategoryErrorHandling Category = "Error Handling"
	CategoryConcurrency   Category = "Concurrency"
	CategoryPerformance   Category = "Performance"
	CategorySecurity      Category = "Security"
	CategoryCodeStyle     Category = "Code Style"
	CategoryCorrectness   Category = "Correctness"
	CategoryDeadCode      Category = "Dead Code"
	CategoryAICode        Category = "AI Quality"
)

type Diagnostic struct {
	FilePath string   `json:"filePath"`
	Plugin   string   `json:"plugin"`
	Rule     string   `json:"rule"`
	Severity Severity `json:"severity"`
	Message  string   `json:"message"`
	Help     string   `json:"help"`
	Line     int      `json:"line"`
	Column   int      `json:"column"`
	Category Category `json:"category"`
}

type Framework string

const (
	FrameworkGin     Framework = "gin"
	FrameworkEcho    Framework = "echo"
	FrameworkFiber   Framework = "fiber"
	FrameworkChi     Framework = "chi"
	FrameworkStdLib  Framework = "stdlib"
	FrameworkGRPC    Framework = "grpc"
	FrameworkKitex   Framework = "kitex"
	FrameworkHertz   Framework = "hertz"
	FrameworkUnknown Framework = "unknown"
)

type ProjectInfo struct {
	RootDirectory   string    `json:"rootDirectory"`
	ProjectName     string    `json:"projectName"`
	GoVersion       string    `json:"goVersion"`
	GoMajorVersion  int       `json:"goMajorVersion"`
	GoMinorVersion  int       `json:"goMinorVersion"`
	Framework       Framework `json:"framework"`
	HasGoMod        bool      `json:"hasGoMod"`
	HasGoSum        bool      `json:"hasGoSum"`
	SourceFileCount int       `json:"sourceFileCount"`
	ModulePath      string    `json:"modulePath"`
}

type ScoreResult struct {
	Score int    `json:"score"`
	Label string `json:"label"`
}

type ScanResult struct {
	Diagnostics   []Diagnostic `json:"diagnostics"`
	Score         *ScoreResult `json:"score"`
	SkippedChecks []string     `json:"skippedChecks"`
	Project       ProjectInfo  `json:"project"`
	ElapsedMs     int64        `json:"elapsedMilliseconds"`
	DiffInfo      *DiffInfo    `json:"diffInfo,omitempty"`
	CommitInfo    *CommitInfo  `json:"commitInfo,omitempty"`
}

type DiffInfo struct {
	BaseBranch    string                 `json:"baseBranch"`
	CurrentBranch string                 `json:"currentBranch"`
	ChangedFiles  []string               `json:"changedFiles"`
	AddedFiles    []string               `json:"addedFiles"`
	ModifiedFiles []string               `json:"modifiedFiles"`
	DeletedFiles  []string               `json:"deletedFiles"`
	ChangedLines  map[string][]LineRange `json:"changedLines,omitempty"`
}

type LineRange struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

type CommitInfo struct {
	CommitHash   string                 `json:"commitHash"`
	Author       string                 `json:"author"`
	Message      string                 `json:"message"`
	ChangedFiles []string               `json:"changedFiles"`
	ChangedLines map[string][]LineRange `json:"changedLines,omitempty"`
}

type ScanOptions struct {
	Lint      bool
	DeadCode  bool
	Verbose   bool
	ScoreOnly bool
	JSON      bool
	DiffBase  string
	Commit    string
}

type Config struct {
	Ignore   *IgnoreConfig `json:"ignore,omitempty"`
	Lint     *bool         `json:"lint,omitempty"`
	DeadCode *bool         `json:"deadCode,omitempty"`
	Verbose  *bool         `json:"verbose,omitempty"`
}

type IgnoreConfig struct {
	Rules []string `json:"rules,omitempty"`
	Files []string `json:"files,omitempty"`
}
