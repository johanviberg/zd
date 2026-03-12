package tui

func renderLogo() string {
	return logoAccentStyle.Render("▌") + logoTextStyle.Render("zd")
}
