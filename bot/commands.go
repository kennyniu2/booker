package bot

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

var (
	minRating = 1.0
	maxRating = 5.0
)

var statusChoices = []*discordgo.ApplicationCommandOptionChoice{
	{Name: "Reading", Value: "reading"},
	{Name: "Completed", Value: "completed"},
	{Name: "Want to read", Value: "want-to-read"},
}

var statusEmoji = map[string]string{
	"reading":      "📖",
	"completed":    "✅",
	"want-to-read": "🔖",
}

var commands = []*discordgo.ApplicationCommand{
	{
		Name:        "log",
		Description: "Add a book to your shelf (optionally with a rating and review)",
		Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionString, Name: "title", Description: "Book title", Required: true, MaxLength: 255},
			{Type: discordgo.ApplicationCommandOptionString, Name: "author", Description: "Book author", Required: true, MaxLength: 255},
			{Type: discordgo.ApplicationCommandOptionString, Name: "status", Description: "Reading status (default: completed)", Choices: statusChoices},
			{Type: discordgo.ApplicationCommandOptionInteger, Name: "rating", Description: "Rating from 1 to 5", MinValue: &minRating, MaxValue: maxRating},
			{Type: discordgo.ApplicationCommandOptionString, Name: "review", Description: "Your review", MaxLength: 1500},
		},
	},
	{
		Name:        "rate",
		Description: "Rate and review a book on your shelf",
		Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionString, Name: "book", Description: "Book from your shelf", Required: true, Autocomplete: true},
			{Type: discordgo.ApplicationCommandOptionInteger, Name: "rating", Description: "Rating from 1 to 5", Required: true, MinValue: &minRating, MaxValue: maxRating},
			{Type: discordgo.ApplicationCommandOptionString, Name: "review", Description: "Your review", MaxLength: 1500},
		},
	},
	{
		Name:        "shelf",
		Description: "Show a reader's books",
		Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionUser, Name: "user", Description: "Whose shelf (default: yours)"},
			{Type: discordgo.ApplicationCommandOptionString, Name: "status", Description: "Filter by status", Choices: statusChoices},
		},
	},
	{
		Name:        "book",
		Description: "Look up a book's average rating and reviews",
		Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionString, Name: "title", Description: "Book title (partial match is fine)", Required: true},
		},
	},
	{
		Name:        "remove",
		Description: "Remove a book from your shelf",
		Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionString, Name: "book", Description: "Book from your shelf", Required: true, Autocomplete: true},
		},
	},
}

func (b *Bot) onInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		var err error
		switch name := i.ApplicationCommandData().Name; name {
		case "log":
			err = b.handleLog(s, i)
		case "rate":
			err = b.handleRate(s, i)
		case "shelf":
			err = b.handleShelf(s, i)
		case "book":
			err = b.handleBook(s, i)
		case "remove":
			err = b.handleRemove(s, i)
		}
		if err != nil {
			log.Printf("command %s failed: %v", i.ApplicationCommandData().Name, err)
			var ue userError
			msg := "Something went wrong, please try again."
			if errors.As(err, &ue) {
				msg = ue.Error()
			}
			respond(s, i, msg, nil, true)
		}
	case discordgo.InteractionApplicationCommandAutocomplete:
		b.handleAutocomplete(s, i)
	}
}

// userError is an error whose message is safe to show to the user.
type userError string

func (e userError) Error() string { return string(e) }

func (b *Bot) handleLog(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	opts := optionMap(i.ApplicationCommandData().Options)
	title := strings.TrimSpace(opts["title"].StringValue())
	author := strings.TrimSpace(opts["author"].StringValue())
	if title == "" || author == "" {
		return userError("Title and author can't be empty.")
	}
	status := "completed"
	if o, ok := opts["status"]; ok {
		status = o.StringValue()
	}
	var rating *int64
	if o, ok := opts["rating"]; ok {
		v := o.IntValue()
		rating = &v
	}
	var review string
	if o, ok := opts["review"]; ok {
		review = strings.TrimSpace(o.StringValue())
	}

	userID, err := b.userID(interactionUser(i))
	if err != nil {
		return err
	}

	var bookID int
	err = b.db.QueryRow(
		`SELECT id FROM books WHERE LOWER(title) = LOWER($1) AND LOWER(author) = LOWER($2) LIMIT 1`,
		title, author).Scan(&bookID)
	if errors.Is(err, sql.ErrNoRows) {
		err = b.db.QueryRow(
			`INSERT INTO books (title, author) VALUES ($1, $2) RETURNING id`,
			title, author).Scan(&bookID)
	}
	if err != nil {
		return err
	}

	_, err = b.db.Exec(`
		INSERT INTO user_books (user_id, book_id, status, rating, review)
		VALUES ($1, $2, $3, $4, NULLIF($5, ''))
		ON CONFLICT (user_id, book_id) DO UPDATE SET
			status = EXCLUDED.status,
			rating = COALESCE(EXCLUDED.rating, user_books.rating),
			review = COALESCE(EXCLUDED.review, user_books.review),
			updated_at = NOW()`,
		userID, bookID, status, rating, review)
	if err != nil {
		return err
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s %s", statusEmoji[status], title),
		Description: fmt.Sprintf("by %s", author),
		Color:       0x5865F2,
		Footer:      &discordgo.MessageEmbedFooter{Text: "Status: " + status},
	}
	if rating != nil {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "Rating", Value: stars(int(*rating))})
	}
	if review != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "Review", Value: review})
	}
	return respond(s, i, fmt.Sprintf("%s added to their shelf:", mention(i)), embed, false)
}

