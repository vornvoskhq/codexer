package shared

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/olekukonko/tablewriter"
)

const (
	MaxContextBodySize           = 25 * 1024 * 1024 // 25MB
	MaxContextCount              = 1000
	MaxContextMapPaths           = 3000
	MaxContextMapSingleInputSize = 500 * 1024             // 500KB
	MaxContextMapTotalInputSize  = 250 * 1024 * 1024      // 250MB
	MaxTotalContextSize          = 1 * 1024 * 1024 * 1024 // 1GB

	ContextMapMaxBatchBytes = 10 * 1024 * 1024 // 10MB
	ContextMapMaxBatchSize  = 500
)

type ContextUpdateResult struct {
	UpdatedContexts []*Context
	TokenDiffsById  map[string]int
	TokensDiff      int
	TotalTokens     int
	NumFiles        int
	NumUrls         int
	NumImages       int
	NumTrees        int
	NumMaps         int
	MaxTokens       int
}

func (c *Context) TypeAndIcon() (string, string) {
	var icon string
	var t string
	switch c.ContextType {
	case ContextFileType:
		icon = "📄"
		t = "file"
	case ContextURLType:
		icon = "🌎"
		t = "url"
	case ContextDirectoryTreeType:
		icon = "🗂 "
		t = "tree"
	case ContextNoteType:
		icon = "✏️ "
		t = "note"
	case ContextPipedDataType:
		icon = "↔️ "
		t = "piped"
	case ContextImageType:
		icon = "🖼️ "
		t = "image"
	case ContextMapType:
		icon = "🗺️ "
		t = "map"
	}

	return t, icon
}

func TableForLoadContext(contexts []*Context, plaintext bool) string {
	tableString := &strings.Builder{}
	table := tablewriter.NewWriter(tableString)
	table.SetHeader([]string{"Name", "Type", "🪙"})
	table.SetAutoWrapText(false)

	for _, context := range contexts {
		t, icon := context.TypeAndIcon()
		row := []string{
			" " + icon + " " + context.Name,
			t,
			"+" + strconv.Itoa(context.NumTokens),
		}

		if !plaintext {
			table.Rich(row, []tablewriter.Colors{
				{tablewriter.FgHiGreenColor, tablewriter.Bold},
				{tablewriter.FgHiGreenColor},
				{tablewriter.FgHiGreenColor},
			})
		} else {
			table.Append(row)
		}
	}

	table.Render()

	return strings.TrimSpace(tableString.String())
}

func MarkdownTableForLoadContext(contexts []*Context) string {
	var sb strings.Builder
	sb.WriteString("| Name | Type | 🪙 |\n")
	sb.WriteString("|------|------|----|\n")

	for _, context := range contexts {
		t, icon := context.TypeAndIcon()
		sb.WriteString(fmt.Sprintf("| %s %s | %s | +%d |\n",
			icon, context.Name, t, context.NumTokens))
	}

	return sb.String()
}

func SummaryForLoadContext(contexts []*Context, tokensAdded, totalTokens int) string {

	var hasNote bool
	var hasPiped bool

	var numFiles int
	var numTrees int
	var numUrls int
	var numMaps int

	for _, context := range contexts {
		switch context.ContextType {
		case ContextFileType:
			numFiles++
		case ContextURLType:
			numUrls++
		case ContextDirectoryTreeType:
			numTrees++
		case ContextNoteType:
			hasNote = true
		case ContextPipedDataType:
			hasPiped = true
		case ContextMapType:
			numMaps++
		}
	}

	var added []string

	if hasNote {
		added = append(added, "a note")
	}
	if hasPiped {
		added = append(added, "piped data")
	}
	if numFiles > 0 {
		label := "file"
		if numFiles > 1 {
			label = "files"
		}
		added = append(added, fmt.Sprintf("%d %s", numFiles, label))
	}
	if numTrees > 0 {
		label := "directory tree"
		if numTrees > 1 {
			label = "directory trees"
		}
		added = append(added, fmt.Sprintf("%d %s", numTrees, label))
	}
	if numUrls > 0 {
		label := "url"
		if numUrls > 1 {
			label = "urls"
		}
		added = append(added, fmt.Sprintf("%d %s", numUrls, label))
	}
	if numMaps > 0 {
		label := "map"
		if numMaps > 1 {
			label = "maps"
		}
		added = append(added, fmt.Sprintf("%d %s", numMaps, label))
	}

	msg := "Loaded "

	if len(added) <= 2 {
		msg += strings.Join(added, " and ")
	} else {
		for i, add := range added {
			if i == len(added)-1 {
				msg += ", and " + add
			} else {
				msg += ", " + add
			}
		}
	}

	msg += fmt.Sprintf(" into context | added → %d 🪙 |  total → %d 🪙", tokensAdded, totalTokens)

	return msg
}

