package inbox

import (
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/opik/miau/internal/config"
	"github.com/opik/miau/internal/image"
	"github.com/opik/miau/internal/imap"
	"github.com/opik/miau/internal/ports"
	"github.com/opik/miau/internal/storage"
)

type state int

const (
	stateInitDB state = iota
	stateConnecting
	stateLoadingFolders
	stateSyncing
	stateLoadingEmails
	stateReady
	stateViewingThread // Nova: visualizando thread
	stateError
	stateNeedsAppPassword
)

// Model is the main TUI model for the inbox
type Model struct {
	width         int
	height        int
	state         state
	// Panel widths (resizable)
	foldersWidth    int  // Largura do painel de pastas (default 25)
	resizingPanel   bool // Se está arrastando para redimensionar
	err           error
	account       *config.Account
	dbAccount     *storage.Account
	dbFolder      *storage.Folder
	client        *imap.Client
	app           ports.App // Application for centralized services
	mailboxes     []imap.Mailbox
	emails        []storage.EmailSummary
	selectedEmail int
	selectedBox   int
	currentBox    string
	showFolders   bool
	passwordInput textinput.Model
	retrying      bool
	syncStatus    string
	totalEmails   int
	syncedEmails  int
	// AI panel
	showAI         bool
	aiInput        textinput.Model
	aiResponse     string
	aiLastQuestion string
	aiLoading      bool
	aiScrollOffset int
	aiEmailContext *storage.EmailSummary // Email selecionado para contexto (quando usa Shift+A)
	aiEmailBody    string                // Corpo do email para contexto
	// Spinner
	spinner spinner.Model
	// Email viewer
	showViewer     bool
	viewerViewport viewport.Model
	viewerEmail    *storage.EmailSummary
	viewerLoading  bool
	// Compose
	showCompose           bool
	composeTo             textinput.Model
	composeSubject        textinput.Model
	composeBody           viewport.Model
	composeBodyText       string
	composeFocus          int // 0=To, 1=Subject, 2=Body, 3=Classification
	composeSending        bool
	composeReplyTo        *storage.EmailSummary
	composeClassification int // índice em smtp.Classifications
	// Debug
	debugMode   bool
	debugLogs   []string
	debugScroll int
	// Bounce monitoring
	sentEmails []SentEmailTracker
	alerts     []Alert
	showAlert  bool
	// Drafts
	showDrafts      bool
	drafts          []storage.Draft
	selectedDraft   int
	editingDraftID  *int64         // Se estamos editando um draft existente
	scheduledDraft  *storage.Draft // Draft atualmente agendado (para overlay de undo)
	showUndoOverlay bool
	// Batch operation filter mode
	filterActive      bool                    // Modo de filtro ativo (preview de batch op)
	filterDescription string                  // "Arquivar 15 emails de zaqueu@..."
	pendingBatchOp    *storage.PendingBatchOp // Operação pendente
	originalEmails    []storage.EmailSummary  // Emails originais antes do filtro
	// Fuzzy search
	searchMode    bool                   // Modo de busca ativo
	searchInput   textinput.Model        // Input de busca
	searchResults []storage.EmailSummary // Resultados da busca
	searchQuery   string                 // Query atual (para highlight)
	// Settings
	showSettings      bool                       // Menu de configurações aberto
	settingsSelection int                        // Item selecionado no menu/lista
	settingsTab       int                        // Tab atual: 0=Folders, 1=Sync, 2=Indexer, 3=About
	settingsFolders   []SettingsFolder           // Lista de pastas para configuração
	settingsSyncFolders []string                 // Pastas selecionadas para sync
	indexState        *storage.ContentIndexState // Estado do indexador
	indexerRunning    bool                       // Se o indexador está ativo nesta sessão
	// Image preview
	showImagePreview  bool                // Overlay de preview de imagem
	imageAttachments  []Attachment        // Imagens extraídas do email atual
	selectedImage     int                 // Índice da imagem selecionada
	imageRenderOutput string              // Output renderizado para display
	imageCapabilities *image.Capabilities // Capabilities detectadas
	imageLoading      bool                // Se está carregando/renderizando
	// Analytics
	showAnalytics    bool           // Painel de analytics visível
	analyticsData    *AnalyticsData // Dados de analytics
	analyticsPeriod  string         // Período atual: "7d", "30d", "90d", "all"
	analyticsLoading bool           // Se está carregando
	// Auto-refresh
	autoRefreshInterval time.Duration // Intervalo de auto-refresh (default 5min)
	autoRefreshStart    time.Time     // Quando começou o timer atual
	autoRefreshEnabled  bool          // Se o auto-refresh está habilitado
	// New email notification
	newEmailCount    int       // Quantidade de novos emails no último sync
	newEmailShowTime time.Time // Quando mostrar até (para fade out)
	// Attachments panel
	showAttachments    bool         // Painel de anexos visível
	viewerAttachments  []Attachment // Todos os anexos do email sendo visualizado
	selectedAttachment int          // Índice do anexo selecionado
	attachmentsLoading bool         // Se está carregando anexos
	// Thread view
	threadView   interface{} // thread.Model (imported dynamically to avoid cycle)
	previousState state       // Estado anterior antes de abrir thread
}

// AnalyticsData contém todos os dados de analytics para o TUI
type AnalyticsData struct {
	Overview     *storage.AnalyticsOverviewResult
	TopSenders   []storage.SenderStatsResult
	Hourly       []storage.HourlyStatsResult
	Weekday      []storage.WeekdayStatsResult
	ResponseTime *storage.ResponseStatsResult
}

// SentEmailTracker rastreia emails enviados para detectar bounces
type SentEmailTracker struct {
	MessageID    string
	To           string
	Subject      string
	SentAt       time.Time
	MonitorUntil time.Time
}

// Alert representa um alerta para o usuário
type Alert struct {
	Type      string // "bounce", "error", "warning"
	Title     string
	Message   string
	Timestamp time.Time
	EmailTo   string
	Dismissed bool
}

// Attachment representa um anexo de email
type Attachment struct {
	Filename    string
	ContentType string
	ContentID   string // Para imagens inline (cid:xxx)
	Size        int64
	Data        []byte
	IsInline    bool
}

// SettingsFolder representa uma pasta na configuração de sync
type SettingsFolder struct {
	Name     string
	Selected bool
}

// Constants
const autoRefreshInterval = 60 * time.Second // 1 minuto