func (b *Bot) handleRate(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	opts := optionMap(i.ApplicationCommandData().Options)
	userBookID, err := strconv.Atoi(opts["book"].StringValue())
	if err != nil {
		return userError("Please pick a book from the suggestion list.")
	}
	rating := opts["rating"].IntValue()
	var review string
	if o, ok := opts["review"]; ok {
		review = strings.TrimSpace(o.StringValue())
	}

	userID, err := b.userID(interactionUser(i))
	if err != nil {
		return err
	}

	var title, author string
	err = b.db.QueryRow(`
		WITH updated AS (
			UPDATE user_books
			SET rating = $1, review = COALESCE(NULLIF($2, ''), review), status = 'completed', updated_at = NOW()
			WHERE id = $3 AND user_id = $4
			RETURNING book_id
		)
		SELECT b.title, b.author FROM updated u JOIN books b ON b.id = u.book_id`,
		rating, review, userBookID, userID).Scan(&title, &author)
	if errors.Is(err, sql.ErrNoRows) {
		return userError("That book isn't on your shelf. Use `/log` to add it first.")
	}
	if err != nil {
		return err
	}

	embed := &discordgo.MessageEmbed{
		Title:       title,
		Description: fmt.Sprintf("by %s", author),
		Color:       0xFEE75C,
		Fields:      []*discordgo.MessageEmbedField{{Name: "Rating", Value: stars(int(rating))}},
	}
	if review != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "Review", Value: review})
	}
	return respond(s, i, fmt.Sprintf("%s rated a book:", mention(i)), embed, false)
}

func (b *Bot) handleShelf(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	opts := optionMap(i.ApplicationCommandData().Options)
	target := interactionUser(i)
	if o, ok := opts["user"]; ok {
		target = o.UserValue(s)
	}
	status := ""
	if o, ok := opts["status"]; ok {
		status = o.StringValue()
	}

	var rows []struct {
		Title  string        `db:"title"`
		Author string        `db:"author"`
		Status string        `db:"status"`
		Rating sql.NullInt64 `db:"rating"`
	}
	err := b.db.Select(&rows, `
		SELECT bk.title, bk.author, ub.status, ub.rating
		FROM user_books ub
		JOIN users u ON u.id = ub.user_id
		JOIN books bk ON bk.id = ub.book_id
		WHERE u.discord_id = $1 AND ($2 = '' OR ub.status = $2)
		ORDER BY ub.updated_at DESC
		LIMIT 25`, target.ID, status)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return respond(s, i, fmt.Sprintf("<@%s> has no books here yet.", target.ID), nil, true)
	}

	var sb strings.Builder
	for _, r := range rows {
		fmt.Fprintf(&sb, "%s **%s** — %s", statusEmoji[r.Status], r.Title, r.Author)
		if r.Rating.Valid {
			fmt.Fprintf(&sb, " · %s", stars(int(r.Rating.Int64)))
		}
		sb.WriteString("\n")
	}
	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s's shelf", target.Username),
		Description: truncate(sb.String(), 4000),
		Color:       0x57F287,
		Footer:      &discordgo.MessageEmbedFooter{Text: "Showing the 25 most recent books"},
	}
	return respond(s, i, "", embed, false)
}

