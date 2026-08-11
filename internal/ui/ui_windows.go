//go:build windows

package ui

import (
	"context"
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/DylonH78/AppAlias/internal/model"
	"github.com/DylonH78/AppAlias/internal/service"
)

// Show opens the non-resident AppAlias management window.
func Show(svc *service.Service) {
	a := app.NewWithID("io.github.appalias.manager")
	w := a.NewWindow("AppAlias")
	w.Resize(fyne.NewSize(940, 640))

	var candidates []model.Candidate
	var filtered []int
	selectedCandidate := -1
	filter := widget.NewEntry()
	filter.SetPlaceHolder("筛选已发现的应用…")
	details := widget.NewMultiLineEntry()
	details.Disable()
	details.SetMinRowsVisible(4)

	candidateList := widget.NewList(
		func() int { return len(filtered) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(id widget.ListItemID, object fyne.CanvasObject) {
			candidate := candidates[filtered[id]]
			name := candidate.Recommended
			if name == "" {
				name = "需要手动命名"
			}
			object.(*widget.Label).SetText(fmt.Sprintf("%s  →  %s", name, candidate.DisplayName))
		},
	)

	refreshCandidates := func() {
		needle := strings.ToLower(strings.TrimSpace(filter.Text))
		filtered = filtered[:0]
		for index, candidate := range candidates {
			contents := strings.ToLower(strings.Join(append([]string{candidate.DisplayName, candidate.Recommended, candidate.Launch.Target}, candidate.Suggestions...), " "))
			if needle == "" || strings.Contains(contents, needle) {
				filtered = append(filtered, index)
			}
		}
		selectedCandidate = -1
		details.SetText("")
		candidateList.Refresh()
	}
	filter.OnChanged = func(_ string) { refreshCandidates() }
	candidateList.OnSelected = func(id widget.ListItemID) {
		selectedCandidate = filtered[id]
		candidate := candidates[selectedCandidate]
		details.SetText(fmt.Sprintf("名称：%s\n来源：%s\n推荐别名：%s\n全部候选：%s\n目标：%s", candidate.DisplayName, candidate.Source, candidate.Recommended, strings.Join(candidate.Suggestions, ", "), candidate.Launch.Target))
	}

	scanButton := widget.NewButton("扫描应用", func() {
		result := svc.Scan(context.Background())
		candidates = result.Candidates
		refreshCandidates()
		if len(result.Diagnostics) > 0 {
			dialog.ShowInformation("扫描完成（部分来源有警告）", strings.Join(result.Diagnostics, "\n"), w)
		}
	})
	applySelected := widget.NewButton("使用推荐别名", func() {
		if selectedCandidate < 0 {
			dialog.ShowInformation("请选择应用", "先在列表中选择一项。", w)
			return
		}
		report, err := svc.Apply([]model.Candidate{candidates[selectedCandidate]})
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		if len(report.Applied) == 0 {
			dialog.ShowInformation("未创建别名", formatSkipped(report.Skipped), w)
			return
		}
		dialog.ShowInformation("已创建别名", strings.Join(report.Applied, ", "), w)
	})
	applyAll := widget.NewButton("应用全部安全候选", func() {
		report, err := svc.Apply(candidates)
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		message := "已创建：" + strings.Join(report.Applied, ", ")
		if len(report.Skipped) > 0 {
			message += "\n\n已跳过：\n" + formatSkipped(report.Skipped)
		}
		dialog.ShowInformation("扫描结果", message, w)
	})
	candidatesTab := container.NewBorder(
		container.NewVBox(filter, container.NewHBox(scanButton, applySelected, applyAll)),
		details,
		nil,
		nil,
		candidateList,
	)

	var aliases []model.Alias
	selectedAlias := -1
	aliasList := widget.NewList(
		func() int { return len(aliases) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(id widget.ListItemID, object fyne.CanvasObject) {
			item := aliases[id]
			object.(*widget.Label).SetText(fmt.Sprintf("%s  →  %s", item.Name, item.DisplayName))
		},
	)
	refreshAliases := func() {
		items, err := svc.List()
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		aliases = items
		selectedAlias = -1
		aliasList.Refresh()
	}
	aliasList.OnSelected = func(id widget.ListItemID) { selectedAlias = id }

	aliasName := widget.NewEntry()
	aliasName.SetPlaceHolder("自定义别名")
	targetPath := widget.NewEntry()
	targetPath.SetPlaceHolder("选择 .exe 文件")
	pickTarget := widget.NewButton("浏览…", func() {
		dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			if reader != nil {
				targetPath.SetText(reader.URI().Path())
				_ = reader.Close()
			}
		}, w).Show()
	})
	addManual := widget.NewButton("添加", func() {
		if err := svc.Add(aliasName.Text, aliasName.Text, model.LaunchSpec{Kind: model.LaunchExecutable, Target: targetPath.Text}); err != nil {
			dialog.ShowError(err, w)
			return
		}
		aliasName.SetText("")
		targetPath.SetText("")
		refreshAliases()
	})
	renameName := widget.NewEntry()
	renameName.SetPlaceHolder("选中项的新名称")
	renameButton := widget.NewButton("重命名", func() {
		if selectedAlias < 0 {
			dialog.ShowInformation("请选择别名", "先在列表中选择一项。", w)
			return
		}
		if err := svc.Rename(aliases[selectedAlias].Name, renameName.Text); err != nil {
			dialog.ShowError(err, w)
			return
		}
		renameName.SetText("")
		refreshAliases()
	})
	testButton := widget.NewButton("测试启动", func() {
		if selectedAlias < 0 {
			dialog.ShowInformation("请选择别名", "先在列表中选择一项。", w)
			return
		}
		if err := svc.Launch(aliases[selectedAlias].Name); err != nil {
			dialog.ShowError(err, w)
		}
	})
	removeButton := widget.NewButton("删除", func() {
		if selectedAlias < 0 {
			dialog.ShowInformation("请选择别名", "先在列表中选择一项。", w)
			return
		}
		item := aliases[selectedAlias]
		dialog.ShowConfirm("删除别名", fmt.Sprintf("删除 %s？原应用不会受到影响。", item.Name), func(confirm bool) {
			if !confirm {
				return
			}
			if err := svc.Remove(item.Name); err != nil {
				dialog.ShowError(err, w)
				return
			}
			refreshAliases()
		}, w)
	})
	doctorButton := widget.NewButton("健康检查", func() {
		report, err := svc.Doctor()
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		message := fmt.Sprintf("根目录：%s\n当前终端可见 PATH：%t\n启动器存在：%t", report.Root, report.PathInSession, report.LauncherPresent)
		if len(report.Issues) > 0 {
			message += "\n\n问题：\n" + strings.Join(report.Issues, "\n")
		}
		dialog.ShowInformation("健康检查", message, w)
	})
	aliasesTab := container.NewBorder(
		container.NewVBox(
			widget.NewLabel("手动添加"),
			container.NewGridWithColumns(2, aliasName, targetPath),
			container.NewHBox(pickTarget, addManual),
			widget.NewSeparator(),
			container.NewHBox(renameName, renameButton, testButton, removeButton, doctorButton),
		),
		nil,
		nil,
		nil,
		aliasList,
	)

	refreshAliases()
	w.SetContent(container.NewAppTabs(
		container.NewTabItem("发现应用", candidatesTab),
		container.NewTabItem("已管理别名", aliasesTab),
	))
	w.ShowAndRun()
}

func formatSkipped(skipped map[string]string) string {
	lines := make([]string, 0, len(skipped))
	for name, reason := range skipped {
		lines = append(lines, fmt.Sprintf("%s：%s", name, reason))
	}
	return strings.Join(lines, "\n")
}
