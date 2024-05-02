### Discord 監控機器人

從Discord專發到Telegram



### 設定檔 config.yaml

```yaml
# 是否開啟debug log
DEBUG_LOG: false

# discord token
DISCORD_TOKEN: "xxxxxxxxxxxxxxxxx"

# telegram bot token
TELEGRAM_BOT_TOKEN: "1234355:xxxxxxx"

# telegram 轉發到哪個chat
TELEGRAM_TO_CHATID: -439250319

# 監聽discord的channel [group:channel]
LISTEN_GUILD_CHANNEL_IDS:
  - "10047078866376999011:1002716795539771552" 


# 黑名單 [group:channel]
BLACKLIST_GUILD_CHANNEL_IDS:
  - "123456789:*"

# 監聽discord的某個userid
LISTEN_USER_IDS:
  - "123456789"
```


### 執行
```
./dicord -c config.yaml
```

### 查看所有群組頻道ID
```
./dicord -c config.yaml list
```