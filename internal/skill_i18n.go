package dub

type skillTextKey int

const (
	skillTextLanguageTitle skillTextKey = iota
	skillTextScopeTitle
	skillTextScopeDescription
	skillTextScopeProject
	skillTextScopeGlobal
	skillTextPlatformTitle
	skillTextPlatformDescription
	skillTextDetected
	skillTextSelectOnePlatform
	skillTextHelpMove
	skillTextHelpToggle
	skillTextHelpFilter
	skillTextHelpAll
	skillTextHelpNone
	skillTextHelpConfirm
	skillTextResultTitle
	skillTextResultScope
	skillTextResultSource
	skillTextStatusInstalled
	skillTextStatusUpdated
	skillTextStatusUnchanged
	skillTextStatusConflict
	skillTextStatusError
	skillTextConflictGuidance
	skillTextNonTTYGuidance
)

var skillTranslations = map[skillLanguage]map[skillTextKey]string{
	skillLanguageEN: {
		skillTextLanguageTitle:       "Language / 语言",
		skillTextScopeTitle:          "Install scope",
		skillTextScopeDescription:    "Choose where OrcaDub should install the Skill",
		skillTextScopeProject:        "Project — current directory",
		skillTextScopeGlobal:         "Global — home directory",
		skillTextPlatformTitle:       "Select platforms",
		skillTextPlatformDescription: "Detected platforms are preselected",
		skillTextDetected:            "detected",
		skillTextSelectOnePlatform:   "Select at least one platform",
		skillTextHelpMove:            "move",
		skillTextHelpToggle:          "toggle",
		skillTextHelpFilter:          "filter",
		skillTextHelpAll:             "all",
		skillTextHelpNone:            "none",
		skillTextHelpConfirm:         "confirm",
		skillTextResultTitle:         "OrcaDub Skill installation",
		skillTextResultScope:         "Scope",
		skillTextResultSource:        "Source",
		skillTextStatusInstalled:     "installed",
		skillTextStatusUpdated:       "updated",
		skillTextStatusUnchanged:     "unchanged",
		skillTextStatusConflict:      "conflict",
		skillTextStatusError:         "error",
		skillTextConflictGuidance:    "kept existing file; rerun with --force",
		skillTextNonTTYGuidance:      "interactive input requires a terminal; use --lang, --scope, --platform, or --yes",
	},
	skillLanguageZH: {
		skillTextLanguageTitle:       "Language / 语言",
		skillTextScopeTitle:          "安装范围",
		skillTextScopeDescription:    "选择 OrcaDub Skill 的安装位置",
		skillTextScopeProject:        "当前项目",
		skillTextScopeGlobal:         "全局安装",
		skillTextPlatformTitle:       "选择安装平台",
		skillTextPlatformDescription: "已检测到的平台已自动勾选",
		skillTextDetected:            "已检测",
		skillTextSelectOnePlatform:   "请至少选择一个平台",
		skillTextHelpMove:            "移动",
		skillTextHelpToggle:          "勾选",
		skillTextHelpFilter:          "搜索",
		skillTextHelpAll:             "全选",
		skillTextHelpNone:            "清空",
		skillTextHelpConfirm:         "确认",
		skillTextResultTitle:         "OrcaDub Skill 安装结果",
		skillTextResultScope:         "安装范围",
		skillTextResultSource:        "来源",
		skillTextStatusInstalled:     "已安装",
		skillTextStatusUpdated:       "已更新",
		skillTextStatusUnchanged:     "无需更新",
		skillTextStatusConflict:      "冲突",
		skillTextStatusError:         "错误",
		skillTextConflictGuidance:    "已保留现有文件；使用 --force 重新运行可覆盖",
		skillTextNonTTYGuidance:      "交互式安装需要终端；请使用 --lang、--scope、--platform 或 --yes",
	},
}

func skillText(language skillLanguage, key skillTextKey) string {
	if text := skillTranslations[language][key]; text != "" {
		return text
	}
	return skillTranslations[skillLanguageEN][key]
}
