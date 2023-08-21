package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/bwmarrin/discordgo"
	"github.com/spf13/viper"
	"github.com/urfave/cli"
)

// envStruct type
type envStruct struct {
	DiscordToken     string `mapstructure:"DISCORD_TOKEN" json:"DISCORD_TOKEN"`
	DebugLog         bool   `mapstructure:"DEBUG_LOG" json:"DEBUG_LOG"`
	TelegramBotToken string `mapstructure:"TELEGRAM_BOT_TOKEN" json:"TELEGRAM_BOT_TOKEN"`
	TelegramToChatid int64  `mapstructure:"TELEGRAM_TO_CHATID" json:"TELEGRAM_TO_CHATID"`

	ListenGuildChannelIds    []string `mapstructure:"LISTEN_GUILD_CHANNEL_IDS" json:"LISTEN_GUILD_CHANNEL_IDS"`
	ListenUserIds            []string `mapstructure:"LISTEN_USER_IDS" json:"LISTEN_USER_IDS"`
	BlacklistGuildChannelIds []string `mapstructure:"BLACKLIST_GUILD_CHANNEL_IDS" json:"BLACKLIST_GUILD_CHANNEL_IDS"`
}

var (
	env                              envStruct
	bot                              *tgbotapi.BotAPI
	dg                               *discordgo.Session
	ListenGuildChannelIdsMapSlice    map[string][]string
	BlacklistGuildChannelIdsMapSlice map[string][]string
)

func main() {
	app := cli.NewApp()
	app.Name = "discord-bot"
	app.Version = "v0.0.1"
	app.Authors = []cli.Author{
		cli.Author{
			Name: "Ken",
		},
	}

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "config, c",
			Value:  "config.yaml",
			Usage:  "app config",
			EnvVar: "CONFIG_PATH",
		},
	}

	app.Action = run

	err := app.Run(os.Args)

	if err != nil {
		panic(err)
	}
}

func run(c *cli.Context) {
	viper.SetConfigFile(c.String("config"))
	viper.AutomaticEnv()

	err := viper.ReadInConfig()

	if err != nil {
		panic(err)
	}

	err = viper.Unmarshal(&env)

	if err != nil {
		panic(err)
	}

	log.Println("ENV:", env)
	log.Println("Cofing 設定成功")

	// 監聽名單
	ListenGuildChannelIdsMapSlice = make(map[string][]string)

	for _, val := range env.ListenGuildChannelIds {
		vals := strings.Split(val, ":")

		if len(vals) == 2 {
			ListenGuildChannelIdsMapSlice[vals[0]] = append(ListenGuildChannelIdsMapSlice[vals[0]], vals[1])
		}
	}

	// 黑名單
	BlacklistGuildChannelIdsMapSlice = make(map[string][]string)

	for _, val := range env.BlacklistGuildChannelIds {
		vals := strings.Split(val, ":")

		if len(vals) == 2 {
			BlacklistGuildChannelIdsMapSlice[vals[0]] = append(BlacklistGuildChannelIdsMapSlice[vals[0]], vals[1])
		}
	}

	bot, err = tgbotapi.NewBotAPI(env.TelegramBotToken)

	if err != nil {
		panic(err)
	}

	bot.Debug = false

	fmt.Printf("Authorized on account %s", bot.Self.UserName)

	go discordrun()

	select {}
}

func discordrun() {
	var err error

	dg, err = discordgo.New(env.DiscordToken)

	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	// Register the messageCreate func as a callback for MessageCreate events.
	dg.AddHandler(messageCreate)

	// In this example, we only care about receiving message events.
	dg.Identify.Intents = discordgo.IntentsAll
	// dg.Identify.Intents = discordgo.IntentsGuildMessages

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")

	sc := make(chan os.Signal, 1)

	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	dg.Close()
}

// getGuild 取的guild物件 by guild id
func getGuild(gs []*discordgo.Guild, id string) *discordgo.Guild {
	for _, guild := range gs {
		if guild.ID == id {
			return guild
		}
	}

	return nil
}

// getChannel 取的channel物件 by channelid
func getChannel(cs []*discordgo.Channel, channelId string) *discordgo.Channel {
	for _, channel := range cs {
		if channel.ID == channelId {
			return channel
		}
	}

	return nil
}

