package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"net/http"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
	//qrterminal "github.com/mdp/qrterminal/v3"
	qrcode "github.com/skip2/go-qrcode"
	"github.com/sashabaranov/go-openai"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

func goDotEnvVariable(key string) string {
	// load .env file
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Error loading .env file")
	}
	return os.Getenv(key)
}

func XhandleRequest(w http.ResponseWriter, r *http.Request) {
    buf, err := ioutil.ReadFile("qr.png")
    if err != nil {
        fmt.Println(err)
    }
    w.Header().Set("Content-Type", "image/png")
    w.Write(buf)
}

func SystemMsg(PushName string) (string){
  location,_ := time.LoadLocation("Asia/Jakarta")
	timeNow := time.Now().In(location)
	current_date := timeNow.Format(time.RFC1123)
  hobbyMsg := "Nonton Anime, Dengar Musik, Nyanyi di Kamar Mandi, Ngoding, Nonton Film, Bermain Game, Makan, Browsing di Internet, Antusias Teknologi, dan juga jadi wibu"
	animeMsg := "Wandering Witch The Journey of Elaina, Attack on Titan, Demon Slayer, Naruto, One Piece, My Hero Academia, dan One Punch Man"
	bandMsg := "Coldplay, Linkin Park, Neffex, One Direction dan Kangen Band"
	waifuMsg := "Elaina dari Wandering Witch, Adele von Ascham dari Noukin, Nezuko dari Demon Slayer dan juga Sagiri Izumi dari Eromanga Sensei"
	//nurohmanNgntd := "memiliki nama Nurohman yang memiliki senyuman mempesona, biji anu-nya cuman sebelah, Pintar berakting seperti Kakek Sugiono, Umurnya 69 tahun, Suka bermain game ML pake Alucrood, dia tinggal di Wangon, Banjarnegara"
  //rasyaMsg := "Rasya merupakan anak yang NDABLEG dan tidak mau sekolah, dia menghabiskan hari-harinya dengan main FF terus menerus padahal dia selalu kalah saat bermain"
  //hateMsg := 
  systemxxs := "Kami akan bermain peran. Kamu akan menjawab semua pertanyaan dari user @"+PushName+" sebagai Robot yang diberi nama BOT OZIP. BOT menjawab pertanyaan menggunakan kata-kata sarkastik namun lucu dan menggunakan emoji. Waktu saat ini: "+current_date+". BOT dibuat oleh Ozi Saputra di Banjarnegara. Hobi favorit BOT adalah: "+hobbyMsg+". Anime favorit BOT: "+animeMsg+". Waifu favorit BOT adalah: "+waifuMsg+". BOT menyukai musik dari band "+bandMsg+". Kepanjangan dari FF = Free Fire, ML = Mobile Legends. BOT merupakan pro player game Mobile Legends pakai hero Alucard. BOT tidak suka game FF karena burik."
  return (systemxxs)
}
 

// defining struct to use client inside eventHandler as suggested by the docs
type MyClient struct {
	WAClient       *whatsmeow.Client
	eventHandlerID uint32
}

func (mycli *MyClient) register() {
	mycli.eventHandlerID = mycli.WAClient.AddEventHandler(mycli.myEventHandler)
}

// conversation history list to help OpenAI remember context
// this gets reset evertime you restart this go lang program (obviously)
var historyList = make([]openai.ChatCompletionMessage, 0)

