package paginator

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake"
)

var _ bot.EventListener = (*Manager)(nil)

type Paginator struct {
	PageFunc        func(page int, embed *discord.EmbedBuilder)
	MaxPages        int
	Creator         snowflake.Snowflake
	ExpiryLastUsage bool
	expiry          time.Time
	currentPage     int
	ID              string
}

func NewManager(opts ...ConfigOpt) *Manager {
	config := DefaultConfig()
	config.Apply(opts)
	manager := &Manager{
		config:     *config,
		paginators: map[string]*Paginator{},
	}
	manager.startCleanup()
	return manager
}

type Manager struct {
	config     Config
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
		if !p.expiry.IsZero() && p.expiry.After(now) {
			// TODO: remove components?
			delete(m.paginators, p.ID)
		}
	}
}

func (m *Manager) Update(responderFunc events.InteractionResponderFunc, paginator *Paginator) error {
	paginator.expiry = time.Now()
	m.add(paginator)

	return responderFunc(discord.InteractionCallbackTypeUpdateMessage, m.makeMessageUpdate(paginator))
}

func (m *Manager) Create(responderFunc events.InteractionResponderFunc, paginator *Paginator) error {
	paginator.expiry = time.Now()
	m.add(paginator)

	return responderFunc(discord.InteractionCallbackTypeCreateMessage, m.makeMessageCreate(paginator))
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

func (m *Manager) OnEvent(event bot.Event) {
	e, ok := event.(*events.ComponentInteractionEvent)
	if !ok {
		return
	}
	customID := e.Data.CustomID()
	if !strings.HasPrefix(customID.String(), m.config.CustomIDPrefix) {
		return
	}
	ids := strings.Split(customID.String(), ":")
	paginatorID, action := ids[1], ids[2]
	paginator, ok := m.paginators[paginatorID]
	if !ok {
		return
	}

	if paginator.Creator != "" && paginator.Creator != e.User().ID {
		if err := e.CreateMessage(discord.NewMessageCreateBuilder().SetContent(m.config.NoPermissionMessage).SetEphemeral(true).Build()); err != nil {
			e.Client().Logger().Error("Failed to send error message: ", err)
		}
		return
	}

	switch action {
	case "first":
		paginator.currentPage = 0

	case "back":
		paginator.currentPage--

	case "stop":
		err := e.UpdateMessage(discord.MessageUpdate{Components: &[]discord.ContainerComponent{}})
		m.remove(paginatorID)
		if err != nil {
			e.Client().Logger().Error("Error updating paginator message: ", err)
		}
		return

	case "next":
		paginator.currentPage++

	case "last":
		paginator.currentPage = paginator.MaxPages - 1
	}

	paginator.expiry = time.Now()

	if err := e.UpdateMessage(m.makeMessageUpdate(paginator)); err != nil {
		e.Client().Logger().Error("Error updating paginator message: ", err)
	}
}

func (m *Manager) makeEmbed(paginator *Paginator) discord.Embed {
	embedBuilder := discord.NewEmbedBuilder().
		SetFooterText(fmt.Sprintf("Page: %d/%d", paginator.currentPage+1, paginator.MaxPages)).
		SetColor(m.config.EmbedColor)

	paginator.PageFunc(paginator.currentPage, embedBuilder)
	return embedBuilder.Build()
}

func (m *Manager) makeMessageCreate(paginator *Paginator) discord.MessageCreate {
	return discord.MessageCreate{Embeds: []discord.Embed{m.makeEmbed(paginator)}, Components: []discord.ContainerComponent{m.createComponents(paginator)}}
}

func (m *Manager) makeMessageUpdate(paginator *Paginator) discord.MessageUpdate {
	return discord.MessageUpdate{Embeds: &[]discord.Embed{m.makeEmbed(paginator)}, Components: &[]discord.ContainerComponent{m.createComponents(paginator)}}
}

func (m *Manager) formatCustomID(paginator *Paginator, action string) discord.CustomID {
	return discord.CustomID(m.config.CustomIDPrefix + ":" + paginator.ID + ":" + action)
}

func (m *Manager) createComponents(paginator *Paginator) discord.ContainerComponent {
	cfg := m.config.ButtonsConfig
	var actionRow discord.ActionRowComponent

	if cfg.First != nil {
		actionRow = actionRow.AddComponents(discord.NewButton(cfg.First.Style, cfg.First.Label, m.formatCustomID(paginator, "first"), "").WithEmoji(cfg.First.Emoji).WithDisabled(paginator.currentPage == 0))
	}
	if cfg.Back != nil {
		actionRow = actionRow.AddComponents(discord.NewButton(cfg.Back.Style, cfg.Back.Label, m.formatCustomID(paginator, "back"), "").WithEmoji(cfg.Back.Emoji).WithDisabled(paginator.currentPage == 0))
	}

	if cfg.Stop != nil {
		actionRow = actionRow.AddComponents(discord.NewButton(cfg.Stop.Style, cfg.Stop.Label, m.formatCustomID(paginator, "stop"), "").WithEmoji(cfg.Stop.Emoji))
	}

	if cfg.Next != nil {
		actionRow = actionRow.AddComponents(discord.NewButton(cfg.Next.Style, cfg.Next.Label, m.formatCustomID(paginator, "next"), "").WithEmoji(cfg.Next.Emoji).WithDisabled(paginator.currentPage == paginator.MaxPages-1))
	}
	if cfg.Last != nil {
		actionRow = actionRow.AddComponents(discord.NewButton(cfg.Last.Style, cfg.Last.Label, m.formatCustomID(paginator, "last"), "").WithEmoji(cfg.Last.Emoji).WithDisabled(paginator.currentPage == paginator.MaxPages-1))
	}

	return actionRow
}
