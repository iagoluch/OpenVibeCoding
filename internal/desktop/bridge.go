//go:build desktop

package desktop

import (
	"reflect"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// folderPicker opens the native directory chooser. It is an interface so the
// bridge is testable without a window.
type folderPicker interface {
	pickFolder() (string, error)
}

// UI is the single bound service of the desktop shell, registered by wails
// under "<package path>.UI". The bridge is deliberately confined: PickFolder
// is its only capability, it holds no state between calls (no path cache),
// and the path it returns is only a proposal — the page must still send it to
// the daemon, which validates it server-side.
//
// Nothing else is reachable from the page: no file read or write, no
// execution, no environment or credential access, no clipboard, no network
// fetch, and no arbitrary ExecJS surface. The confinement test enumerates the
// bound methods and fails if anything beyond PickFolder is registered.
type UI struct {
	mu     sync.Mutex
	picker folderPicker
}

func newUI(picker folderPicker) *UI {
	return &UI{picker: picker}
}

// PickFolder opens one native folder dialog and returns the chosen path, or
// an empty string when the user cancels. Calls are mutex-serialized so only
// one modal exists at a time. Errors are returned to the caller (the page),
// never swallowed.
func (u *UI) PickFolder() (string, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.picker.pickFolder()
}

// wailsPicker drives the wails native file dialog (POC-verified API).
type wailsPicker struct{}

func (wailsPicker) pickFolder() (string, error) {
	app := application.Get()
	dialog := app.Dialog.OpenFile()
	dialog.SetTitle("OpenVibeCoding — escolha uma pasta")
	dialog.CanChooseDirectories(true)
	dialog.CanChooseFiles(false)
	return dialog.PromptForSingleSelection()
}

// bindingFqn is the name wails registers for the bound method: the full Go
// package path plus type and method name. The page pins the short contract
// name "desktop.UI.PickFolder"; the shell-owned alias module served at
// /ovc-bridge.js rewrites exactly that one name to this FQN.
func bindingFqn() string {
	return reflect.TypeOf(UI{}).PkgPath() + ".UI.PickFolder"
}

// boundServices lists every service the shell binds, in one function so the
// confinement test can assert the complete set.
func boundServices(ui *UI) []application.Service {
	return []application.Service{application.NewService(ui)}
}