func (b *Bot) handleBook(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	query := strings.TrimSpace(i.ApplicationCommandData().Options[0].StringValue())

	var book struct {
		ID      int             `db:"id"`
		Title   string          `db:"title"`
		Author  string          `db:"author"`
		Average sql.NullFloat64 `db:"avg"`
		Ratings int             `db:"ratings"`
	}
	err := b.db.Get(&book, `
		SELECT bk.id, bk.title, bk.author, AVG(ub.rating)::float8 AS avg, COUNT(ub.rating) AS ratings
		FROM books bk
		LEFT JOIN user_books ub ON ub.book_id = bk.id
		WHERE bk.title ILIKE '%' || $1 || '%'
		GROUP BY bk.id
		ORDER BY COUNT(ub.id) DESC
		LIMIT 1`, query)
	if errors.Is(err, sql.ErrNoRows) {
		return respond(s, i, "No book found matching that title.", nil, true)
	}
	if err != nil {
		return err
	}

	embed := &discordgo.MessageEmbed{
		Title:       book.Title,
		Description: fmt.Sprintf("by %s", book.Author),
		Color:       0xEB459E,
	}
	if book.Average.Valid {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:  "Average rating",
			Value: fmt.Sprintf("%.1f / 5 (%d ratings)", book.Average.Float64, book.Ratings),
		})
	} else {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "Average rating", Value: "No ratings yet"})
	}

	var reviews []struct {
		DiscordID sql.NullString `db:"discord_id"`
		Rating    sql.NullInt64  `db:"rating"`
		Review    string         `db:"review"`
	}
	err = b.db.Select(&reviews, `
		SELECT u.discord_id, ub.rating, ub.review
		FROM user_books ub JOIN users u ON u.id = ub.user_id
		WHERE ub.book_id = $1 AND ub.review IS NOT NULL AND ub.review <> ''
		ORDER BY ub.updated_at DESC
		LIMIT 5`, book.ID)
	if err != nil {
		return err
	}
	for _, r := range reviews {
		name := "Reader"
		if r.DiscordID.Valid {
			name = fmt.Sprintf("<@%s>", r.DiscordID.String)
		}
		if r.Rating.Valid {
			name += " · " + stars(int(r.Rating.Int64))
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "Review", Value: truncate(name+"\n"+r.Review, 1000)})
	}
	return respond(s, i, "", embed, false)
}

func (b *Bot) handleRemove(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	userBookID, err := strconv.Atoi(i.ApplicationCommandData().Options[0].StringValue())
	if err != nil {
		return userError("Please pick a book from the suggestion list.")
	}
	userID, err := b.userID(interactionUser(i))
	if err != nil {
		return err
	}
	res, err := b.db.Exec(`DELETE FROM user_books WHERE id = $1 AND user_id = $2`, userBookID, userID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return userError("That book isn't on your shelf.")
	}
	return respond(s, i, "Removed from your shelf.", nil, true)
}

// handleAutocomplete suggests books from the invoking user's shelf.
func (b *Bot) handleAutocomplete(s *discordgo.Session, i *discordgo.InteractionCreate) {
	var typed string
	for _, o := range i.ApplicationCommandData().Options {
		if o.Focused {
			typed = o.StringValue()
		}
	}

	var rows []struct {
		ID     int    `db:"id"`
		Title  string `db:"title"`
		Author string `db:"author"`
	}
	err := b.db.Select(&rows, `
		SELECT ub.id, bk.title, bk.author
		FROM user_books ub
		JOIN users u ON u.id = ub.user_id
		JOIN books bk ON bk.id = ub.book_id
		WHERE u.discord_id = $1 AND bk.title ILIKE '%' || $2 || '%'
		ORDER BY ub.updated_at DESC
		LIMIT 25`, interactionUser(i).ID, typed)
	if err != nil {
		log.Printf("autocomplete failed: %v", err)
	}

	choices := make([]*discordgo.ApplicationCommandOptionChoice, 0, len(rows))
	for _, r := range rows {
		choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
			Name:  truncate(fmt.Sprintf("%s — %s", r.Title, r.Author), 100),
			Value: strconv.Itoa(r.ID),
		})
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{Choices: choices},
	})
}

// userID returns the internal user id for a Discord user, creating the user on first use.
func (b *Bot) userID(u *discordgo.User) (int, error) {
	var id int
	err := b.db.QueryRow(`
		INSERT INTO users (discord_id, username, email, password_hash)
		VALUES ($1, $2, $3, '')
		ON CONFLICT (discord_id) DO UPDATE SET discord_id = EXCLUDED.discord_id
		RETURNING id`,
		u.ID, "discord_"+u.ID, u.ID+"@discord.invalid").Scan(&id)
	return id, err
}

func optionMap(opts []*discordgo.ApplicationCommandInteractionDataOption) map[string]*discordgo.ApplicationCommandInteractionDataOption {
	m := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(opts))
	for _, o := range opts {
		m[o.Name] = o
	}
	return m
}

// interactionUser returns the invoking user, whether in a guild or a DM.
func interactionUser(i *discordgo.InteractionCreate) *discordgo.User {
	if i.Member != nil {
		return i.Member.User
	}
	return i.User
}

func mention(i *discordgo.InteractionCreate) string {
	return "<@" + interactionUser(i).ID + ">"
}

func respond(s *discordgo.Session, i *discordgo.InteractionCreate, content string, embed *discordgo.MessageEmbed, ephemeral bool) error {
	data := &discordgo.InteractionResponseData{Content: content}
	if embed != nil {
		data.Embeds = []*discordgo.MessageEmbed{embed}
	}
	if ephemeral {
		data.Flags = discordgo.MessageFlagsEphemeral
	}
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: data,
	})
}

func stars(n int) string {
	if n < 0 {
		n = 0
	}
	if n > 5 {
		n = 5
	}
	return strings.Repeat("⭐", n) + fmt.Sprintf(" (%d/5)", n)
}

func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-1]) + "…"
}
