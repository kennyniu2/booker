package bot

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/bwmarrin/discordgo"
	"github.com/jmoiron/sqlx"
)

func TestStars(t *testing.T) {
	tests := []struct {
		in   int
		want string
	}{
		{0, " (0/5)"},
		{3, "⭐⭐⭐ (3/5)"},
		{5, "⭐⭐⭐⭐⭐ (5/5)"},
		{-2, " (0/5)"},
		{9, "⭐⭐⭐⭐⭐ (5/5)"},
	}
	for _, tt := range tests {
		if got := stars(tt.in); got != tt.want {
			t.Errorf("stars(%d) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		in   string
		max  int
		want string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"this is too long", 10, "this is t…"},
		{"héllo wörld", 6, "héllo…"},
	}
	for _, tt := range tests {
		got := truncate(tt.in, tt.max)
		if got != tt.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.in, tt.max, got, tt.want)
		}
		if n := len([]rune(got)); n > tt.max {
			t.Errorf("truncate(%q, %d) returned %d runes", tt.in, tt.max, n)
		}
	}
}

func TestOptionMap(t *testing.T) {
	opts := []*discordgo.ApplicationCommandInteractionDataOption{
		{Name: "title", Type: discordgo.ApplicationCommandOptionString, Value: "Dune"},
		{Name: "rating", Type: discordgo.ApplicationCommandOptionInteger, Value: float64(4)},
	}
	m := optionMap(opts)
	if len(m) != 2 {
		t.Fatalf("len = %d, want 2", len(m))
	}
	if got := m["title"].StringValue(); got != "Dune" {
		t.Errorf("title = %q", got)
	}
	if got := m["rating"].IntValue(); got != 4 {
		t.Errorf("rating = %d", got)
	}
	if _, ok := m["review"]; ok {
		t.Error("unexpected review option")
	}
}

func TestInteractionUser(t *testing.T) {
	guild := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		Member: &discordgo.Member{User: &discordgo.User{ID: "111"}},
	}}
	if got := interactionUser(guild).ID; got != "111" {
		t.Errorf("guild user = %q, want 111", got)
	}
	if got := mention(guild); got != "<@111>" {
		t.Errorf("mention = %q", got)
	}

	dm := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		User: &discordgo.User{ID: "222"},
	}}
	if got := interactionUser(dm).ID; got != "222" {
		t.Errorf("dm user = %q, want 222", got)
	}
}

func TestCommandDefinitions(t *testing.T) {
	want := map[string]bool{"log": false, "rate": false, "shelf": false, "book": false, "remove": false}
	for _, c := range commands {
		if _, ok := want[c.Name]; !ok {
			t.Errorf("unexpected command %q", c.Name)
			continue
		}
		want[c.Name] = true
		if c.Description == "" || len(c.Description) > 100 {
			t.Errorf("%s: description length %d invalid", c.Name, len(c.Description))
		}
		seenOptional := false
		for _, o := range c.Options {
			// Discord rejects required options that follow optional ones.
			if o.Required && seenOptional {
				t.Errorf("%s: required option %q after optional option", c.Name, o.Name)
			}
			if !o.Required {
				seenOptional = true
			}
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("missing command %q", name)
		}
	}
}

func TestUserID(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer mockDB.Close()
	b := &Bot{db: sqlx.NewDb(mockDB, "sqlmock")}

	user := &discordgo.User{ID: "42"}

	mock.ExpectQuery("INSERT INTO users").
		WithArgs("42", "discord_42", "42@discord.invalid").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(7))
	id, err := b.userID(user)
	if err != nil || id != 7 {
		t.Fatalf("userID = %d, %v; want 7, nil", id, err)
	}

	mock.ExpectQuery("INSERT INTO users").WillReturnError(errors.New("boom"))
	if _, err := b.userID(user); err == nil {
		t.Fatal("expected error")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}
