package components

import (
	"fmt"
	"sort"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/nq/hv-tui/internal/vault"
)

var (
	svPathStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#BB9AF7")).
			Bold(true)

	svLabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#565F89"))

	svMetaValStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#C0CAF5"))

	svDividerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#414868"))

	svKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E0AF68"))

	svValueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#C0CAF5"))

	svSectionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7AA2F7")).
			Bold(true)

	svEmptyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#565F89")).
			Italic(true)

	svVersionActiveStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#9ECE6A"))

	svVersionDeletedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#565F89"))

	svVersionDestroyedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#F7768E"))

	svVersionNumStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#BB9AF7")).
				Bold(true)

	svTimestampStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#565F89"))
)

func RenderSecret(secret *vault.SecretEntry, width int) string {
	if secret == nil {
		return ""
	}

	var b strings.Builder

	b.WriteString("  ")
	b.WriteString(svPathStyle.Render(secret.Path))
	b.WriteString("\n")

	divWidth := width - 4
	if divWidth < 10 {
		divWidth = 10
	}
	b.WriteString("  ")
	b.WriteString(svDividerStyle.Render(strings.Repeat("─", divWidth)))
	b.WriteString("\n\n")

	if secret.Metadata != nil {
		b.WriteString(svSectionStyle.Render("  Metadata"))
		b.WriteString("\n\n")

		b.WriteString("  ")
		b.WriteString(svLabelStyle.Render("version   "))
		b.WriteString(svMetaValStyle.Render(fmt.Sprintf("v%d", secret.Metadata.Version)))
		b.WriteString("\n")

		if !secret.Metadata.CreatedTime.IsZero() {
			b.WriteString("  ")
			b.WriteString(svLabelStyle.Render("created   "))
			b.WriteString(svMetaValStyle.Render(secret.Metadata.CreatedTime.Format("2006-01-02 15:04:05 MST")))
			b.WriteString("\n")
		}

		b.WriteString("\n")
	}

	b.WriteString(svSectionStyle.Render("  Data"))
	b.WriteString("\n\n")

	if len(secret.Data) == 0 {
		b.WriteString("  ")
		b.WriteString(svEmptyStyle.Render("(empty)"))
		b.WriteString("\n")
		return b.String()
	}

	sortedKeys := make([]string, 0, len(secret.Data))
	for k := range secret.Data {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Strings(sortedKeys)

	maxKeyLen := 0
	for _, k := range sortedKeys {
		if len(k) > maxKeyLen {
			maxKeyLen = len(k)
		}
	}

	for _, k := range sortedKeys {
		v := fmt.Sprintf("%v", secret.Data[k])
		padding := strings.Repeat(" ", maxKeyLen-len(k))

		b.WriteString("  ")
		b.WriteString(svKeyStyle.Render(k))
		b.WriteString(padding)
		b.WriteString(svDividerStyle.Render("  │ "))
		b.WriteString(svValueStyle.Render(v))
		b.WriteString("\n")
	}

	return b.String()
}

func RenderVersions(versions []vault.VersionInfo, path string) string {
	if len(versions) == 0 {
		return ""
	}

	var b strings.Builder

	b.WriteString("  ")
	b.WriteString(svPathStyle.Render(path))
	b.WriteString("\n")
	b.WriteString("  ")
	b.WriteString(svDividerStyle.Render(strings.Repeat("─", 48)))
	b.WriteString("\n\n")

	b.WriteString(svSectionStyle.Render("  Version History"))
	b.WriteString("\n\n")

	b.WriteString("  ")
	b.WriteString(svLabelStyle.Render(fmt.Sprintf("  %-8s  %-24s  %s", "VERSION", "CREATED", "STATUS")))
	b.WriteString("\n")
	b.WriteString("  ")
	b.WriteString(svDividerStyle.Render(strings.Repeat("─", 48)))
	b.WriteString("\n")

	for _, v := range versions {
		status := "active"
		var statusStyle lipgloss.Style
		if v.Destroyed {
			status = "destroyed"
			statusStyle = svVersionDestroyedStyle
		} else if v.Deleted {
			status = "deleted"
			statusStyle = svVersionDeletedStyle
		} else {
			statusStyle = svVersionActiveStyle
		}

		b.WriteString("  ")
		b.WriteString(svVersionNumStyle.Render(fmt.Sprintf("  v%-6d", v.Version)))
		b.WriteString(svTimestampStyle.Render(fmt.Sprintf("  %-24s", v.CreatedTime.Format("2006-01-02 15:04:05"))))
		b.WriteString("  ")
		b.WriteString(statusStyle.Render(status))
		b.WriteString("\n")
	}

	return b.String()
}
