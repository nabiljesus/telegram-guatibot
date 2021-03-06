package main

import (
    "log"
    "os"
    "fmt"
    "math/rand"
    "time"
    "strings"
    "strconv"
    "sort"
    "net/http"
    "unicode/utf8"
    "github.com/go-telegram-bot-api/telegram-bot-api"

    "golang.org/x/oauth2"
    "golang.org/x/oauth2/google"
    "google.golang.org/api/sheets/v4"
)

const maxStrLen = 4096


func randomInsult() string {
    rand.Seed(time.Now().Unix())
    reasons := []string{
        "Marico.",
        "Eres gilipollas?",
        "Mamalo.",
        "Uh?",
        "Alto guaraná",
        "Naaa",
        "Marico el que lo lea",
        "Pasa pack.",
        "Cinco veces, bitch!",
        "Burro e piquete!",
        "Burro e pique!",
    }
    
    return reasons[rand.Intn(len(reasons))]
}


func splitString(longString string) []string {
    splits := []string{}

    var l, r int
    for l, r = 0, maxStrLen; r < len(longString); l, r = r, r+maxStrLen {
        for !utf8.RuneStart(longString[r]) {
            r--
        }
        splits = append(splits, longString[l:r])
    }
    splits = append(splits, longString[l:])
    return splits
}

func getSheetsService() *sheets.Service {
    creds := []byte(os.Getenv("GoogleCreds"))

    // If modifying these scopes, delete your previously saved token.json.
    config, err := google.JWTConfigFromJSON(creds, "https://www.googleapis.com/auth/spreadsheets")
    if err != nil {
            log.Printf("Unable to parse client secret file to config: %v", err)
    }
    client := config.Client(oauth2.NoContext)

    srv, err := sheets.New(client)
    if err != nil {
            log.Printf("Unable to retrieve Sheets client: %v", err)
    }

    return srv
}

func refreshSheet(srv *sheets.Service, spreadsheetId string) {
    var vr sheets.ValueRange
    var row []interface{}
    row = append(row, "Palabras")
    vr.Values = append(vr.Values, row)

    _, _ = srv.Spreadsheets.Values.Update(spreadsheetId, "A1", &vr).
                                       ValueInputOption("USER_ENTERED").Do()
}

func getRangeFromSheet(srv *sheets.Service, spreadsheetId string, cellsRange string) []string {
    refreshSheet(srv, spreadsheetId)
    resp, err := srv.Spreadsheets.Values.Get(spreadsheetId, cellsRange).Do()
    if err != nil {
            log.Printf("Unable to retrieve data from sheet: %v", err)
    }

    var words[]string

    for _, resp := range resp.Values {
        if len(resp) > 0 {
            entry := resp[0].(string)
            words = append(words, entry)
        }
    }

    return words
}

func removeDuplicates(words []string) []string {
    keys := make(map[string]bool)

    var filteredWords []string

    for _, word := range words {
        trimmedWord := strings.TrimSpace(word)
        if _, value := keys[trimmedWord]; !value && trimmedWord != "" {
            keys[trimmedWord] = true
            filteredWords = append(filteredWords, strings.Title(trimmedWord))
        }
    }

    sort.Strings(filteredWords)

    return filteredWords
}

func addToSheet(words []string) error {
    if len(words) == 0 || words[0] == "" {
        return fmt.Errorf("Lista vacia")
    }

    srv := getSheetsService()
    spreadsheetId := os.Getenv("SpreadsheetId")
    cellsRange := "Palabras!A2:A"

    existingWords := getRangeFromSheet(srv, spreadsheetId, cellsRange)
    updatedWords := removeDuplicates(append(existingWords, words...))

    var vr sheets.ValueRange

    for _, word := range updatedWords {
        var row []interface{}
        row = append(row, word)
        vr.Values = append(vr.Values, row)
    }

    _, err := srv.Spreadsheets.Values.Update(spreadsheetId, cellsRange, &vr).
                                       ValueInputOption("RAW").Do()

    return err
}

