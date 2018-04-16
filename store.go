package bankldapService

import (
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type Store interface {
	storeNewToken(userid string, employeeNumber string, email string, token string)
	getTokenForUser(userid string) (string, error)
	authorizeUser(userid string) error
	isAuthorized(userid string) bool
}

type user struct {
	TelegramId string
	EmployeeNumber string
	Email string
	Token string
	Authorised bool
}

type mongoDb struct {
	db *mgo.Database
}

func NewMongoStore(db *mgo.Database)  Store {
	return &mongoDb{db:db}
}

func (s mongoDb) isAuthorized(userid string) bool {
	c:= s.db.C("BANK_USERS")
	u := &user{}
	err := c.Find(bson.M{"telegramid":userid}).One(&u)
	if err != nil {
		return false
	}

	return u.Authorised
}

func (s mongoDb)storeNewToken(userid string, employeeNumber string, email string, token string){
	c:= s.db.C("BANK_USERS")
	u := &user{}
	err := c.Find(bson.M{"telegramid":userid}).One(&u)
	if err != nil {
		u = &user{TelegramId:userid,Email:email,Authorised:false,Token:token, EmployeeNumber:employeeNumber}
		c.Insert(&u)
	} else {
		u.Token = token
		c.Update(bson.M{"telegramid":userid},&u)
	}
}

func (s mongoDb)getTokenForUser(userid string) (string, error){
	c:= s.db.C("BANK_USERS")
	u := &user{}
	err := c.Find(bson.M{"telegramid":userid}).One(&u)
	if err != nil {
		return "", err
	}
	return u.Token,nil

}
func (s mongoDb)authorizeUser(userid string) error{
	c:= s.db.C("BANK_USERS")
	u := &user{}
	err := c.Find(bson.M{"telegramid":userid}).One(&u)
	if err != nil {
		return err
	}
	u.Authorised = true
	c.Update(bson.M{"telegramid":userid},&u)
	return nil
}

