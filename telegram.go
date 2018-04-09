package bankldapService

import (
	"gopkg.in/telegram-bot-api.v4"
	"github.com/weAutomateEverything/go2hal/telegram"
	"golang.org/x/net/context"
	"net/http"
	"os"
	"fmt"
	"encoding/json"
	"io/ioutil"
	"net/smtp"
	"log"
	"net"
	"math/rand"
	"strconv"
)

type register struct {
	service telegram.Service
	store   Store
}

func NewRegisterCommand(service telegram.Service, store Store) telegram.Command {
	return &register{service: service, store: store}
}

func (*register) CommandIdentifier() string {
	return "Register"
}

func (*register) CommandDescription() string {
	return "Register yourself as a user. Add your employee number to the call"
}

func (s *register) Execute(update tgbotapi.Update) {
	arg := update.Message.CommandArguments()
	if len(arg) == 0 {
		s.service.SendMessage(context.TODO(), update.Message.Chat.ID, "to register, please type /Register <employee number>", update.Message.MessageID)
		return
	}

	log.Println(getLdapEndpoint() + arg)
	resp, err := http.Get(getLdapEndpoint() + arg)
	if err != nil {
		s.service.SendMessage(context.TODO(), update.Message.Chat.ID, fmt.Sprintf("Unable to lookup user %v, error from http %v", arg, err.Error()),
			update.Message.MessageID)
		return
	}

	var dat map[string]interface{}
	b, err := ioutil.ReadAll(resp.Body)
	log.Println(string(b))
	if err != nil {
		s.service.SendMessage(context.TODO(), update.Message.Chat.ID, fmt.Sprintf("Unable to lookup user %v, error reding body %v", arg, err.Error()),
			update.Message.MessageID)
		return
	}

	if err := json.Unmarshal(b, &dat); err != nil {
		s.service.SendMessage(context.TODO(), update.Message.Chat.ID, fmt.Sprintf("Unable to lookup user %v, error unmarshalling response %v", arg, err.Error()),
			update.Message.MessageID)
		return
	}

	e := dat["email"].(string)
	if len(e) == 0 {
		s.service.SendMessage(context.TODO(), update.Message.Chat.ID, fmt.Sprintf("Unable to lookup user %v,no email address found", arg),
			update.Message.MessageID)
		return
	}

	token := rand.Intn(99999)
	s.store.storeNewToken(strconv.Itoa(update.Message.From.ID), arg, e, strconv.Itoa(token))

	conn, err := net.Dial("tcp", getSMTPServer()+":25")
	if err != nil {
		log.Panic(err)
	}

	c, err := smtp.NewClient(conn, getSMTPServer())
	if err != nil {
		s.service.SendMessage(context.TODO(), update.Message.Chat.ID, fmt.Sprintf("We were unable to create a SMTP connection to send you a token. %v", err.Error()),
			update.Message.MessageID)
		return
	}

	if err = c.Mail(getFromEmailAddress()); err != nil {
		s.service.SendMessage(context.TODO(), update.Message.Chat.ID, fmt.Sprintf("We were unable to communicate with the SMTP server. %v", err.Error()),
			update.Message.MessageID)
		return
	}

	if err = c.Rcpt(e); err != nil {
		s.service.SendMessage(context.TODO(), update.Message.Chat.ID, fmt.Sprintf("We were unable to communicate with the SMTP server. %v", err.Error()),
			update.Message.MessageID)
		return
	}

	w, err := c.Data()
	if err != nil {
		s.service.SendMessage(context.TODO(), update.Message.Chat.ID, fmt.Sprintf("We were unable to communicate with the SMTP server. %v", err.Error()),
			update.Message.MessageID)
		return
	}

	_, err = w.Write([]byte(
		fmt.Sprintf("From %v,\r\nTo:%v\r\nSubject:Hal Authentication Token\r\n\r\nYour HAL registration token is %v", getFromEmailAddress(), e, token),
	))
	if err != nil {
		s.service.SendMessage(context.TODO(), update.Message.Chat.ID, fmt.Sprintf("We were unable to communicate with the SMTP server. %v", err.Error()),
			update.Message.MessageID)
		return
	}

	err = w.Close()
	if err != nil {
		s.service.SendMessage(context.TODO(), update.Message.Chat.ID, fmt.Sprintf("We were unable to communicate with the SMTP server. %v", err.Error()),
			update.Message.MessageID)
		return
	}

	c.Quit()
	s.service.SendMessage(context.TODO(), update.Message.Chat.ID, fmt.Sprintf("Registration token has been sent to %v", e),
		update.Message.MessageID)
}

type token struct {
	telegram.Service
	Store
}

func NewTokenCommand(service telegram.Service, store Store) telegram.Command {
	return &token{service, store}
}

func (token) CommandIdentifier() string {
	return "Token"
}

func (token) CommandDescription() string {
	return "Complete the registration process by providing the token you recieved via email. "
}

func (s token) Execute(update tgbotapi.Update) {
	arg := update.Message.CommandArguments()
	if len(arg) == 0 {
		s.SendMessage(context.TODO(), update.Message.Chat.ID, "To complete your registration please enter /Token <token> - the token you would have received by using /Register <employee number>", update.Message.MessageID)
		return
	}

	token, err := s.getTokenForUser(strconv.Itoa(update.Message.From.ID))
	if err != nil {
		s.SendMessage(context.TODO(), update.Message.Chat.ID, fmt.Sprintf("There was a problem fetching your registration record. To start the registration process please use /Register <employee number>. Error was %v",err.Error()), update.Message.MessageID)
		return
	}

	if token != arg {
		s.SendMessage(context.TODO(), update.Message.Chat.ID, "The token provided does not match the one we have on record. To get a new token use /Register <employee number>", update.Message.MessageID)
		return
	}

	err = s.authorizeUser(strconv.Itoa(update.Message.From.ID))

	if err != nil {
		s.SendMessage(context.TODO(), update.Message.Chat.ID, fmt.Sprintf("Although your token was correct, there was a problem activating you. Error was %v",err.Error()), update.Message.MessageID)
		return
	}
	s.SendMessage(context.TODO(), update.Message.Chat.ID,"You have been successfully authorised", update.Message.MessageID)
}

func getLdapEndpoint() string {
	return os.Getenv("BANK_LDAP_ENDPOINT")
}

func getSMTPServer() string {
	return os.Getenv("SMTP_SERVER")
}

func getFromEmailAddress() string {
	return os.Getenv("SMTP_FROM_ADDRESS")
}