func changePercent(newPercentStr string) (string, error) {
    srv := getSheetsService()
    spreadsheetId := os.Getenv("SpreadsheetId")
    cellsRange := "Palabras!D2"

    var vr sheets.ValueRange
    var row []interface{}
    floatPercent, err := strconv.ParseFloat(strings.ReplaceAll(newPercentStr, ",", "."), 64)
    if (err != nil){
        return "", err
    }

    row = append(row, fmt.Sprintf("%.f%%", floatPercent))
    vr.Values = append(vr.Values, row)

    _, err = srv.Spreadsheets.Values.Update(spreadsheetId, cellsRange, &vr).
                                       ValueInputOption("USER_ENTERED").Do()
    return "Todo listo, mano.", err
}

func retrieveWordList() string {
    srv := getSheetsService()
    spreadsheetId := os.Getenv("SpreadsheetId")
    cellsRange := "Palabras!C2"

    wordList := getRangeFromSheet(srv, spreadsheetId, cellsRange)

    if (len(wordList) == 0) {
        return "No hay na', mano."
    } else {
        return wordList[0]
    }
}

func showHelp() string {
    return "**Guatibot Commands:** \n\n" +
        "• **/add, /AddToDibujadera, /incluir <lista,de palabras>** - Añade lista de palabras separadas por comas.\n" +
        "• **/palabras, /get, /fetch** - Obtiene la lista de palabras lista para pegar en skribbl.io\n" +
        "• **/percent, /porcentaje <Nuevo porcentaje>** Cambia el porcentaje de palabras disponibles a <nuevo porcentaje>\n"
}

func sendEhibervoice(bot *tgbotapi.BotAPI, message *tgbotapi.Message) error {
    msg := tgbotapi.NewVoiceUpload(message.Chat.ID, "resources/audios/ehiber-dark.ogg")
    msg.ReplyToMessageID = message.MessageID

    _, err := bot.Send(msg)

    return err
}

func processCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message){
    var err error
    var responseMessage string
    sendMenssage := true

    switch strings.ToLower(message.Command()) {
        case "addtodibujadera", "a", "add", "incluir", "incluye":
            err = addToSheet(strings.Split(message.CommandArguments(), ","))
            responseMessage = "Todo listo, mano."
        case "palabras", "g", "get", "fetch":
            responseMessage = retrieveWordList()
        case "help", "h", "ayuda", "comandos":
            responseMessage = showHelp()
        case "percent", "p", "porcentaje":
            responseMessage, err = changePercent(message.CommandArguments())
        case "ehibervoice", "ev":
            err = sendEhibervoice(bot, message)
        default:
           err = fmt.Errorf("Unexpected Command")
    }

    if err != nil {
        log.Printf("Command failed with %s", err)
        responseMessage = "Nolsa, mano."
    } else if sendMenssage {
        response := tgbotapi.NewMessage(message.Chat.ID, responseMessage)
        response.ReplyToMessageID = message.MessageID
        response.ParseMode = "markdown"
        multiMessage(bot, response)
    }
}

func multiMessage(bot *tgbotapi.BotAPI, response tgbotapi.MessageConfig) {
    for _, splittedMessage := range splitString(response.Text) {
        response.Text = splittedMessage
        bot.Send(response)
    }
}

func main() {
    bot, err := tgbotapi.NewBotAPI(os.Getenv("BotToken"))
    if err != nil {
        log.Printf("Error: %s", err)
    }

    bot.Debug = true
    log.Printf("Authorized on account %s", bot.Self.UserName)

    var updates tgbotapi.UpdatesChannel
    if (strings.Compare(os.Getenv("isLocal"), "true") == 0) {
            bot.RemoveWebhook()
            u := tgbotapi.NewUpdate(0)
            u.Timeout = 60
            updates, err = bot.GetUpdatesChan(u)
    } else {
        _, err = bot.SetWebhook(tgbotapi.NewWebhook(os.Getenv("UrlPath")+bot.Token))
        updates = bot.ListenForWebhook("/update" + bot.Token)
        go http.ListenAndServe(":" + os.Getenv("PORT"), nil)
    }

    if err != nil {
        log.Print(err)
    }

    for update := range updates {
        message := update.Message

        if update.Message == nil { // ignore any non-Message Updates
            message = update.ChannelPost
            if message == nil || !strings.Contains(message.Text, "@guatibot") && !message.IsCommand() {
                continue
            }
        }

        if message.IsCommand() {
            processCommand(bot, message)
        } else {
            response := tgbotapi.NewMessage(message.Chat.ID, randomInsult())
            response.ReplyToMessageID = message.MessageID
            response.ParseMode = "markdown"

            multiMessage(bot, response)
        }
    }
}
