package bot

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/jmoiron/sqlx"
)

// Bot is a Discord bot for logging, rating and reviewing books.
type Bot struct {
	session *discordgo.Session
	db      *sqlx.DB
	guildID string
}

// New creates a bot. guildID is optional: when set, slash commands are
// registered for that guild only (instant), otherwise globally (can take a while to propagate).
func New(token, guildID string, db *sqlx.DB) (*Bot, error) {
	s, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, err
	}
	s.Identify.Intents = discordgo.IntentsGuilds
	return &Bot{session: s, db: db, guildID: guildID}, nil
}

// Start connects to Discord and registers slash commands.
func (b *Bot) Start() error {
	b.session.AddHandler(b.onInteraction)

	if err := b.session.Open(); err != nil {
		return fmt.Errorf("open discord session: %w", err)
	}
	appID := b.session.State.User.ID
	if _, err := b.session.ApplicationCommandBulkOverwrite(appID, b.guildID, commands); err != nil {
		b.session.Close()
		return fmt.Errorf("register commands: %w", err)
	}
	log.Printf("Discord bot online as %s", b.session.State.User.Username)
	return nil
}

// Close disconnects from Discord.
func (b *Bot) Close() error {
	return b.session.Close()
}