// checkIdExist 檢查id是否存在list
func checkIdExist(ids []string, id string) bool {
	for _, v := range ids {
		// 如果是星號就代表都過
		if v == "*" {
			return true
		}

		if v == id {
			return true
		}
	}

	return false
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the authenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	blacklistChannelIds, exist := BlacklistGuildChannelIdsMapSlice[m.GuildID]

	// 如果在黑名單裡面就直接結束
	if exist && checkIdExist(blacklistChannelIds, m.ChannelID) {
		return
	}

	channelIds, exist := ListenGuildChannelIdsMapSlice[m.GuildID]

	// 如果在監控名單才會傳送給TG
	isSendBot := checkIdExist(env.ListenUserIds, m.Author.ID) || (exist && checkIdExist(channelIds, m.ChannelID))

	msg := ""

	msglog := fmt.Sprintf("\nuser id: %s \nuser name: %s \n",
		m.Author.ID,
		m.Author.Username,
	)

	guild := getGuild(dg.State.Guilds, m.GuildID)

	if guild != nil {
		channel := getChannel(guild.Channels, m.ChannelID)

		if channel != nil {
			if isSendBot {
				msg = msg + fmt.Sprintf("名字:  %s \n群組:  %s \n頻道:  %s \n",
					m.Author.Username,
					guild.Name,
					channel.Name,
				)
			}

			msglog = msglog + fmt.Sprintf("guild id: %s \nguild name: %s \nchannel id: %s \nchannel name: %s \n",
				m.GuildID,
				guild.Name,
				m.ChannelID,
				channel.Name,
			)
		}
	}

	if isSendBot {
		msg = msg + fmt.Sprintf("內容:\n%s \n", m.Content)
	}

	msglog = msglog + fmt.Sprintf("內容:\n%s \n", m.Content)

	for _, attachment := range m.Attachments {
		if isSendBot {
			msg = msg + fmt.Sprintf("附件:\n%s \n", attachment.URL)
		}

		msglog = msglog + fmt.Sprintf("附件:\n%s \n", attachment.URL)
	}

	// 處理Embeds
	for _, embed := range m.Embeds {
		if embed.URL != "" {
			if isSendBot {
				msg = msg + fmt.Sprintf("Embed URL: %s \n", embed.URL)
			}
			msglog = msglog + fmt.Sprintf("Embed: %s \n", embed.URL)
		}

		if embed.Title != "" {
			if isSendBot {
				msg = msg + fmt.Sprintf("Embed Title: %s \n", embed.Title)
			}
			msglog = msglog + fmt.Sprintf("Embed Title: %s \n", embed.Title)
		}

		if embed.Description != "" {
			if isSendBot {
				msg = msg + fmt.Sprintf("Embed Description: %s \n", embed.Description)
			}
			msglog = msglog + fmt.Sprintf("Embed Description: %s \n", embed.Description)
		}

		if embed.Image != nil {
			if isSendBot {
				msg = msg + fmt.Sprintf("Embed Image URL: %s \n", embed.Image.URL)
			}
			msglog = msglog + fmt.Sprintf("Embed Image URL: %s \n", embed.Image.URL)
		}

		if embed.Video != nil {
			if isSendBot {
				msg = msg + fmt.Sprintf("Embed Video URL: %s \n", embed.Video.URL)
			}
			msglog = msglog + fmt.Sprintf("Embed Video URL: %s \n", embed.Video.URL)
		}

		if embed.Provider != nil {
			if isSendBot {
				msg = msg + fmt.Sprintf("Embed Provider URL: %s \n", embed.Provider.URL)
			}
			msglog = msglog + fmt.Sprintf("Embed Provider URL: %s \n", embed.Provider.URL)
		}

		if embed.Footer != nil {
			if isSendBot {
				msg = msg + fmt.Sprintf("Embed Footer Text: %s \n", embed.Footer.Text)
			}
			msglog = msglog + fmt.Sprintf("Embed Footer Text: %s \n", embed.Footer.Text)
		}

		if embed.Footer != nil {
			if isSendBot {
				msg = msg + fmt.Sprintf("Embed Footer Text: %s \n", embed.Footer.Text)
			}
			msglog = msglog + fmt.Sprintf("Embed Footer Text: %s \n", embed.Footer.Text)
		}

		for _, f := range embed.Fields {
			if isSendBot {
				msg = msg + fmt.Sprintf("Embed Value Text: %s \n", f.Value)
			}
			msglog = msglog + fmt.Sprintf("Embed Value Text: %s \n", f.Value)
		}
	}

	if env.DebugLog {
		log.Println(msglog)
	}

	if isSendBot {
		// log.Println(msglog)
		// fmt.Println("m.Attachments", m.Attachments)
		// fmt.Println("m.Components", m.Components)
		// if len(m.Components) != 0 {
		// 	fmt.Println("m.Components[0]", m.Components[0])
		// }
		// fmt.Println("m.Embeds", m.Embeds)
		// if len(m.Embeds) != 0 {
		// 	fmt.Println("m.Embeds[0]", m.Embeds[0])
		// }
		// fmt.Println("m.MessageReference", m.MessageReference)
		// fmt.Println("m.ReferencedMessage", m.ReferencedMessage) // 參考訊息
		// fmt.Println("m.Interaction", m.Interaction)

		// msg = "```\n" + msg + "```"

		botSendMsg(env.TelegramToChatid, 0, msg, tgbotapi.ModeMarkdownV2)
	}

	// // If the message is "ping" reply with "Pong!"
	// if m.Content == "ping" {
	//  fmt.Println("#########################")
	//  s.ChannelMessageSend(m.ChannelID, "Pong!")
	// }
}

func botSendMsg(chatId int64, replyToMessageID int, msg string, parseMode string) {
	chattable := tgbotapi.NewMessage(
		chatId,
		msg,
	)

	// chattable.ParseMode = parseMode
	// chattable.DisableWebPagePreview = false

	// if replyToMessageID != 0 {
	// 	chattable.ReplyToMessageID = replyToMessageID
	// }

	_, err := bot.Send(chattable)

	if err != nil {
		log.Println("bbotSendMsgot send msg err: ", err, "chatid: ", chatId)
	}
}
