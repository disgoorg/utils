package paginator

import (
	"time"

	"github.com/DisgoOrg/disgo/discord"
	"github.com/DisgoOrg/snowflake"
)

type Paginator struct {
	PageFunc        func(page int, embed *discord.EmbedBuilder) discord.Embed
	MaxPages        int
	CurrentPage     int
	Creator         snowflake.Snowflake
	Expiry          time.Time
	ExpiryLastUsage bool
	ID              string
}
