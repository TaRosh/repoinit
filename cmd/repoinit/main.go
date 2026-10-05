// Command repoinit — окно для быстрой настройки нового проекта на репозиторий
// GitHub через deploy-ключ: вставил ссылку, выбрал папку, скопировал ключ.
package main

import (
	"context"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/TaRosh/repoinit/internal/ghurl"
	"github.com/TaRosh/repoinit/internal/setup"
)

// prefBaseDir — папка, в которой лежат проекты (запоминается между запусками).
const prefBaseDir = "baseDir"

type ui struct {
	app fyne.App
	win fyne.Window

	urlEntry *widget.Entry
	dirEntry *widget.Entry
	autoDir  string // путь, подставленный автоматически (его можно перезаписывать)

	initBtn  *widget.Button
	progress *widget.ProgressBarInfinite

	resultBox *fyne.Container // скрыт до успешной настройки
	steps     *widget.Label
	keyLabel  *widget.Label
	copyBtn   *widget.Button
	openBtn   *widget.Button
	checkBtn  *widget.Button
	status    *widget.Label

	result setup.Result
}

func main() {
	a := app.NewWithID("io.github.tarosh.repoinit")
	u := &ui{app: a, win: a.NewWindow("Repo Init — deploy-ключ для GitHub")}
	u.build()
	u.win.Resize(fyne.NewSize(680, 560))
	u.win.ShowAndRun()
}

func (u *ui) build() {
	u.urlEntry = widget.NewEntry()
	u.urlEntry.SetPlaceHolder("https://github.com/owner/repo")
	u.urlEntry.OnChanged = u.suggestDir

	u.dirEntry = widget.NewEntry()
	u.dirEntry.SetPlaceHolder(`F:\golangProjects\project`)
	browse := widget.NewButtonWithIcon("Обзор…", theme.FolderOpenIcon(), u.chooseDir)

	u.initBtn = widget.NewButtonWithIcon("Инициализировать", theme.ConfirmIcon(), u.runSetup)
	u.initBtn.Importance = widget.HighImportance
	u.progress = widget.NewProgressBarInfinite()
	u.progress.Hide()

	form := widget.NewForm(
		widget.NewFormItem("Репозиторий", u.urlEntry),
		widget.NewFormItem("Папка", container.NewBorder(nil, nil, nil, browse, u.dirEntry)),
	)

	u.steps = widget.NewLabel("")
	u.keyLabel = widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Monospace: true})
	u.keyLabel.Wrapping = fyne.TextWrapBreak // ключ — одно длинное "слово"
	u.keyLabel.Selectable = true

	u.copyBtn = widget.NewButtonWithIcon("Копировать ключ", theme.ContentCopyIcon(), u.copyKey)
	u.copyBtn.Importance = widget.HighImportance
	u.openBtn = widget.NewButtonWithIcon("Открыть Deploy keys", theme.ComputerIcon(), u.openDeployKeys)
	u.checkBtn = widget.NewButtonWithIcon("Проверить доступ", theme.ViewRefreshIcon(), u.checkAccess)
	u.status = widget.NewLabel("")
	u.status.Wrapping = fyne.TextWrapWord

	hint := widget.NewLabel("На GitHub: Add deploy key → вставьте ключ → отметьте «Allow write access», " +
		"чтобы можно было делать git push. Затем нажмите «Проверить доступ».")
	hint.Wrapping = fyne.TextWrapWord

	u.resultBox = container.NewVBox(
		widget.NewSeparator(),
		u.steps,
		widget.NewLabelWithStyle("Публичный ключ", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		u.keyLabel,
		container.NewGridWithColumns(3, u.copyBtn, u.openBtn, u.checkBtn),
		hint,
		u.status,
	)
	u.resultBox.Hide()

	u.win.SetContent(container.NewPadded(container.NewVScroll(container.NewVBox(
		form,
		u.initBtn,
		u.progress,
		u.resultBox,
	))))
}

// suggestDir подставляет папку <последняя папка проектов>\<имя репозитория>,
// пока пользователь не ввёл путь вручную.
func (u *ui) suggestDir(raw string) {
	base := u.app.Preferences().String(prefBaseDir)
	if base == "" || (u.dirEntry.Text != "" && u.dirEntry.Text != u.autoDir) {
		return
	}
	repo, err := ghurl.Parse(raw)
	if err != nil {
		return
	}
	u.autoDir = filepath.Join(base, repo.Name)
	u.dirEntry.SetText(u.autoDir)
}

func (u *ui) chooseDir() {
	dialog.ShowFolderOpen(func(dir fyne.ListableURI, err error) {
		if err != nil {
			dialog.ShowError(err, u.win)
			return
		}
		if dir == nil {
			return // пользователь нажал "Отмена"
		}
		path := dir.Path()
		// Выбрали папку, где лежат все проекты, — добавим имя репозитория.
		if repo, err := ghurl.Parse(u.urlEntry.Text); err == nil && filepath.Base(path) != repo.Name {
			path = filepath.Join(path, repo.Name)
		}
		u.dirEntry.SetText(path)
	}, u.win)
}

func (u *ui) runSetup() {
	rawURL, dir := u.urlEntry.Text, u.dirEntry.Text
	u.setBusy(true)

	// Git может работать несколько секунд — делаем это не в UI-потоке,
	// иначе окно "зависнет" и перестанет перерисовываться.
	go func() {
		res, err := setup.Run(context.Background(), rawURL, dir)

		// Менять виджеты можно только из UI-потока: fyne.Do передаёт функцию туда.
		fyne.Do(func() {
			u.setBusy(false)
			if err != nil {
				dialog.ShowError(err, u.win)
				return
			}
			u.result = res
			u.app.Preferences().SetString(prefBaseDir, filepath.Dir(res.Dir))
			u.showResult()
		})
	}()
}

func (u *ui) showResult() {
	lines := make([]string, len(u.result.Steps))
	for i, s := range u.result.Steps {
		lines[i] = "• " + s
	}
	u.steps.SetText(strings.Join(lines, "\n"))
	u.keyLabel.SetText(u.result.PublicKey)
	u.status.SetText("")
	u.resultBox.Show()
}

func (u *ui) copyKey() {
	u.app.Clipboard().SetContent(u.result.PublicKey)
	u.copyBtn.SetText("Скопировано ✓")
	time.AfterFunc(2*time.Second, func() {
		fyne.Do(func() { u.copyBtn.SetText("Копировать ключ") })
	})
}

func (u *ui) openDeployKeys() {
	link, err := url.Parse(u.result.Repo.DeployKeysURL())
	if err == nil {
		err = u.app.OpenURL(link)
	}
	if err != nil {
		dialog.ShowError(err, u.win)
	}
}

func (u *ui) checkAccess() {
	u.checkBtn.Disable()
	u.status.SetText("Подключаюсь к GitHub…")
	go func() {
		err := setup.CheckAccess(context.Background(), u.result.Dir)
		fyne.Do(func() {
			u.checkBtn.Enable()
			if err != nil {
				u.status.SetText("Нет доступа. Ключ добавлен в Deploy keys этого репозитория?\n" + err.Error())
				return
			}
			u.status.SetText("Доступ есть — можно делать git push.")
		})
	}()
}

func (u *ui) setBusy(busy bool) {
	if busy {
		u.initBtn.Disable()
		u.progress.Show()
		u.progress.Start()
		return
	}
	u.progress.Stop()
	u.progress.Hide()
	u.initBtn.Enable()
}
