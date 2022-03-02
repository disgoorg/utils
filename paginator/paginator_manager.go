package paginator

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/DisgoOrg/disgo/core"
	"github.com/DisgoOrg/disgo/core/events"
	"github.com/DisgoOrg/disgo/discord"
)

var _ core.EventListener = (*Manager)(nil)

func NewManager(opts ...ConfigOpt) *Manager {
	config := &DefaultConfig
	config.Apply(opts)
	return &Manager{
		Config: *config,
	}
}

type Manager struct {
	Config Config

	mu         sync.Mutex
	paginators map[string]*Paginator
}

func (m *Manager) startCleanup() {
	go func() {
		ticker := time.NewTimer(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			m.cleanup()
		}
	}()
}

func (m *Manager) cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	for _, p := range m.paginators {
		if !p.Expiry.IsZero() && p.Expiry.After(now) {
			// TODO: remove components?
			delete(m.paginators, p.ID)
		}
	}
}

func (m *Manager) Create(interaction *core.CreateInteraction, paginator *Paginator) {
	if paginator.ID == "" {
		paginator.ID = interaction.ID.String()
	}

	m.add(paginator)

	var err error
	if interaction.Acknowledged {
		_, err = interaction.UpdateOriginalMessage(m.makeMessageUpdate(paginator))
	} else {
		err = interaction.CreateMessage(m.makeMessageCreate(paginator))
	}
	if err != nil {
		interaction.Bot.Logger.Error("Failed to create paginator message: ", err)
	}
}

func (m *Manager) add(paginator *Paginator) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.paginators[paginator.ID] = paginator
}

func (m *Manager) remove(paginatorID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.paginators, paginatorID)
}

func (m *Manager) OnEvent(event core.Event) {
	e, ok := event.(*events.ComponentInteractionEvent)
	if !ok {
		return
	}
	customID := e.Data.ID()
	if !strings.HasPrefix(customID.String(), m.Config.CustomIDPrefix) {
		return
	}
	ids := strings.Split(customID.String(), ":")
	paginatorID, action := ids[1], ids[2]
	paginator, ok := m.paginators[paginatorID]
	if !ok {
		return
	}

	if paginator.Creator != "" && paginator.Creator != e.User.ID {
		if err := e.CreateMessage(discord.NewMessageCreateBuilder().SetContent("You can't interact with this paginator because it's not yours.").SetEphemeral(true).Build()); err != nil {
			e.Bot().Logger.Error("Failed to send error message: ", err)
		}
		return
	}

	switch action {
	case "first":
		paginator.CurrentPage = 0

	case "back":
		paginator.CurrentPage--

	case "stop":
		err := e.UpdateMessage(discord.MessageUpdate{Components: &[]discord.ContainerComponent{}})
		m.remove(paginatorID)
		if err != nil {
			e.Bot().Logger.Error("Error updating paginator message: ", err)
		}
		return

	case "next":
		paginator.CurrentPage++

	case "last":
		paginator.CurrentPage = paginator.MaxPages - 1
	}

	if err := e.UpdateMessage(m.makeMessageUpdate(paginator)); err != nil {
		e.Bot().Logger.Error("Error updating paginator message: ", err)
	}
}

func (m *Manager) makeEmbed(paginator *Paginator) discord.Embed {
	embedBuilder := discord.NewEmbedBuilder().
		SetFooterText(fmt.Sprintf("Page: %d/%d", paginator.CurrentPage+1, paginator.MaxPages)).
		SetColor(m.Config.EmbedColor)

	return paginator.PageFunc(paginator.CurrentPage, embedBuilder)
}

func (m *Manager) makeMessageCreate(paginator *Paginator) discord.MessageCreate {
	return discord.MessageCreate{Embeds: []discord.Embed{m.makeEmbed(paginator)}, Components: []discord.ContainerComponent{m.createComponents(paginator)}}
}

func (m *Manager) makeMessageUpdate(paginator *Paginator) discord.MessageUpdate {
	return discord.MessageUpdate{Embeds: &[]discord.Embed{m.makeEmbed(paginator)}, Components: &[]discord.ContainerComponent{m.createComponents(paginator)}}
}

func (m *Manager) formatCustomID(paginator *Paginator, action string) discord.CustomID {
	return discord.CustomID(m.Config.CustomIDPrefix + ":" + paginator.ID + ":" + action)
}

func (m *Manager) createComponents(paginator *Paginator) discord.ContainerComponent {
	cfg := m.Config.ButtonsConfig
	var actionRow discord.ActionRowComponent

	if cfg.First != nil {
		actionRow.AddComponents(discord.NewButton(cfg.First.Style, cfg.First.Label, m.formatCustomID(paginator, "first"), "").WithEmoji(cfg.First.Emoji).WithDisabled(paginator.CurrentPage == 0))
	}
	if cfg.Back != nil {
		actionRow.AddComponents(discord.NewButton(cfg.Back.Style, cfg.Back.Label, m.formatCustomID(paginator, "back"), "").WithEmoji(cfg.Back.Emoji).WithDisabled(paginator.CurrentPage == 0))
	}

	if cfg.Stop != nil {
		actionRow.AddComponents(discord.NewButton(cfg.Stop.Style, cfg.Stop.Label, m.formatCustomID(paginator, "stop"), "").WithEmoji(cfg.Stop.Emoji))
	}

	if cfg.Next != nil {
		actionRow.AddComponents(discord.NewButton(cfg.Next.Style, cfg.Next.Label, m.formatCustomID(paginator, "next"), "").WithEmoji(cfg.Next.Emoji).WithDisabled(paginator.CurrentPage == paginator.MaxPages-1))
	}
	if cfg.Last != nil {
		actionRow.AddComponents(discord.NewButton(cfg.Last.Style, cfg.Last.Label, m.formatCustomID(paginator, "last"), "").WithEmoji(cfg.Last.Emoji).WithDisabled(paginator.CurrentPage == paginator.MaxPages-1))
	}

	return actionRow
}
