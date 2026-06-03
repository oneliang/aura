package tui_test

import (
	"github.com/oneliang/aura/cli/pkg/tui"
	"github.com/oneliang/aura/shared/pkg/i18n"
)

func init() {
	i18n.Init("", "en")
	tui.InitI18nConstants()
}