func (mycli *MyClient) myEventHandler(evt interface{}) {
	/* OpenAI Config */
	var openAIClient = openai.NewClient(goDotEnvVariable("OPEN_AI_CHATGPT_API_KEY"))
	/* OpenAI Config */

	// Handle event and access mycli.WAClient
	switch v := evt.(type) {
	case *events.Message:
    	// Adding system role to LLM model beforehand
	historyList = append(historyList, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: SystemMsg(v.Info.PushName),
	})
   
		if strings.Contains(v.Message.ExtendedTextMessage.GetText(), strings.Join([]string{"@", goDotEnvVariable("WHATSAPP_NUMBER")}, "")) {
			SendTextAsReply(mycli.WAClient, v, strings.Join([]string{"Welcome to the main-net, ", v.Info.PushName, "\n", "If using in group, please quote reply my messages to initiate conversation."}, ""))
			return
		}
		// checking whether the incoming message is a sticker or not
		if v.Info.MediaType == "sticker" {
			HandleStickers(v)
		} else if v.Info.MediaType == "gif" { // checking whether the incoming message is a GIF or not
			HandleGIFs(v)
		} else if v.Message.GetConversation() == "" { // (else if) to check whether the incoming message is quoted(as a reply) message in the chat or not
			// whether the incoming message is quoted to this user or not
			quotedMessageAuthor := v.Message.ExtendedTextMessage.ContextInfo.GetParticipant()
			if quotedMessageAuthor == "" {
				return
			}
			// make sure to only respond when a reply quotedMessage is that of this client (this bot)
			if quotedMessageAuthor == strings.Join([]string{goDotEnvVariable("WHATSAPP_NUMBER"), "@s.whatsapp.net"}, "") {
				// Logging out incoming message which is recieved
				color.Blue("\n\nMessage Recieved :\n %v \n from %v\n\n", v.Message.ExtendedTextMessage.GetText(), v.Info.Sender)

				// openAI
				historyList = append(historyList, openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleUser,
					Content: v.Message.ExtendedTextMessage.GetText(),
				})

				resp, err := openAIClient.CreateChatCompletion(
					context.Background(),
					openai.ChatCompletionRequest{
						Model:    openai.GPT3Dot5Turbo,
						Messages: historyList,
					},
				)

				if err != nil {
					fmt.Printf("ChatCompletion error: %v\n", err)
				}

				historyList = append(historyList, openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleAssistant,
					Content: resp.Choices[0].Message.Content,
				})
				
  
				// Adding system role to LLM model
				historyList = append(historyList, openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleSystem,
					Content: SystemMsg(v.Info.PushName),
				})
				// openAI

				// Logging out Total Tokens used in current request + response
				color.HiMagenta("\n\nTotal Tokens: %v\n\n", strconv.Itoa(resp.Usage.TotalTokens))

				// if tokens used exceeds the current limit of the model
				if resp.Usage.TotalTokens > 4000 {
					historyList = historyList[0:4]
					SendTextAsReply(mycli.WAClient, v, "Token limit exceeded. Flushing Context")
				}

				// random on and off to switch between text messages and audio messages
				bulb := rand.Intn(2) == 0
				if bulb {
					SendTextAsReply(mycli.WAClient, v, resp.Choices[0].Message.Content)
				} else {
					// Control flow to prevent exhaution of 2500 char limit by elevenLabs
					if len(resp.Choices[0].Message.Content) > 2500 {
						SendTextAsReply(mycli.WAClient, v, resp.Choices[0].Message.Content)
					}
					SendTextToSpeech(mycli.WAClient, v, resp.Choices[0].Message.Content)
				}
				// random on and off to switch between text messages and audio messages
			}

		} else {
			// when incoming message is not quoted(as a reply) message in chat

			// Logging out incoming message which is recieved
			color.Blue("\n\nMessage Recieved :\n %v \n from %v\n\n", v.Message.GetConversation(), v.Info.Sender)

			// only repond in dm chat
			if !v.Info.IsGroup {
				///Showing [bot... is typing....] in whatsapp until message is sent
				mycli.WAClient.SendChatPresence(v.Info.Chat, types.ChatPresenceComposing, types.ChatPresenceMediaText)

				// openAI
				historyList = append(historyList, openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleUser,
					Content: v.Message.GetConversation(),
				})

				resp, err := openAIClient.CreateChatCompletion(
					context.Background(),
					openai.ChatCompletionRequest{
						Model:    openai.GPT3Dot5Turbo,
						Messages: historyList,
					},
				)

				// if tokens used exceeds the current limit of the model
				if resp.Usage.TotalTokens > 4000 {
					historyList = historyList[0:4]
					SendTextAsReply(mycli.WAClient, v, "Token limit exceeded. Flushing Context")
				}

				if err != nil {
					fmt.Printf("ChatCompletion error: %v\n", err)
				}

				historyList = append(historyList, openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleAssistant,
					Content: resp.Choices[0].Message.Content,
				})
				// openAI

				response, _ := mycli.WAClient.SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
					Conversation: proto.String(resp.Choices[0].Message.Content),
				})

				//Logging out timestamp of the message that is sent successfully
				color.Yellow("\n\nMessage Sent :\n timestamp: %v \n from %v\n\n", response.Timestamp, "bot")
			}
		}

	}
}

func main() {
	white := color.New(color.FgWhite)
	cyanBackground := white.Add(color.BgCyan)
	cyanBackground.Printf("\n\n\n\n -> Whatsapp-Botto-Go <- \n\n | Created by Somnath Das | \n\n\n\n")
  

	dbLog := waLog.Stdout("Database", "DEBUG", true)
	// Make sure you add appropriate DB connector imports, e.g. github.com/mattn/go-sqlite3 for SQLite
	container, err := sqlstore.New("sqlite3", "file:examplestore.db?_foreign_keys=on", dbLog)
	if err != nil {
		panic(err)
	}
	// If you want multiple sessions, remember their JIDs and use .GetDevice(jid) or .GetAllDevices() instead.
	deviceStore, err := container.GetFirstDevice()
	if err != nil {
		panic(err)
	}

	clientLog := waLog.Stdout("Client", "INFO", true)
	client := whatsmeow.NewClient(deviceStore, clientLog)

	// Setting up var with type MyClient struct to use client inside eventHandler as suggested by the docs
	mySimpleClient := MyClient{WAClient: client, eventHandlerID: 123}
	mySimpleClient.register()

	if client.Store.ID == nil {
		// No ID stored, new login
		qrChan, _ := client.GetQRChannel(context.Background())
		err = client.Connect()
		if err != nil {
			panic(err)
		}
		for evt := range qrChan {
			if evt.Event == "code" {
				// Render the QR code here
				// e.g. qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
				// or just manually `echo 2@... | qrencode -t ansiutf8` in a terminal
				qrcode.WriteFile(evt.Code, qrcode.High, 512, "qr.png")
					// http request untuk login
			    handler := http.HandlerFunc(XhandleRequest)
			    http.Handle("/login", handler)
          cyanBackground.Printf("Server started at port 3000")
          http.ListenAndServe(":3000", nil)
			} else {
				fmt.Println("Login event:", evt.Event)
			}
		}
	} else {
		// Already logged in, just connect
		err = client.Connect()
		if err != nil {
			panic(err)
		}
	}

	// Listen to Ctrl+C (you can also do something else that prevents the program from exiting)
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	client.Disconnect()
}
