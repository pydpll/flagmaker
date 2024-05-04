package flagmaker

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

const maxWidth = 80

type errMsg error

type model struct {
	form          *huh.Form
	viewport      viewport.Model
	vReady        bool
	documentation *string
	err           error
	width         int
	lg            *lipgloss.Renderer
	styles        *styles
}

func initialModel(f *huh.Form, d *string) model {
	m := model{
		form: f.
			WithWidth(45).
			WithShowHelp(false).
			WithShowErrors(false),
		documentation: d,
		width:         maxWidth,
		lg:            lipgloss.DefaultRenderer(),
	}
	m.styles = newStyles(m.lg)
	return m
}

func (m model) Init() tea.Cmd {
	one := m.form.Init()
	two := m.viewport.Init()
	return tea.Sequence(one, two)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {

	case tea.KeyMsg:
		if key.Matches(msg, Km.quitKeys) {
			return m, tea.Quit
		} else if key.Matches(msg, Km.up) || key.Matches(msg, Km.down) {
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}
		aform, cmd := m.form.Update(msg)
		if f, ok := aform.(*huh.Form); ok {
			m.form = f
		}
		return m, cmd
	case errMsg:
		m.err = msg
		return m, nil
	case tea.WindowSizeMsg:
		m.width = min(msg.Width, maxWidth) - m.styles.Base.GetHorizontalFrameSize()
		headerHeight := lipgloss.Height(m.headerView())
		footerHeight := lipgloss.Height(m.footerView())
		verticalMarginHeight := headerHeight + footerHeight
		if !m.vReady {
			m.viewport = viewport.New(msg.Width, msg.Height-verticalMarginHeight-8)
			m.viewport.YPosition = headerHeight
			m.viewport.SetContent(*m.documentation)
			m.vReady = true
			m.viewport.YPosition = headerHeight + 1
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - verticalMarginHeight
		}

	default:
		frm, cmd := m.form.Update(msg)
		m.form = frm.(*huh.Form)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	if m.form.State == huh.StateCompleted {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(tea.QuitMsg{})
		cmds = append(cmds, tea.Quit, cmd)
	}
	return m, tea.Sequence(cmds...)
}

func (m model) View() string {
	v := strings.TrimSuffix(m.form.View(), "\n\n")
	form := m.lg.NewStyle().Margin(1, 0).Render(v)

	const docWidth = 28
	docMarginLeft := m.width - docWidth - lipgloss.Width(form) - m.styles.Documentation.GetMarginRight()
	m.viewport.Style.MarginLeft(docMarginLeft)
	m.viewport.Style.Height(lipgloss.Height(form))
	m.viewport.Style.Width(docWidth)

	if !m.vReady {
		return "\n  Initializing..."
	}
	docPanel := fmt.Sprintf("%s\n%s\n%s", m.headerView(), m.viewport.View(), m.footerView())
	errors := m.form.Errors()
	header := m.appBoundaryView("Options and notes")
	if len(errors) > 0 {
		header = m.appErrorBoundaryView(m.errorView())
	}
	body := lipgloss.JoinHorizontal(lipgloss.Top, form, docPanel)

	footer := m.appBoundaryView(m.form.Help().ShortHelpView(m.form.KeyBinds()))
	if len(errors) > 0 {
		footer = m.appErrorBoundaryView("")
	}

	return m.styles.Base.Render(header + "\n" + body + "\n\n" + footer)
}

func New(f *huh.Form, d *string) *tea.Program {
	return tea.NewProgram(initialModel(f, d))
}