func TableForRemoveContext(contexts []*Context) string {
	tableString := &strings.Builder{}
	table := tablewriter.NewWriter(tableString)
	table.SetHeader([]string{"Name", "Type", "🪙"})
	table.SetAutoWrapText(false)

	for _, context := range contexts {
		t, icon := context.TypeAndIcon()
		row := []string{
			" " + icon + " " + context.Name,
			t,
			"-" + strconv.Itoa(context.NumTokens),
		}

		table.Rich(row, []tablewriter.Colors{
			{tablewriter.FgHiRedColor, tablewriter.Bold},
			{tablewriter.FgHiRedColor},
			{tablewriter.FgHiRedColor},
		})
	}

	table.Render()

	return tableString.String()
}

func SummaryForRemoveContext(contexts []*Context, previousTotalTokens int) string {
	removedTokens := 0

	for _, context := range contexts {
		removedTokens += context.NumTokens
	}

	totalTokens := previousTotalTokens - removedTokens

	suffix := ""
	if len(contexts) > 1 {
		suffix = "s"
	}

	return fmt.Sprintf("Removed %d piece%s of context | removed → %d 🪙 | total → %d 🪙", len(contexts), suffix, removedTokens, totalTokens)
}

type SummaryForUpdateContextParams struct {
	NumFiles    int
	NumTrees    int
	NumUrls     int
	NumMaps     int
	TokensDiff  int
	TotalTokens int
}

func SummaryForUpdateContext(params SummaryForUpdateContextParams) string {
	numFiles := params.NumFiles
	numTrees := params.NumTrees
	numUrls := params.NumUrls
	numMaps := params.NumMaps
	tokensDiff := params.TokensDiff
	totalTokens := params.TotalTokens

	msg := "Updated"
	var toAdd []string
	if numFiles > 0 {
		postfix := "s"
		if numFiles == 1 {
			postfix = ""
		}
		toAdd = append(toAdd, fmt.Sprintf("%d file%s", numFiles, postfix))
	}
	if numTrees > 0 {
		postfix := "s"
		if numTrees == 1 {
			postfix = ""
		}
		toAdd = append(toAdd, fmt.Sprintf("%d tree%s", numTrees, postfix))
	}
	if numUrls > 0 {
		postfix := "s"
		if numUrls == 1 {
			postfix = ""
		}
		toAdd = append(toAdd, fmt.Sprintf("%d url%s", numUrls, postfix))
	}
	if numMaps > 0 {
		postfix := "s"
		if numMaps == 1 {
			postfix = ""
		}
		toAdd = append(toAdd, fmt.Sprintf("%d map%s", numMaps, postfix))
	}

	if len(toAdd) <= 2 {
		msg += " " + strings.Join(toAdd, " and ")
	} else {
		for i, add := range toAdd {
			if i == len(toAdd)-1 {
				msg += ", and " + add
			} else {
				msg += ", " + add
			}
		}
	}

	msg += " in context"

	action := "added"
	if tokensDiff < 0 {
		action = "removed"
	}
	absTokenDiff := int(math.Abs(float64(tokensDiff)))
	msg += fmt.Sprintf(" | %s → %d 🪙 | total → %d 🪙", action, absTokenDiff, totalTokens)

	return msg
}

func TableForContextUpdate(updateRes *ContextUpdateResult) string {
	contexts := updateRes.UpdatedContexts
	tokenDiffsById := updateRes.TokenDiffsById

	if len(contexts) == 0 {
		return ""
	}

	tableString := &strings.Builder{}
	table := tablewriter.NewWriter(tableString)
	table.SetHeader([]string{"Name", "Type", "🪙"})
	table.SetAutoWrapText(false)

	for _, context := range contexts {
		t, icon := context.TypeAndIcon()
		diff := tokenDiffsById[context.Id]

		diffStr := "+" + strconv.Itoa(diff)
		tableColor := tablewriter.FgHiGreenColor

		if diff < 0 {
			diffStr = strconv.Itoa(diff)
			tableColor = tablewriter.FgHiRedColor
		}

		row := []string{
			" " + icon + " " + context.Name,
			t,
			diffStr,
		}

		table.Rich(row, []tablewriter.Colors{
			{tableColor, tablewriter.Bold},
			{tableColor},
			{tableColor},
		})
	}

	table.Render()

	return tableString.String()
}
