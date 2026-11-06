package mainwindow

import (
	"context"
	"log"
	"sync"

	"github.com/adrg/xdg"
	"github.com/lmorg/ttyphoon/types"
	"github.com/lmorg/ttyphoon/utils/lsp"
	"github.com/lmorg/ttyphoon/utils/notes"
	"github.com/lmorg/ttyphoon/utils/syntaxcompletion"
	"github.com/lmorg/ttyphoon/window/elements/element_table/tablecore"
)

// App struct
type WApp struct {
	ctx                   context.Context
	uiRuntime             types.DesktopRuntime
	mdBaseDir             string
	mdBaseDirMu           sync.RWMutex
	projRoot              string
	groupName             string
	usrNotesDir           string
	homeDir               string
	globalNotes           string
	visible               bool
	visibleMu             sync.Mutex
	notesKills            map[string]func()
	notesStickies         map[string]types.Notification
	notesMu               sync.Mutex
	notesListMu           sync.Mutex
	notesListCancel       context.CancelFunc
	notesListSeq          uint64
	notesContext          notes.FileContext
	notesFileIndex        *notes.FileIndex
	notesIndexActivated   bool
	notesIndexClosed      bool
	notesIndexUnsubscribe func()
	gitStatusMu           sync.Mutex
	gitStatusCache        map[string]gitStatusCacheEntry
	gitStatusRequest      map[string]uint64
	gitWatchMu            sync.Mutex
	gitWatchCancel        context.CancelFunc
	gitWatchRoot          string
	gitWatchRequest       uint64
	lspStartErrMu         sync.Mutex
	lspStartErrs          map[string]string
	lspManager            *lsp.Manager
	lspDocs               *lsp.DocumentStore
	typosDocs             *lsp.DocumentStore
	typosUnavail          map[string]bool
	typosMu               sync.Mutex
	syntaxEngine          *syntaxcompletion.Engine
	notesTables           *tablecore.Registry
}

// NewApp creates a new App application struct
func NewWailsApp() *WApp {
	a := &WApp{
		visible:          true,
		notesKills:       map[string]func(){},
		notesStickies:    map[string]types.Notification{},
		lspStartErrs:     map[string]string{},
		homeDir:          xdg.Home,
		projRoot:         "",
		globalNotes:      notes.DirGlobal(),
		lspManager:       lsp.NewManager(),
		lspDocs:          lsp.NewDocumentStore(),
		typosDocs:        lsp.NewDocumentStore(),
		typosUnavail:     map[string]bool{},
		notesTables:      tablecore.NewRegistry(),
		gitStatusCache:   map[string]gitStatusCacheEntry{},
		gitStatusRequest: map[string]uint64{},
	}

	engine, err := syntaxcompletion.NewDefaultEngine()
	if err != nil {
		log.Printf("syntax completion: engine init failed: %v", err)
	} else {
		a.syntaxEngine = engine
	}

	return a
}
