package main

import (
	"image/color"
	
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type myTheme struct{}

func (m myTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		if variant == theme.VariantLight {
			return color.RGBA{R: 250, G: 250, B: 250, A: 255}
		}
		return color.RGBA{R: 20, G: 20, B: 20, A: 255}
	case theme.ColorNameForeground:
		if variant == theme.VariantLight {
			return color.RGBA{R: 30, G: 30, B: 30, A: 255}
		}
		return color.RGBA{R: 220, G: 220, B: 220, A: 255}
	case theme.ColorNamePrimary:
		return color.RGBA{R: 0, G: 123, B: 255, A: 255} // BMW Blue
	}
	
	return theme.DefaultTheme().Color(name, variant)
}

func (m myTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (m myTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (m myTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}