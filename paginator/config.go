package paginator

import "github.com/DisgoOrg/disgo/discord"

var DefaultConfig = Config{
	ButtonsConfig: ButtonsConfig{
		First: &ComponentOptions{
			Emoji: discord.ComponentEmoji{
				Name: "⏮",
			},
			Style: discord.ButtonStylePrimary,
		},
		Back: &ComponentOptions{
			Emoji: discord.ComponentEmoji{
				Name: "◀",
			},
			Style: discord.ButtonStylePrimary,
		},
		Stop: &ComponentOptions{
			Emoji: discord.ComponentEmoji{
				Name: "🗑",
			},
			Style: discord.ButtonStyleDanger,
		},
		Next: &ComponentOptions{
			Emoji: discord.ComponentEmoji{
				Name: "▶",
			},
			Style: discord.ButtonStylePrimary,
		},
		Last: &ComponentOptions{
			Emoji: discord.ComponentEmoji{
				Name: "⏩",
			},
			Style: discord.ButtonStylePrimary,
		},
	},
	CustomIDPrefix: "paginator",
	EmbedColor:     0x4c50c1,
}

type Config struct {
	ButtonsConfig  ButtonsConfig
	CustomIDPrefix string
	EmbedColor     int
}

type ButtonsConfig struct {
	First *ComponentOptions
	Back  *ComponentOptions
	Stop  *ComponentOptions
	Next  *ComponentOptions
	Last  *ComponentOptions
}

type ComponentOptions struct {
	Emoji discord.ComponentEmoji
	Label string
	Style discord.ButtonStyle
}

type ConfigOpt func(config *Config)

// Apply applies the given RequestOpt(s) to the RequestConfig & sets the context if none is set
func (c *Config) Apply(opts []ConfigOpt) {
	for _, opt := range opts {
		opt(c)
	}
}

func WithButtonsConfig(buttonsConfig ButtonsConfig) ConfigOpt {
	return func(config *Config) {
		config.ButtonsConfig = buttonsConfig
	}
}

func WithCustomIDPrefix(prefix string) ConfigOpt {
	return func(config *Config) {
		config.CustomIDPrefix = prefix
	}
}

func WithEmbedColor(color int) ConfigOpt {
	return func(config *Config) {
		config.EmbedColor = color
	}
}
