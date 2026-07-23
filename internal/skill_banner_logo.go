package dub

import (
	_ "embed"
	"strings"
)

const (
	skillBannerLogoWidth  = 40
	skillBannerHeight     = 20
	skillBannerWordWidth  = 34
	skillBannerGapWidth   = 3
	skillBannerTotalWidth = skillBannerLogoWidth + skillBannerGapWidth + skillBannerWordWidth
)

//go:embed assets/orca_logo_color.ansi
var skillBannerLogoColor string

//go:embed assets/orca_logo_plain.txt
var skillBannerLogoPlain string

func skillBannerLogoRows(color bool) []string {
	rows, _ := skillBannerLogoRowsWithValidity(color)
	return rows
}

func skillBannerLogoRowsWithValidity(color bool) ([]string, bool) {
	value := skillBannerLogoPlain
	if color {
		value = skillBannerLogoColor
	}
	rows := strings.Split(strings.TrimSuffix(value, "\n"), "\n")
	if len(rows) == skillBannerHeight {
		if !color {
			for index := range rows {
				rows[index] = strings.ReplaceAll(rows[index], ".", " ")
			}
		}
		return rows, true
	}
	blank := strings.Repeat(" ", skillBannerLogoWidth)
	rows = make([]string, skillBannerHeight)
	for index := range rows {
		rows[index] = blank
	}
	return rows, false
}
