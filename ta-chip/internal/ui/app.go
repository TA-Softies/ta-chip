package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"ta-chip/internal/checks"
	"ta-chip/internal/config"
	"ta-chip/internal/submit"
)

type screen int

const (
	screenBanner screen = iota
	screenAutoChecks
	screenRounder
	screenHardware
	screenKeyboardTest
	screenSoftwareReview
	screenDomainTest
	screenRemarks
	screenReview
	screenSubmit
	screenDone
)

type autoCheckResults struct {
	hostname    string
	timeStatus  string
	timeDetail  string
	wallpaper   string
	wallDetail  string
	office      string
	officeDetail string
	teams       string
	teamsDetail string
	browser     string
	browserDetail string
	df          checks.DeepFreezeResult
	domainMem   string
	domainDetail string
	domainLogin bool
	domainLoginDetail string
}

type softwareRow struct {
	label  string
	status string
	detail string
}

type checksCompleteMsg struct{ results autoCheckResults }
type submitDoneMsg struct {
	err error
	row int
}

// Model is the root Bubble Tea model.
type Model struct {
	cfg        *config.Config
	screen     screen
	spinner    spinner.Model

	// auto-check results
	autoResults  autoCheckResults
	checksReady  bool

	// rounder input
	rounderInput textinput.Model

	// hardware check state
	hwIndex    int   // which hardware item
	hwSelected int   // 0=V,1=Y,2=X
	hwResults  []HardwareResult

	// keyboard test
	keyTest KeyTestModel
	// track whether keyboard test result to use for mouse_keyboard
	keyTestStatus string // "V"/"Y"/"X"

	// software review — allows V/Y/X override per row
	swRows      []softwareRow
	swFocusRow  int

	// domain test override
	domainOverride string // "", "V", "Y", "X"

	// remarks
	remarks textarea.Model

	// submit
	submitting bool
	submitErr  error
	submitRow  int

	// window size
	width  int
	height int
}

func New(cfg *config.Config) *Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot

	ri := textinput.New()
	ri.Placeholder = "Your name"
	ri.Focus()
	ri.CharLimit = 64

	rm := textarea.New()
	rm.Placeholder = "Any remarks? (optional — press Tab to skip)"
	rm.SetWidth(60)
	rm.SetHeight(4)
	rm.CharLimit = 512

	return &Model{
		cfg:          cfg,
		screen:       screenBanner,
		spinner:      sp,
		rounderInput: ri,
		hwResults:    make([]HardwareResult, len(hardwareItems)),
		keyTest:      newKeyTestModel(),
		keyTestStatus: "V",
		remarks:      rm,
	}
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, tea.EnterAltScreen)
}

