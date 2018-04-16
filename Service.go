package bankldapService

import "github.com/weAutomateEverything/go2hal/auth"

type bankldap struct {
	store Store
}

func NewService(store Store) auth.Service{
	return &bankldap{
		store:store,
	}
}

func (s *bankldap) Authorize(user string) bool {
	return s.store.isAuthorized(user)
}

