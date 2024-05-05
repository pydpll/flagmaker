// This library offers quick form handling for user facing workflows. Using the power of huh.Form, it makes it easy to display forms to the user and gather input. The forms must have embedded pointer values for each field to have access to the result. Calling the Interact function requires a function that generates the form, a pointer to the documentation string, and a closure that checks if the saved values are valid.
package flagmaker

/*
TODO: Set colours
*/
import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/pydpll/errorutils"
	"github.com/sirupsen/logrus"
)

const maxWidth = 80

type errMsg error

type model struct {
	// user sets these
	documentation *string
	form          *huh.Form
	err           *error
	// internals
	viewport viewport.Model
	width    int
	lg       *lipgloss.Renderer
	styles   *styles

	// state messages
	vReady   bool
	quitting bool
}

func (m model) isQuitting() bool {
	return m.quitting
}

func initialModel(f *huh.Form, d *string, e *error) model {
	m := model{
		form: f.
			WithWidth(45).
			WithShowHelp(false).
			WithShowErrors(false),
		documentation: d,
		width:         maxWidth,
		lg:            lipgloss.DefaultRenderer(),
		err:           e,
	}
	m.styles = newStyles(m.lg)
	return m
}

func (m model) Init() tea.Cmd {
	one := m.form.Init()
	two := m.viewport.Init()
	return tea.Sequence(one, two)
}

// Update handles user input and updates the model accordingly.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	logrus.Debug("msg: ", msg)
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(msg, km.quitKeys) {
			m.quitting = true
			return m, tea.Quit
		}
		// viewport only keys
		if key.Matches(msg, km.up) || key.Matches(msg, km.down) {
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}

		// Trigger an interrupt is communicated to the calling program.
		if key.Matches(msg, km.interruptKey) {
			return m, func() tea.Msg { return errMsg(fmt.Errorf("interrupt")) }
		}

		// Form handles all other inputs.
		aform, cmd := m.form.Update(msg)
		if f, ok := aform.(*huh.Form); ok {
			m.form = f
		}
		return m, cmd

	case errMsg:
		*m.err = msg.(error)
		if msg.Error() == "interrupt" {
			m.quitting = true
			return m, tea.Quit
		}
		return m, nil

	// Handle window resize events.
	case tea.WindowSizeMsg:
		m.width = min(msg.Width, maxWidth) - m.styles.Base.GetHorizontalFrameSize()

		// Resize the viewport.
		headerHeight := lipgloss.Height(m.headerView())
		footerHeight := lipgloss.Height(m.footerView())
		verticalMarginHeight := headerHeight + footerHeight
		if !m.vReady {
			m.viewport = viewport.New(msg.Width-28, msg.Height-verticalMarginHeight-8)
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

	if m.form.State == huh.StateCompleted && !m.isQuitting() {
		var cmd tea.Cmd
		*m.err = fmt.Errorf("complete")
		m.viewport, cmd = m.viewport.Update(tea.QuitMsg{})
		m.quitting = true
		cmds = append(cmds, tea.Quit, cmd)
	}
	return m, tea.Sequence(cmds...)
}

func (m model) View() string {
	if m.isQuitting() {
		return ""
	}
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
	header := m.appBoundaryView("tester package")
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

func new(f *huh.Form, d *string, e *error) *tea.Program {

	return tea.NewProgram(initialModel(f, d, e), tea.WithAltScreen())
}

// Interact is a function that interacts with the user through a form.
//
// It takes three parameters:
// - nFo: a form factory function that returns a new form every time during the retry cycle.
// - doc: a pointer to a string that represents the documentation.
// - invalid: a function that validates the linked values of the form.
//
// It returns an error.
func Interact(nFo func() *huh.Form, doc *string, invalid func() bool) error {
	var (
		retry bool = true
		err1  error
	)

	for retry && err1 == nil {
		_, err := new(nFo(), doc, &err1).Run()
		errorutils.ExitOnFail(err)
		if err1 != nil && err1.Error() == "interrupt" {
			return errorutils.NewReport("user cancelled", "abc")
		} else if err1 != nil && err1.Error() == "complete" {
			retry = false
		}
		if invalid() {
			err := huh.NewConfirm().
				Negative("yes, stop exec").
				Affirmative("I didn't mean to, take me back").
				Description("missing valid selections, did you toggle all programs off?").
				Value(&retry).Run()
			errorutils.ExitOnFail(err)
			if retry {
				err1 = nil
			} else {
				err1 = fmt.Errorf("interrupt")
			}
		}
	}
	if err1 != nil && err1.Error() == "interrupt" {
		return errorutils.NewReport("user cancelled", "xyz")

	}
	return nil
}