// runAutoChecks executes all automated checks in a goroutine and returns a message.
func runAutoChecks(cfg *config.Config) tea.Cmd {
	return func() tea.Msg {
		r := autoCheckResults{}
		r.hostname = checks.GetHostname()
		r.timeStatus, r.timeDetail = checks.CheckTimeDate(cfg.NTPToleranceSecs)
		r.wallpaper, r.wallDetail = checks.CheckWallpaper(cfg.ExpectedWallpaper)
		r.office, r.officeDetail = checks.CheckOffice()
		r.teams, r.teamsDetail = checks.CheckTeams()
		r.browser, r.browserDetail = checks.CheckBrowser()
		r.df = checks.CheckDeepFreeze()
		r.domainMem, r.domainDetail = checks.CheckDomainMembership(cfg.DomainName)
		r.domainLogin, r.domainLoginDetail = checks.TestDomainLogin(
			cfg.DomainName, cfg.DomainTestUser, cfg.DomainTestPassword)
		return checksCompleteMsg{results: r}
	}
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case tea.MouseMsg:
		if m.screen == screenKeyboardTest {
			switch msg.Button {
			case tea.MouseButtonLeft:
				m.keyTest.handleMousePress("LMB")
			case tea.MouseButtonMiddle:
				m.keyTest.handleMousePress("MMB")
			case tea.MouseButtonRight:
				m.keyTest.handleMousePress("RMB")
			}
		}
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case checksCompleteMsg:
		m.autoResults = msg.results
		m.checksReady = true
		m.buildSoftwareRows()
		return m, nil

	case submitDoneMsg:
		m.submitting = false
		m.submitErr = msg.err
		m.submitRow = msg.row
		m.screen = screenDone
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	// Delegate to sub-components when relevant
	var cmd tea.Cmd
	switch m.screen {
	case screenRounder:
		m.rounderInput, cmd = m.rounderInput.Update(msg)
	case screenRemarks:
		m.remarks, cmd = m.remarks.Update(msg)
	}
	return m, cmd
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.screen {

	// ── Banner ────────────────────────────────────────────
	case screenBanner:
		if msg.Type == tea.KeyEnter || msg.Type == tea.KeySpace {
			m.screen = screenAutoChecks
			return m, runAutoChecks(m.cfg)
		}

	// ── Auto Checks ───────────────────────────────────────
	case screenAutoChecks:
		if m.checksReady && msg.Type == tea.KeyEnter {
			m.screen = screenRounder
		}

	// ── Rounder ───────────────────────────────────────────
	case screenRounder:
		if msg.Type == tea.KeyEnter {
			if strings.TrimSpace(m.rounderInput.Value()) != "" {
				m.screen = screenHardware
				m.hwIndex = 0
				m.hwSelected = 0
			}
			return m, nil
		}
		var cmd tea.Cmd
		m.rounderInput, cmd = m.rounderInput.Update(msg)
		return m, cmd

	// ── Hardware Checks ───────────────────────────────────
	case screenHardware:
		switch msg.Type {
		case tea.KeyUp:
			if m.hwSelected > 0 {
				m.hwSelected--
			}
		case tea.KeyDown:
			if m.hwSelected < 2 {
				m.hwSelected++
			}
		case tea.KeyEnter:
			statuses := []string{"V", "Y", "X"}
			m.hwResults[m.hwIndex] = HardwareResult{
				Item:   hardwareItems[m.hwIndex],
				Status: statuses[m.hwSelected],
			}
			m.hwIndex++
			m.hwSelected = 0
			if m.hwIndex >= len(hardwareItems) {
				m.screen = screenKeyboardTest
			}
		case tea.KeyRunes:
			switch strings.ToUpper(string(msg.Runes)) {
			case "V":
				m.hwSelected = 0
			case "Y":
				m.hwSelected = 1
			case "X":
				m.hwSelected = 2
			}
		}

	// ── Keyboard Test ─────────────────────────────────────
	case screenKeyboardTest:
		if msg.Type == tea.KeyEnter {
			// Determine status based on what was tested
			if m.keyTest.pressedCount() > 10 {
				m.keyTestStatus = "V"
			} else if m.keyTest.pressedCount() > 0 {
				m.keyTestStatus = "Y"
			} else {
				m.keyTestStatus = "X"
			}
			m.screen = screenSoftwareReview
			return m, nil
		}
		m.keyTest.handleKeyPress(msg)

	// ── Software Review ───────────────────────────────────
	case screenSoftwareReview:
		switch msg.Type {
		case tea.KeyUp:
			if m.swFocusRow > 0 {
				m.swFocusRow--
			}
		case tea.KeyDown:
			if m.swFocusRow < len(m.swRows)-1 {
				m.swFocusRow++
			}
		case tea.KeyEnter, tea.KeyTab:
			m.screen = screenDomainTest
		case tea.KeyRunes:
			switch strings.ToUpper(string(msg.Runes)) {
			case "V":
				m.swRows[m.swFocusRow].status = "V"
			case "Y":
				m.swRows[m.swFocusRow].status = "Y"
			case "X":
				m.swRows[m.swFocusRow].status = "X"
			}
		}

	// ── Domain Test ───────────────────────────────────────
	case screenDomainTest:
		switch msg.Type {
		case tea.KeyEnter, tea.KeyTab:
			m.screen = screenRemarks
		case tea.KeyRunes:
			switch strings.ToUpper(string(msg.Runes)) {
			case "V":
				m.domainOverride = "V"
			case "Y":
				m.domainOverride = "Y"
			case "X":
				m.domainOverride = "X"
			}
		}

	// ── Remarks ───────────────────────────────────────────
	case screenRemarks:
		if msg.Type == tea.KeyTab || (msg.Type == tea.KeyEnter && msg.Alt) {
			m.screen = screenReview
			return m, nil
		}
		var cmd tea.Cmd
		m.remarks, cmd = m.remarks.Update(msg)
		return m, cmd

	// ── Review ────────────────────────────────────────────
	case screenReview:
		switch msg.Type {
		case tea.KeyEnter:
			m.screen = screenSubmit
		case tea.KeyEsc:
			m.screen = screenRemarks
		}

	// ── Submit ────────────────────────────────────────────
	case screenSubmit:
		switch {
		case msg.Type == tea.KeyEnter || strings.ToUpper(string(msg.Runes)) == "Y":
			m.submitting = true
			return m, m.doSubmit()
		case msg.Type == tea.KeyEsc || strings.ToUpper(string(msg.Runes)) == "N":
			m.screen = screenReview
		}

	// ── Done ──────────────────────────────────────────────
	case screenDone:
		if msg.Type == tea.KeyEnter || msg.Type == tea.KeyEsc {
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m *Model) buildSoftwareRows() {
	r := m.autoResults
	m.swRows = []softwareRow{
		{"Boot to Windows", "V", "Running"},
		{"Time & Date", r.timeStatus, r.timeDetail},
		{"Wallpaper", r.wallpaper, r.wallDetail},
		{"Microsoft Office", r.office, r.officeDetail},
		{"Microsoft Teams", r.teams, r.teamsDetail},
		{"Browser", r.browser, r.browserDetail},
		{"DeepFreeze Frozen", r.df.Frozen, r.df.Detail},
	}
}

func (m *Model) domainStatus() string {
	if m.domainOverride != "" {
		return m.domainOverride
	}
	if m.autoResults.domainMem == "V" && m.autoResults.domainLogin {
		return "V"
	}
	if m.autoResults.domainMem == "V" {
		return "Y"
	}
	return "X"
}

func (m *Model) doSubmit() tea.Cmd {
	data := submit.InspectionData{
		PCLocation:     m.autoResults.hostname,
		Rounder:        m.rounderInput.Value(),
		ShiftTime:      time.Now().Format("15:04"),
		Display:        m.hwResults[0].Status,
		MouseKeyboard:  m.keyTestStatus,
		KensingtonLock: m.hwResults[1].Status,
		Conduiting:     m.hwResults[2].Status,
		Tidiness:       m.hwResults[3].Status,
		BootToWindows:  m.swRows[0].status,
		TimeDate:       m.swRows[1].status,
		Wallpaper:      m.swRows[2].status,
		Domain:         m.domainStatus(),
		MSOffice:       m.swRows[3].status,
		MSTeams:        m.swRows[4].status,
		Browser:        m.swRows[5].status,
		DFFrozen:       m.swRows[6].status,
		DFPolicy:       m.autoResults.df.PolicyName,
		Remarks:        m.remarks.Value(),
	}
	cfg := m.cfg
	return func() tea.Msg {
		resp, err := submit.Submit(cfg.AppScriptURL, data)
		row := 0
		if resp != nil {
			row = resp.Row
		}
		return submitDoneMsg{err: err, row: row}
	}
}

func (m *Model) View() string {
	switch m.screen {
	case screenBanner:
		return m.viewBanner()
	case screenAutoChecks:
		return m.viewAutoChecks()
	case screenRounder:
		return m.viewRounder()
	case screenHardware:
		return m.viewHardware()
	case screenKeyboardTest:
		return m.viewKeyboardTest()
	case screenSoftwareReview:
		return m.viewSoftwareReview()
	case screenDomainTest:
		return m.viewDomainTest()
	case screenRemarks:
		return m.viewRemarks()
	case screenReview:
		return m.viewReview()
	case screenSubmit:
		return m.viewSubmit()
	case screenDone:
		return m.viewDone()
	}
	return ""
}

func (m *Model) viewBanner() string {
	return fmt.Sprintf("%s\n\n%s  %s\n\n  %s",
		styleBanner.Render(banner),
		styleDim.Render("  Version:"),
		styleDim.Render("(loaded from config)"),
		styleDim.Render("Press Enter or Space to start →"),
	)
}

func (m *Model) viewAutoChecks() string {
	if !m.checksReady {
		return fmt.Sprintf("\n\n  %s  %s\n",
			m.spinner.View(),
			styleLabel.Render("Running automated checks..."),
		)
	}
	r := m.autoResults
	lines := []string{
		styleHeader.Render("  Automated Checks Complete"),
		"",
		fmt.Sprintf("  %-22s %s  %s", "PC Location", statusStyle("V"), styleLabel.Render(r.hostname)),
		fmt.Sprintf("  %-22s %s  %s", "Time & Date", statusStyle(r.timeStatus), styleDim.Render(r.timeDetail)),
		fmt.Sprintf("  %-22s %s  %s", "Wallpaper", statusStyle(r.wallpaper), styleDim.Render(r.wallDetail)),
		fmt.Sprintf("  %-22s %s  %s", "Office", statusStyle(r.office), styleDim.Render(r.officeDetail)),
		fmt.Sprintf("  %-22s %s  %s", "Teams", statusStyle(r.teams), styleDim.Render(r.teamsDetail)),
		fmt.Sprintf("  %-22s %s  %s", "Browser", statusStyle(r.browser), styleDim.Render(r.browserDetail)),
		fmt.Sprintf("  %-22s %s  %s", "DeepFreeze", statusStyle(r.df.Frozen), styleDim.Render(r.df.Detail)),
		fmt.Sprintf("  %-22s %s  %s", "Domain ("+m.cfg.DomainName+")", statusStyle(r.domainMem), styleDim.Render(r.domainDetail)),
		"",
		styleDim.Render("  Press Enter to continue →"),
	}
	return strings.Join(lines, "\n")
}

func (m *Model) viewRounder() string {
	return fmt.Sprintf("\n\n  %s\n\n  %s\n\n  %s",
		styleHeader.Render("Who is doing rounds today?"),
		m.rounderInput.View(),
		styleDim.Render("Enter to continue"),
	)
}

func (m *Model) viewHardware() string {
	item := hardwareItems[m.hwIndex]
	progress := styleDim.Render(fmt.Sprintf("  Hardware check %d of %d", m.hwIndex+1, len(hardwareItems)))
	return fmt.Sprintf("\n  %s\n\n%s", progress, renderHardwareScreen(item, m.hwSelected))
}

func (m *Model) viewKeyboardTest() string {
	return "\n" + renderKeyboardTestScreen(m.keyTest)
}

func (m *Model) viewSoftwareReview() string {
	var sb strings.Builder
	sb.WriteString(styleHeader.Render("  Software Check Review") + "\n")
	sb.WriteString(styleDim.Render("  ↑↓ to navigate  •  V / Y / X to override  •  Enter to continue") + "\n\n")

	for i, row := range m.swRows {
		cursor := "  "
		label := row.label
		if i == m.swFocusRow {
			cursor = styleSelected.Render("▶")
			label = styleLabel.Render(label)
		}
		sb.WriteString(fmt.Sprintf("%s %-24s %s  %s\n",
			cursor,
			label,
			statusStyle(row.status),
			styleDim.Render(row.detail),
		))
	}
	return sb.String()
}

func (m *Model) viewDomainTest() string {
	r := m.autoResults
	loginResult := "X  Failed"
	if r.domainLogin {
		loginResult = "V  Success"
	}

	override := ""
	if m.domainOverride != "" {
		override = "  Override: " + statusStyle(m.domainOverride)
	}

	return fmt.Sprintf("\n  %s\n\n"+
		"  %-24s %s  %s\n"+
		"  %-24s %s\n\n"+
		"  %s\n\n"+
		"  %s",
		styleHeader.Render("Domain Check"),
		"Domain Membership", statusStyle(r.domainMem), styleDim.Render(r.domainDetail),
		"Login Test ("+m.cfg.DomainTestUser+")", styleDim.Render(loginResult),
		styleDim.Render("V / Y / X to override"+override),
		styleDim.Render("Enter to continue →"),
	)
}

func (m *Model) viewRemarks() string {
	return fmt.Sprintf("\n\n  %s\n\n  %s\n\n  %s",
		styleHeader.Render("Remarks"),
		m.remarks.View(),
		styleDim.Render("Alt+Enter or Tab to continue (or just Tab to skip)"),
	)
}

func (m *Model) viewReview() string {
	r := m.autoResults

	sw := func(i int) string { return m.swRows[i].status }
	hw := func(i int) string {
		if i < len(m.hwResults) {
			return m.hwResults[i].Status
		}
		return "-"
	}

	lines := []string{
		styleHeader.Render("  Summary — Review before submitting"),
		"",
		fmt.Sprintf("  %-26s %s", "PC Location", styleLabel.Render(r.hostname)),
		fmt.Sprintf("  %-26s %s", "Rounder", styleLabel.Render(m.rounderInput.Value())),
		fmt.Sprintf("  %-26s %s", "Shift Time", styleDim.Render(time.Now().Format("15:04"))),
		"",
		fmt.Sprintf("  %-26s %s", "Display", statusStyle(hw(0))),
		fmt.Sprintf("  %-26s %s  %s", "Mouse & Keyboard", statusStyle(m.keyTestStatus), styleDim.Render(fmt.Sprintf("(%d keys tested)", m.keyTest.pressedCount()))),
		fmt.Sprintf("  %-26s %s", "Kensington Lock", statusStyle(hw(1))),
		fmt.Sprintf("  %-26s %s", "Conduiting", statusStyle(hw(2))),
		fmt.Sprintf("  %-26s %s", "Tidiness", statusStyle(hw(3))),
		"",
		fmt.Sprintf("  %-26s %s", "Boot to Windows", statusStyle(sw(0))),
		fmt.Sprintf("  %-26s %s  %s", "Time & Date", statusStyle(sw(1)), styleDim.Render(r.timeDetail)),
		fmt.Sprintf("  %-26s %s", "Wallpaper", statusStyle(sw(2))),
		fmt.Sprintf("  %-26s %s", "Domain ("+m.cfg.DomainName+")", statusStyle(m.domainStatus())),
		fmt.Sprintf("  %-26s %s", "Microsoft Office", statusStyle(sw(3))),
		fmt.Sprintf("  %-26s %s", "Microsoft Teams", statusStyle(sw(4))),
		fmt.Sprintf("  %-26s %s", "Browser", statusStyle(sw(5))),
		fmt.Sprintf("  %-26s %s", "DeepFreeze Frozen", statusStyle(sw(6))),
		fmt.Sprintf("  %-26s %s", "DeepFreeze Policy", styleDim.Render(r.df.PolicyName)),
		"",
		fmt.Sprintf("  %-26s %s", "Remarks", styleDim.Render(m.remarks.Value())),
		"",
		styleDim.Render("  Enter to submit  •  Esc to go back"),
	}
	return strings.Join(lines, "\n")
}

func (m *Model) viewSubmit() string {
	if m.submitting {
		return fmt.Sprintf("\n\n  %s  %s", m.spinner.View(), styleLabel.Render("Submitting to Google Sheets..."))
	}
	return fmt.Sprintf("\n\n  %s\n\n  %s",
		styleHeader.Render("Submit results to Google Sheets?"),
		styleDim.Render("Y / Enter to confirm  •  N / Esc to go back"),
	)
}

func (m *Model) viewDone() string {
	if m.submitErr != nil {
		return fmt.Sprintf("\n\n  %s\n\n  %s\n\n  %s",
			styleError.Render("Submission failed"),
			styleDim.Render(m.submitErr.Error()),
			styleDim.Render("Press Enter to exit"),
		)
	}
	return fmt.Sprintf("\n\n  %s\n\n  %s\n\n  %s",
		styleSuccess.Render("Done! Results submitted to Google Sheets."),
		styleDim.Render(fmt.Sprintf("Row %d written.", m.submitRow)),
		styleDim.Render("Press Enter to exit"),
	)
}
