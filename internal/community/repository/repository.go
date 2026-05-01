package repository

import legacycommunity "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/community"

type Store struct {
	Communities legacycommunity.CommunityRepo
}

func NewStore(communities legacycommunity.CommunityRepo) Store {
	return Store{Communities: communities}
}
